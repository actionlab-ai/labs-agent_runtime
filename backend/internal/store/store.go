package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"novel-agent-runtime/internal/db/sqlc"
	"novel-agent-runtime/internal/project"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

type Project struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ProjectDocument struct {
	ID        int64           `json:"id"`
	ProjectID string          `json:"project_id"`
	Kind      string          `json:"kind"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ModelProfile struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id"`
	BaseURL         string          `json:"base_url"`
	APIKey          string          `json:"-"`
	APIKeySet       bool            `json:"api_key_set"`
	APIKeyEnv       string          `json:"-"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Run struct {
	ID        int64           `json:"id"`
	ProjectID string          `json:"project_id,omitempty"`
	SessionID int64           `json:"session_id,omitempty"`
	Input     string          `json:"input"`
	FinalText string          `json:"final_text"`
	RunDir    string          `json:"run_dir"`
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateProjectParams struct {
	Name        string          `json:"name"`
	ID          string          `json:"id,omitempty"`
	Description string          `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type UpdateProjectParams struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type UpsertProjectDocumentParams struct {
	ProjectID string          `json:"project_id"`
	Kind      string          `json:"kind"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type CreateModelProfileParams struct {
	ID              string          `json:"id,omitempty"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider,omitempty"`
	ModelID         string          `json:"model_id"`
	BaseURL         string          `json:"base_url"`
	APIKey          string          `json:"api_key,omitempty"`
	APIKeyEnv       string          `json:"api_key_env,omitempty"`
	ContextWindow   int             `json:"context_window,omitempty"`
	MaxOutputTokens int             `json:"max_output_tokens,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	TimeoutSeconds  int             `json:"timeout_seconds,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

type UpdateModelProfileParams struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id"`
	BaseURL         string          `json:"base_url"`
	APIKey          string          `json:"api_key,omitempty"`
	APIKeyEnv       string          `json:"api_key_env"`
	ContextWindow   int             `json:"context_window"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Temperature     float64         `json:"temperature"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, queries: dbsqlc.New(pool)}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func Migrate(databaseURL, migrationsDir string) error {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return fmt.Errorf("database url is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.ToSlash(migrationsDir), "postgres", driver)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (s *Store) CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error) {
	name := strings.TrimSpace(arg.Name)
	if name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	id := strings.TrimSpace(arg.ID)
	if id == "" {
		id = project.Slug(name)
	}
	p, err := s.queries.CreateProject(ctx, dbsqlc.CreateProjectParams{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(arg.Description),
		Metadata:    jsonOrEmpty(arg.Metadata),
	})
	if err != nil {
		return Project{}, err
	}
	return convertProject(p), nil
}

func (s *Store) GetProject(ctx context.Context, id string) (Project, error) {
	p, err := s.queries.GetProject(ctx, project.Slug(id))
	if err != nil {
		return Project{}, err
	}
	return convertProject(p), nil
}

func (s *Store) ListProjects(ctx context.Context, limit, offset int32) ([]Project, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.queries.ListProjects(ctx, dbsqlc.ListProjectsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]Project, 0, len(items))
	for _, item := range items {
		out = append(out, convertProject(item))
	}
	return out, nil
}

func (s *Store) UpdateProject(ctx context.Context, arg UpdateProjectParams) (Project, error) {
	status := strings.TrimSpace(arg.Status)
	if status == "" {
		status = "active"
	}
	p, err := s.queries.UpdateProject(ctx, dbsqlc.UpdateProjectParams{
		ID:          project.Slug(arg.ID),
		Name:        strings.TrimSpace(arg.Name),
		Description: strings.TrimSpace(arg.Description),
		Status:      status,
		Metadata:    jsonOrEmpty(arg.Metadata),
	})
	if err != nil {
		return Project{}, err
	}
	return convertProject(p), nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	return s.queries.DeleteProject(ctx, project.Slug(id))
}

func (s *Store) CreateModelProfile(ctx context.Context, arg CreateModelProfileParams) (ModelProfile, error) {
	normalized, err := normalizeCreateModelProfile(arg)
	if err != nil {
		return ModelProfile{}, err
	}
	item, err := s.queries.CreateModelProfile(ctx, dbsqlc.CreateModelProfileParams{
		ID:              normalized.ID,
		Name:            normalized.Name,
		Provider:        normalized.Provider,
		ModelID:         normalized.ModelID,
		BaseUrl:         normalized.BaseURL,
		ApiKey:          normalized.APIKey,
		ApiKeyEnv:       normalized.APIKeyEnv,
		ContextWindow:   int32(normalized.ContextWindow),
		MaxOutputTokens: int32(normalized.MaxOutputTokens),
		Temperature:     normalized.Temperature,
		TimeoutSeconds:  int32(normalized.TimeoutSeconds),
		Metadata:        jsonOrEmpty(normalized.Metadata),
	})
	if err != nil {
		return ModelProfile{}, err
	}
	return convertModelProfile(item), nil
}

func (s *Store) GetModelProfile(ctx context.Context, id string) (ModelProfile, error) {
	item, err := s.queries.GetModelProfile(ctx, project.Slug(id))
	if err != nil {
		return ModelProfile{}, err
	}
	return convertModelProfile(item), nil
}

func (s *Store) ListModelProfiles(ctx context.Context, limit, offset int32) ([]ModelProfile, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.queries.ListModelProfiles(ctx, dbsqlc.ListModelProfilesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]ModelProfile, 0, len(items))
	for _, item := range items {
		out = append(out, convertModelProfile(item))
	}
	return out, nil
}

