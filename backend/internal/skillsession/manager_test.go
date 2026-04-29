package skillsession

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/model"
	"novel-agent-runtime/internal/runstore"
	"novel-agent-runtime/internal/runtime"
	"novel-agent-runtime/internal/skill"
)

func TestManagerStartsAndContinuesAskHumanSkillSession(t *testing.T) {
	rt := newSessionTestRuntime(t)
	t.Setenv("TEST_API_KEY", "dummy")

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := requestCount.Add(1)
		var req model.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch current {
		case 1:
			_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"Need one answer","tool_calls":[{"id":"ask_1","type":"function","function":{"name":"AskHuman","arguments":"{\"reason\":\"missing payoff\",\"questions\":[{\"field\":\"payoff\",\"header\":\"Payoff\",\"question\":\"What payoff should the story promise?\",\"options\":[{\"label\":\"Dignity\",\"description\":\"Regain respect\"},{\"label\":\"Wealth\",\"description\":\"Get rich\"}]}]}"}}]},"finish_reason":"tool_calls"}]}`)
		case 2:
			if !strings.Contains(req.Messages[len(req.Messages)-1].Content, "Dignity") {
				t.Fatalf("expected resumed request to include human answer, got %#v", req.Messages[len(req.Messages)-1])
			}
			_, _ = fmt.Fprint(w, `{"id":"resp-2","choices":[{"index":0,"message":{"role":"assistant","content":"# Completed\n\nDignity payoff selected."},"finish_reason":"stop"}]}`)
		default:
			t.Fatalf("unexpected request #%d", current)
		}
	}))
	defer server.Close()

	rt.ModelConfig.BaseURL = server.URL
	rt.Model = model.NewOpenAICompatible(server.URL, "TEST_API_KEY", "test-model", 5)

	manager := NewManager()
	first, err := manager.Start(context.Background(), rt, StartInput{
		ProjectID: "case-1",
		SkillID:   "ask-skill",
		Request:   "build kernel",
		Arguments: map[string]any{"task": "build kernel"},
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if first.Status != StatusNeedsInput || first.AskHuman == nil {
		t.Fatalf("expected needs_input snapshot, got %#v", first)
	}
	if first.AskHuman.Questions[0].Field != "payoff" {
		t.Fatalf("unexpected pending question: %#v", first.AskHuman)
	}

	second, err := manager.Continue(context.Background(), first.ID, ContinueInput{
		Input:   "choose dignity",
		Answers: map[string]string{"What payoff should the story promise?": "Dignity"},
	})
	if err != nil {
		t.Fatalf("Continue failed: %v", err)
	}
	if second.Status != StatusCompleted || !strings.Contains(second.FinalText, "Dignity payoff selected") {
		t.Fatalf("expected completed snapshot, got %#v", second)
	}
	if len(second.Turns) < 4 {
		t.Fatalf("expected session transcript turns, got %#v", second.Turns)
	}
}

func newSessionTestRuntime(t *testing.T) *runtime.Runtime {
	t.Helper()
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	skillDir := filepath.Join(skillsDir, "ask-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: Ask Skill
description: Needs human clarification
allowed_tools:
  - Read
---
Ask for missing facts before writing.
`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	reg, err := skill.LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	rs, err := runstore.New(filepath.Join(root, "runs"))
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	return &runtime.Runtime{
		Config: config.Config{Runtime: config.RuntimeConfig{
			SkillsDir:          skillsDir,
			RunsDir:            filepath.Join(root, "runs"),
			WorkspaceRoot:      root,
			DocumentOutputDir:  filepath.Join(root, "projects"),
			MaxSkillToolRounds: 4,
		}},
		ModelConfig: runtime.ModelConfig{ID: "test-model", BaseURL: "http://example.invalid", APIKeyEnv: "TEST_API_KEY", MaxOutput: 4096, Temperature: 0.7, TimeoutSeconds: 5},
		Registry:    reg,
		Store:       rs,
	}
}
