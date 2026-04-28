package workflow

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"novel-agent-runtime/internal/store"
)

func TestLocalSkillProviderListAndLoad(t *testing.T) {
	skillsDir := t.TempDir()
	skillDir := filepath.Join(skillsDir, "idea-bootstrap")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: Idea Bootstrap
description: Build novel project seed docs
when_to_use: Use when initializing worldview and character baseline
version: v1
tags:
  - novel
tool_output_contract: bootstrap_v1
tool_input_schema:
  type: object
  properties:
    premise:
      type: string
---
Body used only at invocation time.
`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	provider := LocalSkillProvider{SkillsDir: skillsDir}
	skills, err := provider.ListSkills(context.Background())
	if err != nil {
		t.Fatalf("ListSkills failed: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected one skill, got %#v", skills)
	}
	if skills[0].ID != "idea-bootstrap" || skills[0].Name != "Idea Bootstrap" {
		t.Fatalf("unexpected skill spec: %#v", skills[0])
	}
	if skills[0].InputSchema["type"] != "object" {
		t.Fatalf("expected input schema to be exposed, got %#v", skills[0].InputSchema)
	}

	def, err := provider.LoadSkill(context.Background(), "idea-bootstrap")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}
	if !strings.Contains(def.MarkdownContent, "Body used only at invocation time.") {
		t.Fatalf("expected full skill body on load, got %q", def.MarkdownContent)
	}
}

func TestPostgresContextProviderBuildsProjectContextPack(t *testing.T) {
	store := fakeProjectStore{
		project: store.Project{
			ID:          "urban-case",
			Name:        "Urban Case",
			Description: "Forensic urban power story",
			Status:      "active",
		},
		docs: []store.ProjectDocument{
			{ProjectID: "urban-case", Kind: "world_rules", Title: "World Rules", Body: "Death echoes last three minutes."},
			{ProjectID: "urban-case", Kind: "current_state", Title: "Current State", Body: "The protagonist has one sealed report."},
		},
	}

	provider := PostgresContextProvider{Store: store}
	pack, err := provider.BuildContext(context.Background(), "urban-case", "continue chapter")
	if err != nil {
		t.Fatalf("BuildContext failed: %v", err)
	}
	if pack.Project.ID != "urban-case" || pack.Project.Name != "Urban Case" {
		t.Fatalf("unexpected project context: %#v", pack.Project)
	}
	if pack.Request != "continue chapter" {
		t.Fatalf("expected request to be recorded, got %q", pack.Request)
	}
	if len(pack.Documents) != 2 {
		t.Fatalf("expected documents in pack, got %#v", pack.Documents)
	}
	for _, want := range []string{"World Rules", "Death echoes last three minutes.", "Current State"} {
		if !strings.Contains(pack.Text, want) {
			t.Fatalf("expected context text to contain %q, got %s", want, pack.Text)
		}
	}
}

func TestSequentialWorkflowRunnerRunsStepsInOrder(t *testing.T) {
	runner := &recordingSkillRunner{}
	workflow := SequentialWorkflowRunner{SkillRunner: runner}
	out, err := workflow.RunWorkflow(context.Background(), WorkflowInput{
		WorkflowID: "bootstrap-flow",
		Request:    "build world bible",
		ProjectID:  "urban-case",
		Context:    ContextPack{ProjectID: "urban-case", Text: "canon"},
		Arguments:  map[string]any{"shared": "base", "override": "base"},
		Steps: []WorkflowStep{
			{ID: "world", SkillID: "worldbuilding", Arguments: map[string]any{"override": "step"}},
			{ID: "cast", SkillID: "character-cast"},
		},
	})
	if err != nil {
		t.Fatalf("RunWorkflow failed: %v", err)
	}
	if len(out.Steps) != 2 {
		t.Fatalf("expected two step outputs, got %#v", out.Steps)
	}
	if got, want := runner.skillIDs, []string{"worldbuilding", "character-cast"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected skill order: got %#v want %#v", got, want)
	}
	if runner.inputs[0].Arguments["shared"] != "base" || runner.inputs[0].Arguments["override"] != "step" {
		t.Fatalf("expected merged step arguments, got %#v", runner.inputs[0].Arguments)
	}
	if runner.inputs[1].Arguments["override"] != "base" {
		t.Fatalf("expected base arguments on second step, got %#v", runner.inputs[1].Arguments)
	}
	if out.StartedAt.IsZero() || out.FinishedAt.IsZero() {
		t.Fatalf("expected workflow timestamps, got %#v", out)
	}
}

type fakeProjectStore struct {
	project store.Project
	docs    []store.ProjectDocument
}

func (s fakeProjectStore) GetProject(context.Context, string) (store.Project, error) {
	return s.project, nil
}

func (s fakeProjectStore) ListProjectDocuments(context.Context, string) ([]store.ProjectDocument, error) {
	return s.docs, nil
}

type recordingSkillRunner struct {
	inputs   []SkillInput
	skillIDs []string
}

func (r *recordingSkillRunner) RunSkill(_ context.Context, input SkillInput) (SkillOutput, error) {
	r.inputs = append(r.inputs, input)
	r.skillIDs = append(r.skillIDs, input.SkillID)
	return SkillOutput{SkillID: input.SkillID, Text: "ok:" + input.SkillID}, nil
}