func (s *Store) UpdateModelProfile(ctx context.Context, arg UpdateModelProfileParams) (ModelProfile, error) {
	normalized, err := normalizeUpdateModelProfile(arg)
	if err != nil {
		return ModelProfile{}, err
	}
	item, err := s.queries.UpdateModelProfile(ctx, dbsqlc.UpdateModelProfileParams{
		ID:              normalized.ID,
		Name:            normalized.Name,
		Provider:        normalized.Provider,
		ModelID:         normalized.ModelID,
		BaseUrl:         normalized.BaseURL,
		ApiKey:          normalized.APIKey,
		ApiKeyEnv:       normalized.APIKeyEnv,
		ContextWindow:   int32(normalized.ContextWindow),
		MaxOutputTokens: int32(normalized.MaxOutputTokens),
		Temperature:     normalized.Temperature,
		TimeoutSeconds:  int32(normalized.TimeoutSeconds),
		Status:          normalized.Status,
		Metadata:        jsonOrEmpty(normalized.Metadata),
	})
	if err != nil {
		return ModelProfile{}, err
	}
	return convertModelProfile(item), nil
}

func (s *Store) DeleteModelProfile(ctx context.Context, id string) error {
	return s.queries.DeleteModelProfile(ctx, project.Slug(id))
}

func (s *Store) UpsertProjectDocument(ctx context.Context, arg UpsertProjectDocumentParams) (ProjectDocument, error) {
	doc, err := s.queries.UpsertProjectDocument(ctx, dbsqlc.UpsertProjectDocumentParams{
		ProjectID: project.Slug(arg.ProjectID),
		Kind:      strings.TrimSpace(arg.Kind),
		Title:     strings.TrimSpace(arg.Title),
		Body:      arg.Body,
		Metadata:  jsonOrEmpty(arg.Metadata),
	})
	if err != nil {
		return ProjectDocument{}, err
	}
	return convertProjectDocument(doc), nil
}

func (s *Store) ListProjectDocuments(ctx context.Context, projectID string) ([]ProjectDocument, error) {
	items, err := s.queries.ListProjectDocuments(ctx, project.Slug(projectID))
	if err != nil {
		return nil, err
	}
	out := make([]ProjectDocument, 0, len(items))
	for _, item := range items {
		out = append(out, convertProjectDocument(item))
	}
	return out, nil
}

func (s *Store) CreateRun(ctx context.Context, projectID string, input string, metadata json.RawMessage) (Run, error) {
	run, err := s.queries.CreateRun(ctx, dbsqlc.CreateRunParams{
		ProjectID: nullableText(optionalProjectID(projectID)),
		SessionID: pgtype.Int8{},
		Input:     input,
		Metadata:  jsonOrEmpty(metadata),
	})
	if err != nil {
		return Run{}, err
	}
	return convertRun(run), nil
}

func (s *Store) FinishRun(ctx context.Context, id int64, finalText, runDir string) (Run, error) {
	run, err := s.queries.FinishRun(ctx, dbsqlc.FinishRunParams{
		ID:        id,
		FinalText: finalText,
		RunDir:    runDir,
	})
	if err != nil {
		return Run{}, err
	}
	return convertRun(run), nil
}

func (s *Store) FailRun(ctx context.Context, id int64, message string) (Run, error) {
	run, err := s.queries.FailRun(ctx, dbsqlc.FailRunParams{ID: id, Error: message})
	if err != nil {
		return Run{}, err
	}
	return convertRun(run), nil
}

func convertProject(p dbsqlc.Project) Project {
	return Project{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		Metadata:    normalizeJSON(p.Metadata),
		CreatedAt:   timeFromTimestamptz(p.CreatedAt),
		UpdatedAt:   timeFromTimestamptz(p.UpdatedAt),
	}
}

