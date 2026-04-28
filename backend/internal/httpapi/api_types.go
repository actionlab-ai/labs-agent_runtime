package httpapi

import (
	"context"
	"encoding/json"

	"novel-agent-runtime/internal/store"
)

type appStore interface {
	CreateProject(context.Context, store.CreateProjectParams) (store.Project, error)
	GetProject(context.Context, string) (store.Project, error)
	ListProjects(context.Context, int32, int32) ([]store.Project, error)
	UpdateProject(context.Context, store.UpdateProjectParams) (store.Project, error)
	DeleteProject(context.Context, string) error
	CreateModelProfile(context.Context, store.CreateModelProfileParams) (store.ModelProfile, error)
	GetModelProfile(context.Context, string) (store.ModelProfile, error)
	ListModelProfiles(context.Context, int32, int32) ([]store.ModelProfile, error)
	UpdateModelProfile(context.Context, store.UpdateModelProfileParams) (store.ModelProfile, error)
	DeleteModelProfile(context.Context, string) error
	GetDefaultModelID(context.Context) (string, error)
	SetDefaultModelID(context.Context, string) error
	ClearDefaultModelID(context.Context) error
	UpsertProjectDocument(context.Context, store.UpsertProjectDocumentParams) (store.ProjectDocument, error)
	ListProjectDocuments(context.Context, string) ([]store.ProjectDocument, error)
	CreateRun(context.Context, string, string, json.RawMessage) (store.Run, error)
	FinishRun(context.Context, int64, string, string) (store.Run, error)
	FailRun(context.Context, int64, string) (store.Run, error)
}

type projectCreateRequest struct {
	ID              string          `json:"id"`
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	StorageProvider string          `json:"storage_provider"`
	StorageBucket   string          `json:"storage_bucket"`
	StoragePrefix   string          `json:"storage_prefix"`
	Metadata        json.RawMessage `json:"metadata"`
}

type projectUpdateRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	Status          string          `json:"status"`
	StorageProvider string          `json:"storage_provider"`
	StorageBucket   string          `json:"storage_bucket"`
	StoragePrefix   string          `json:"storage_prefix"`
	Metadata        json.RawMessage `json:"metadata"`
}

type projectDocumentUpsertRequest struct {
	Kind     string          `json:"kind"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Metadata json.RawMessage `json:"metadata"`
}

type runRequest struct {
	Input   string `json:"input" binding:"required"`
	Project string `json:"project"`
	Model   string `json:"model"`
	DryRun  bool   `json:"dry_run"`
	Debug   *bool  `json:"debug"`
}

type modelProfileCreateRequest struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id" binding:"required"`
	BaseURL         string          `json:"base_url" binding:"required"`
	APIKey          string          `json:"api_key"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Metadata        json.RawMessage `json:"metadata"`
}

type modelProfileUpdateRequest struct {
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id" binding:"required"`
	BaseURL         string          `json:"base_url" binding:"required"`
	APIKey          string          `json:"api_key"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
}

type defaultModelUpdateRequest struct {
	Model string `json:"model" binding:"required"`
}
