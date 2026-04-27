package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

type fakeStore struct {
	projects map[string]store.Project
	docs     map[string][]store.ProjectDocument
	models   map[string]store.ModelProfile
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		projects: map[string]store.Project{},
		docs:     map[string][]store.ProjectDocument{},
		models:   map[string]store.ModelProfile{},
	}
}

func (f *fakeStore) CreateProject(_ context.Context, arg store.CreateProjectParams) (store.Project, error) {
	id := arg.ID
	if id == "" {
		id = project.Slug(arg.Name)
	}
	p := store.Project{ID: id, Name: arg.Name, Description: arg.Description, Status: "active"}
	f.projects[id] = p
	return p, nil
}

func (f *fakeStore) GetProject(_ context.Context, id string) (store.Project, error) {
	return f.projects[project.Slug(id)], nil
}

func (f *fakeStore) ListProjects(_ context.Context, _, _ int32) ([]store.Project, error) {
	out := make([]store.Project, 0, len(f.projects))
	for _, p := range f.projects {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStore) UpdateProject(_ context.Context, arg store.UpdateProjectParams) (store.Project, error) {
	p := store.Project{ID: project.Slug(arg.ID), Name: arg.Name, Description: arg.Description, Status: arg.Status}
	f.projects[p.ID] = p
	return p, nil
}

func (f *fakeStore) DeleteProject(_ context.Context, id string) error {
	delete(f.projects, project.Slug(id))
	return nil
}

func (f *fakeStore) CreateModelProfile(_ context.Context, arg store.CreateModelProfileParams) (store.ModelProfile, error) {
	id := arg.ID
	if id == "" {
		id = project.Slug(arg.Name)
	}
	item := store.ModelProfile{
		ID:              id,
		Name:            arg.Name,
		Provider:        "openai_compatible",
		ModelID:         arg.ModelID,
		BaseURL:         arg.BaseURL,
		APIKey:          arg.APIKey,
		APIKeySet:       arg.APIKey != "",
		APIKeyEnv:       arg.APIKeyEnv,
		ContextWindow:   arg.ContextWindow,
		MaxOutputTokens: arg.MaxOutputTokens,
		Temperature:     arg.Temperature,
		TimeoutSeconds:  arg.TimeoutSeconds,
		Status:          "active",
	}
	f.models[id] = item
	return item, nil
}

func (f *fakeStore) GetModelProfile(_ context.Context, id string) (store.ModelProfile, error) {
	return f.models[project.Slug(id)], nil
}

func (f *fakeStore) ListModelProfiles(_ context.Context, _, _ int32) ([]store.ModelProfile, error) {
	out := make([]store.ModelProfile, 0, len(f.models))
	for _, item := range f.models {
		out = append(out, item)
	}
	return out, nil
}

func (f *fakeStore) UpdateModelProfile(_ context.Context, arg store.UpdateModelProfileParams) (store.ModelProfile, error) {
	item := store.ModelProfile{
		ID:              project.Slug(arg.ID),
		Name:            arg.Name,
		Provider:        arg.Provider,
		ModelID:         arg.ModelID,
		BaseURL:         arg.BaseURL,
		APIKey:          arg.APIKey,
		APIKeySet:       arg.APIKey != "",
		APIKeyEnv:       arg.APIKeyEnv,
		ContextWindow:   arg.ContextWindow,
		MaxOutputTokens: arg.MaxOutputTokens,
		Temperature:     arg.Temperature,
		TimeoutSeconds:  arg.TimeoutSeconds,
		Status:          arg.Status,
	}
	f.models[item.ID] = item
	return item, nil
}

func (f *fakeStore) DeleteModelProfile(_ context.Context, id string) error {
	delete(f.models, project.Slug(id))
	return nil
}

func (f *fakeStore) UpsertProjectDocument(_ context.Context, arg store.UpsertProjectDocumentParams) (store.ProjectDocument, error) {
	doc := store.ProjectDocument{ID: int64(len(f.docs[arg.ProjectID]) + 1), ProjectID: arg.ProjectID, Kind: arg.Kind, Title: arg.Title, Body: arg.Body}
	f.docs[arg.ProjectID] = append(f.docs[arg.ProjectID], doc)
	return doc, nil
}

func (f *fakeStore) ListProjectDocuments(_ context.Context, projectID string) ([]store.ProjectDocument, error) {
	return f.docs[project.Slug(projectID)], nil
}

func (f *fakeStore) CreateRun(_ context.Context, projectID string, input string, _ json.RawMessage) (store.Run, error) {
	return store.Run{ID: 1, ProjectID: project.Slug(projectID), Input: input, Status: "running"}, nil
}

func (f *fakeStore) FinishRun(_ context.Context, id int64, finalText, runDir string) (store.Run, error) {
	return store.Run{ID: id, FinalText: finalText, RunDir: runDir, Status: "completed"}, nil
}

func (f *fakeStore) FailRun(_ context.Context, id int64, message string) (store.Run, error) {
	return store.Run{ID: id, Status: "failed", Error: message}, nil
}

func TestHTTPCreateProjectAndDryRun(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	db := newFakeStore()
	router := buildHTTPRouter(cfg, false, db)

	projectBody := bytes.NewBufferString(`{"name":"都市异能悬疑"}`)
	projectReq := httptest.NewRequest(http.MethodPost, "/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	projectRec := httptest.NewRecorder()
	router.ServeHTTP(projectRec, projectReq)
	if projectRec.Code != http.StatusOK {
		t.Fatalf("expected project create 200, got %d: %s", projectRec.Code, projectRec.Body.String())
	}

	var projectPayload struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	}
	if err := json.Unmarshal(projectRec.Body.Bytes(), &projectPayload); err != nil {
		t.Fatalf("decode project payload failed: %v", err)
	}
	if projectPayload.Project.ID != "都市异能悬疑" {
		t.Fatalf("unexpected project id: %q", projectPayload.Project.ID)
	}

	runBody := bytes.NewBufferString(`{"project":"都市异能悬疑","input":"先做起盘","dry_run":true}`)
	runReq := httptest.NewRequest(http.MethodPost, "/v1/runs", runBody)
	runReq.Header.Set("Content-Type", "application/json")
	runRec := httptest.NewRecorder()
	router.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected dry run 200, got %d: %s", runRec.Code, runRec.Body.String())
	}

	var runPayload struct {
		DryRun  bool   `json:"dry_run"`
		RunID   string `json:"run_id"`
		RunDir  string `json:"run_dir"`
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	}
	if err := json.Unmarshal(runRec.Body.Bytes(), &runPayload); err != nil {
		t.Fatalf("decode run payload failed: %v", err)
	}
	if !runPayload.DryRun || runPayload.RunID == "" || runPayload.RunDir == "" {
		t.Fatalf("expected dry-run metadata, got %#v", runPayload)
	}
	if runPayload.Project.ID != "都市异能悬疑" {
		t.Fatalf("expected active project in run response, got %#v", runPayload.Project)
	}
}