func convertModelProfile(item dbsqlc.ModelProfile) ModelProfile {
	return ModelProfile{
		ID:              item.ID,
		Name:            item.Name,
		Provider:        item.Provider,
		ModelID:         item.ModelID,
		BaseURL:         item.BaseUrl,
		APIKey:          item.ApiKey,
		APIKeySet:       strings.TrimSpace(item.ApiKey) != "",
		APIKeyEnv:       item.ApiKeyEnv,
		ContextWindow:   int(item.ContextWindow),
		MaxOutputTokens: int(item.MaxOutputTokens),
		Temperature:     item.Temperature,
		TimeoutSeconds:  int(item.TimeoutSeconds),
		Status:          item.Status,
		Metadata:        normalizeJSON(item.Metadata),
		CreatedAt:       timeFromTimestamptz(item.CreatedAt),
		UpdatedAt:       timeFromTimestamptz(item.UpdatedAt),
	}
}

func convertProjectDocument(doc dbsqlc.ProjectDocument) ProjectDocument {
	return ProjectDocument{
		ID:        doc.ID,
		ProjectID: doc.ProjectID,
		Kind:      doc.Kind,
		Title:     doc.Title,
		Body:      doc.Body,
		Metadata:  normalizeJSON(doc.Metadata),
		CreatedAt: timeFromTimestamptz(doc.CreatedAt),
		UpdatedAt: timeFromTimestamptz(doc.UpdatedAt),
	}
}

func convertRun(run dbsqlc.Run) Run {
	return Run{
		ID:        run.ID,
		ProjectID: textFromPg(run.ProjectID),
		SessionID: int64FromPg(run.SessionID),
		Input:     run.Input,
		FinalText: run.FinalText,
		RunDir:    run.RunDir,
		Status:    run.Status,
		Error:     run.Error,
		Metadata:  normalizeJSON(run.Metadata),
		CreatedAt: timeFromTimestamptz(run.CreatedAt),
		UpdatedAt: timeFromTimestamptz(run.UpdatedAt),
	}
}

func jsonOrEmpty(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func normalizeJSON(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func nullableText(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func optionalProjectID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return project.Slug(id)
}

func textFromPg(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func int64FromPg(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func timeFromTimestamptz(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func normalizeCreateModelProfile(arg CreateModelProfileParams) (CreateModelProfileParams, error) {
	arg.Name = strings.TrimSpace(arg.Name)
	arg.ID = strings.TrimSpace(arg.ID)
	arg.Provider = firstNonEmpty(arg.Provider, "openai_compatible")
	arg.ModelID = strings.TrimSpace(arg.ModelID)
	arg.BaseURL = strings.TrimRight(strings.TrimSpace(arg.BaseURL), "/")
	arg.APIKey = strings.TrimSpace(arg.APIKey)
	arg.APIKeyEnv = strings.TrimSpace(arg.APIKeyEnv)
	if arg.ID == "" {
		arg.ID = project.Slug(firstNonEmpty(arg.Name, arg.ModelID))
	}
	if arg.Name == "" {
		arg.Name = arg.ID
	}
	if arg.ModelID == "" {
		return arg, fmt.Errorf("model_id is required")
	}
	if arg.BaseURL == "" {
		return arg, fmt.Errorf("base_url is required")
	}
	if arg.ContextWindow <= 0 {
		arg.ContextWindow = 131072
	}
	if arg.MaxOutputTokens <= 0 {
		arg.MaxOutputTokens = 4096
	}
	if arg.Temperature == 0 {
		arg.Temperature = 0.7
	}
	if arg.TimeoutSeconds <= 0 {
		arg.TimeoutSeconds = 180
	}
	return arg, nil
}

func normalizeUpdateModelProfile(arg UpdateModelProfileParams) (UpdateModelProfileParams, error) {
	arg.ID = project.Slug(arg.ID)
	arg.Name = strings.TrimSpace(arg.Name)
	arg.Provider = firstNonEmpty(arg.Provider, "openai_compatible")
	arg.ModelID = strings.TrimSpace(arg.ModelID)
	arg.BaseURL = strings.TrimRight(strings.TrimSpace(arg.BaseURL), "/")
	arg.APIKey = strings.TrimSpace(arg.APIKey)
	arg.APIKeyEnv = strings.TrimSpace(arg.APIKeyEnv)
	arg.Status = firstNonEmpty(arg.Status, "active")
	if arg.Name == "" {
		arg.Name = arg.ID
	}
	if arg.ModelID == "" {
		return arg, fmt.Errorf("model_id is required")
	}
	if arg.BaseURL == "" {
		return arg, fmt.Errorf("base_url is required")
	}
	if arg.ContextWindow <= 0 {
		arg.ContextWindow = 131072
	}
	if arg.MaxOutputTokens <= 0 {
		arg.MaxOutputTokens = 4096
	}
	if arg.Temperature == 0 {
		arg.Temperature = 0.7
	}
	if arg.TimeoutSeconds <= 0 {
		arg.TimeoutSeconds = 180
	}
	return arg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
