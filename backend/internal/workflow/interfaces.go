package workflow

import (
	"context"
	"time"
)

type SkillSpec struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Version        string         `json:"version,omitempty"`
	Description    string         `json:"description,omitempty"`
	WhenToUse      string         `json:"when_to_use,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
	OutputContract string         `json:"output_contract,omitempty"`
	Source         string         `json:"source"`
}

type SkillDefinition struct {
	Spec            SkillSpec `json:"spec"`
	MarkdownContent string    `json:"markdown_content"`
	EntryPath       string    `json:"entry_path"`
	SkillRoot       string    `json:"skill_root"`
}

type SkillProvider interface {
	ListSkills(ctx context.Context) ([]SkillSpec, error)
	LoadSkill(ctx context.Context, id string) (SkillDefinition, error)
}

type ContextDocument struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ContextProject struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Status          string `json:"status,omitempty"`
	StorageProvider string `json:"storage_provider,omitempty"`
	StorageBucket   string `json:"storage_bucket,omitempty"`
	StoragePrefix   string `json:"storage_prefix,omitempty"`
}

type ContextPack struct {
	Project   ContextProject    `json:"project,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	Request   string            `json:"request,omitempty"`
	Documents []ContextDocument `json:"documents,omitempty"`
	Text      string            `json:"text"`
}

type ContextProvider interface {
	BuildContext(ctx context.Context, projectID string, request string) (ContextPack, error)
}

type SkillInput struct {
	SkillID       string         `json:"skill_id"`
	Request       string         `json:"request"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	ProjectID     string         `json:"project_id,omitempty"`
	Context       ContextPack    `json:"context,omitempty"`
	WorkflowRunID string         `json:"workflow_run_id,omitempty"`
}

type SkillOutput struct {
	SkillID string `json:"skill_id"`
	Text    string `json:"text"`
	RunID   string `json:"run_id,omitempty"`
	RunDir  string `json:"run_dir,omitempty"`
}

type SkillRunner interface {
	RunSkill(ctx context.Context, input SkillInput) (SkillOutput, error)
}

type WorkflowStep struct {
	ID        string         `json:"id"`
	SkillID   string         `json:"skill_id"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type WorkflowInput struct {
	WorkflowID string         `json:"workflow_id"`
	Request    string         `json:"request"`
	ProjectID  string         `json:"project_id,omitempty"`
	Context    ContextPack    `json:"context,omitempty"`
	Arguments  map[string]any `json:"arguments,omitempty"`
	Steps      []WorkflowStep `json:"steps"`
}

type WorkflowStepOutput struct {
	StepID string      `json:"step_id"`
	Output SkillOutput `json:"output"`
}

type WorkflowOutput struct {
	WorkflowID string               `json:"workflow_id"`
	StartedAt  time.Time            `json:"started_at"`
	FinishedAt time.Time            `json:"finished_at"`
	Steps      []WorkflowStepOutput `json:"steps"`
}

type WorkflowRunner interface {
	RunWorkflow(ctx context.Context, input WorkflowInput) (WorkflowOutput, error)
}