func TestHTTPSkills(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	router := buildHTTPRouter(cfg, false, newFakeStore())

	req := httptest.NewRequest(http.MethodGet, "/v1/skills", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected skills 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Test Skill")) {
		t.Fatalf("expected response to include test skill, got %s", rec.Body.String())
	}
}

func TestHTTPModelCRUDAndDryRunSelection(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	db := newFakeStore()
	router := buildHTTPRouter(cfg, false, db)

	body := bytes.NewBufferString(`{"id":"deepseek-flash","name":"DeepSeek Flash","model_id":"deepseek-v4-flash","base_url":"https://api.deepseek.com","api_key":"sk-test","max_output_tokens":8192}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/models", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected model create 200, got %d: %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected model list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if !bytes.Contains(listRec.Body.Bytes(), []byte("deepseek-flash")) {
		t.Fatalf("expected model list to include profile, got %s", listRec.Body.String())
	}
	if bytes.Contains(listRec.Body.Bytes(), []byte("sk-test")) {
		t.Fatalf("expected model list to hide raw api key, got %s", listRec.Body.String())
	}
	if bytes.Contains(listRec.Body.Bytes(), []byte("api_key_env")) {
		t.Fatalf("expected model list to hide api_key_env compatibility field, got %s", listRec.Body.String())
	}
	if !bytes.Contains(listRec.Body.Bytes(), []byte("api_key_set")) {
		t.Fatalf("expected model list to include api_key_set, got %s", listRec.Body.String())
	}

	runBody := bytes.NewBufferString(`{"model":"deepseek-flash","input":"先做起盘","dry_run":true}`)
	runReq := httptest.NewRequest(http.MethodPost, "/v1/runs", runBody)
	runReq.Header.Set("Content-Type", "application/json")
	runRec := httptest.NewRecorder()
	router.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected dry run with model 200, got %d: %s", runRec.Code, runRec.Body.String())
	}
	if !bytes.Contains(runRec.Body.Bytes(), []byte("deepseek-flash")) {
		t.Fatalf("expected dry run response to include active model, got %s", runRec.Body.String())
	}
}

func newHTTPTestConfig(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	skillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: Test Skill
description: Test skill
when_to_use: Use for tests
tags:
  - test
user_invocable: true
---
Body
`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return config.Config{
		Model: config.RuntimeModel{
			ID:             "test-model",
			BaseURL:        "http://example.invalid",
			APIKeyEnv:      "TEST_API_KEY",
			TimeoutSeconds: 1,
		},
		Runtime: config.RuntimeConfig{
			SkillsDir:            skillsDir,
			RunsDir:              filepath.Join(root, "runs"),
			WorkspaceRoot:        root,
			DocumentOutputDir:    filepath.Join(root, "docs", "generated"),
			MaxToolRounds:        4,
			MaxSkillToolRounds:   6,
			ForceToolSearchFirst: true,
			ReturnSkillDirect:    true,
			FallbackSkillSearch:  true,
			FallbackMinScore:     0.18,
			MaxActivatedSkills:   3,
			MaxRetainedSkills:    6,
			ActivationMinScore:   0.18,
			ActivationScoreRatio: 0.55,
		},
	}
}
