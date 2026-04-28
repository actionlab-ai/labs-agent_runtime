package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

// fakeStore 是一个用于测试的内存存储实现，实现了 store.Store 接口。
// 它避免了在单元测试中依赖真实的数据库连接，提高了测试速度和隔离性。
type fakeStore struct {
	projects       map[string]store.Project
	docs           map[string][]store.ProjectDocument
	models         map[string]store.ModelProfile
	defaultModelID string
}

// newFakeStore 创建并初始化一个新的 fakeStore实例。
func newFakeStore() *fakeStore {
	return &fakeStore{
		projects: map[string]store.Project{},
		docs:     map[string][]store.ProjectDocument{},
		models:   map[string]store.ModelProfile{},
	}
}

// CreateProject 模拟创建项目操作。
// 如果参数中未提供 ID，则根据项目名称生成 slug 作为 ID。
func (f *fakeStore) CreateProject(_ context.Context, arg store.CreateProjectParams) (store.Project, error) {
	id := arg.ID
	if id == "" {
		id = project.Slug(arg.Name)
	}
	storageProvider := firstNonEmpty(arg.StorageProvider, "filesystem")
	storagePrefix := firstNonEmpty(arg.StoragePrefix, id)
	now := time.Now().UTC()
	p := store.Project{
		ID:              id,
		Name:            arg.Name,
		Description:     arg.Description,
		Status:          "active",
		StorageProvider: storageProvider,
		StorageBucket:   arg.StorageBucket,
		StoragePrefix:   storagePrefix,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	f.projects[id] = p
	return p, nil
}

// GetProject 模拟根据 ID 获取项目操作。
// 注意：这里同样使用 project.Slug 处理输入 ID，以保持一致性。
func (f *fakeStore) GetProject(_ context.Context, id string) (store.Project, error) {
	return f.projects[project.Slug(id)], nil
}

// ListProjects 模拟列出所有项目操作。
// 忽略分页参数，返回内存中存储的所有项目。
func (f *fakeStore) ListProjects(_ context.Context, _, _ int32) ([]store.Project, error) {
	out := make([]store.Project, 0, len(f.projects))
	for _, p := range f.projects {
		out = append(out, p)
	}
	return out, nil
}

// UpdateProject 模拟更新项目操作。
// 根据提供的参数更新内存中的项目信息。
func (f *fakeStore) UpdateProject(_ context.Context, arg store.UpdateProjectParams) (store.Project, error) {
	id := project.Slug(arg.ID)
	existing := f.projects[id]
	storageProvider := firstNonEmpty(arg.StorageProvider, existing.StorageProvider, "filesystem")
	storageBucket := firstNonEmpty(arg.StorageBucket, existing.StorageBucket)
	storagePrefix := firstNonEmpty(arg.StoragePrefix, existing.StoragePrefix, id)
	now := time.Now().UTC()
	p := store.Project{
		ID:              id,
		Name:            arg.Name,
		Description:     arg.Description,
		Status:          arg.Status,
		StorageProvider: storageProvider,
		StorageBucket:   storageBucket,
		StoragePrefix:   storagePrefix,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       now,
	}
	f.projects[p.ID] = p
	return p, nil
}

// DeleteProject 模拟删除项目操作。
// 从内存映射中移除指定 ID 的项目。
func (f *fakeStore) DeleteProject(_ context.Context, id string) error {
	delete(f.projects, project.Slug(id))
	return nil
}

// CreateModelProfile 模拟创建模型配置操作。
// 初始化模型配置文件，包括设置默认提供商和状态。
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

// GetModelProfile 模拟根据 ID 获取模型配置操作。
func (f *fakeStore) GetModelProfile(_ context.Context, id string) (store.ModelProfile, error) {
	return f.models[project.Slug(id)], nil
}

// ListModelProfiles 模拟列出所有模型配置操作。
// 返回内存中存储的所有模型配置。
func (f *fakeStore) ListModelProfiles(_ context.Context, _, _ int32) ([]store.ModelProfile, error) {
	out := make([]store.ModelProfile, 0, len(f.models))
	for _, item := range f.models {
		out = append(out, item)
	}
	return out, nil
}

// UpdateModelProfile 模拟更新模型配置操作。
// 根据提供的参数更新内存中的模型配置信息。
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

// DeleteModelProfile 模拟删除模型配置操作。
// 从内存映射中移除指定 ID 的模型配置。
func (f *fakeStore) DeleteModelProfile(_ context.Context, id string) error {
	delete(f.models, project.Slug(id))
	return nil
}

// GetDefaultModelID 模拟获取默认模型 ID 的操作。
func (f *fakeStore) GetDefaultModelID(_ context.Context) (string, error) {
	return f.defaultModelID, nil
}

// SetDefaultModelID 模拟设置默认模型 ID 的操作。
// 将传入的 modelID 转换为 slug 格式后存储。
func (f *fakeStore) SetDefaultModelID(_ context.Context, modelID string) error {
	f.defaultModelID = project.Slug(modelID)
	return nil
}

// ClearDefaultModelID 模拟清除默认模型 ID 的操作。
func (f *fakeStore) ClearDefaultModelID(_ context.Context) error {
	f.defaultModelID = ""
	return nil
}

// UpsertProjectDocument 模拟插入或更新项目文档操作。
// 简单地将文档追加到对应项目的文档列表中。
func (f *fakeStore) UpsertProjectDocument(_ context.Context, arg store.UpsertProjectDocumentParams) (store.ProjectDocument, error) {
	projectID := project.Slug(arg.ProjectID)
	now := time.Now().UTC()
	docs := f.docs[projectID]
	for i := range docs {
		if docs[i].Kind != arg.Kind {
			continue
		}
		docs[i].Title = arg.Title
		docs[i].Body = arg.Body
		docs[i].Metadata = arg.Metadata
		docs[i].UpdatedAt = now
		f.docs[projectID] = docs
		return docs[i], nil
	}
	doc := store.ProjectDocument{
		ID:        int64(len(docs) + 1),
		ProjectID: projectID,
		Kind:      arg.Kind,
		Title:     arg.Title,
		Body:      arg.Body,
		Metadata:  arg.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.docs[projectID] = append(docs, doc)
	return doc, nil
}

// ListProjectDocuments 模拟列出指定项目的所有文档操作。
func (f *fakeStore) ListProjectDocuments(_ context.Context, projectID string) ([]store.ProjectDocument, error) {
	return f.docs[project.Slug(projectID)], nil
}

// CreateRun 模拟创建运行记录操作。
// 返回一个初始状态为 "running" 的运行记录。
func (f *fakeStore) CreateRun(_ context.Context, projectID string, input string, _ json.RawMessage) (store.Run, error) {
	return store.Run{ID: 1, ProjectID: project.Slug(projectID), Input: input, Status: "running"}, nil
}

// FinishRun 模拟完成运行记录操作。
// 更新运行记录的状态为 "completed" 并设置最终文本和目录。
func (f *fakeStore) FinishRun(_ context.Context, id int64, finalText, runDir string) (store.Run, error) {
	return store.Run{ID: id, FinalText: finalText, RunDir: runDir, Status: "completed"}, nil
}

// FailRun 模拟运行失败操作。
// 更新运行记录的状态为 "failed" 并设置错误信息。
func (f *fakeStore) FailRun(_ context.Context, id int64, message string) (store.Run, error) {
	return store.Run{ID: id, Status: "failed", Error: message}, nil
}

// TestHTTPCreateProjectAndDryRun 测试创建项目和干跑（dry-run）流程。
// 验证项目创建成功后，可以使用该项目进行干跑，并返回预期的元数据。
func TestHTTPCreateProjectAndDryRun(t *testing.T) {
	// 初始化测试配置
	cfg := newHTTPTestConfig(t)
	// 创建假的存储实例
	db := newFakeStore()
	// 预置一个测试用的模型配置，用于后续的干跑测试
	db.models["deepseek-flash"] = store.ModelProfile{
		ID:              "deepseek-flash",
		Name:            "DeepSeek Flash",
		Provider:        "openai_compatible",
		ModelID:         "deepseek-v4-flash",
		BaseURL:         "https://api.deepseek.com",
		APIKey:          "sk-test",
		APIKeySet:       true,
		ContextWindow:   131072,
		MaxOutputTokens: 8192,
		Temperature:     0.7,
		TimeoutSeconds:  30,
		Status:          "active",
	}
	// 构建 HTTP 路由
	router := buildHTTPRouter(cfg, false, db, nil, nil)

	// 1. 测试创建项目
	// 构造创建项目的请求体
	projectBody := bytes.NewBufferString(`{"name":"都市异能悬疑"}`)
	// 创建 HTTP 请求
	projectReq := httptest.NewRequest(http.MethodPost, "/v1/projects", projectBody)
	projectReq.Header.Set("Content-Type", "application/json")
	// 创建响应记录器
	projectRec := httptest.NewRecorder()
	// 执行请求
	router.ServeHTTP(projectRec, projectReq)
	// 验证响应状态码是否为 200 OK
	if projectRec.Code != http.StatusOK {
		t.Fatalf("expected project create 200, got %d: %s", projectRec.Code, projectRec.Body.String())
	}

	// 解析创建项目的响应内容
	var projectPayload struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	}
	if err := json.Unmarshal(projectRec.Body.Bytes(), &projectPayload); err != nil {
		t.Fatalf("decode project payload failed: %v", err)
	}
	// 验证返回的项目 ID 是否符合预期（基于名称生成的 slug）
	if projectPayload.Project.ID != "都市异能悬疑" {
		t.Fatalf("unexpected project id: %q", projectPayload.Project.ID)
	}
	projectDir := filepath.Join(cfg.Runtime.WorkspaceRoot, "projects", projectPayload.Project.ID)
	assertPathExists(t, filepath.Join(projectDir, "documents"))
	var projectMeta struct {
		ProjectID       string   `json:"project_id"`
		StorageProvider string   `json:"storage_provider"`
		StoragePrefix   string   `json:"storage_prefix"`
		DocumentCount   int      `json:"document_count"`
		DocumentKinds   []string `json:"document_kinds"`
	}
	readJSONFile(t, filepath.Join(projectDir, "meta.json"), &projectMeta)
	if projectMeta.ProjectID != projectPayload.Project.ID {
		t.Fatalf("unexpected meta project_id: %#v", projectMeta)
	}
	if projectMeta.StorageProvider != "filesystem" || projectMeta.StoragePrefix != projectPayload.Project.ID {
		t.Fatalf("unexpected meta storage fields: %#v", projectMeta)
	}
	if projectMeta.DocumentCount != 0 || len(projectMeta.DocumentKinds) != 0 {
		t.Fatalf("expected empty project meta on create, got %#v", projectMeta)
	}

	documentBody := bytes.NewBufferString(`{"title":"世界规则","body":"遗物会保留死者最后三分钟的执念。"}`)
	documentReq := httptest.NewRequest(http.MethodPut, "/v1/projects/"+projectPayload.Project.ID+"/documents/world_rules", documentBody)
	documentReq.Header.Set("Content-Type", "application/json")
	documentRec := httptest.NewRecorder()
	router.ServeHTTP(documentRec, documentReq)
	if documentRec.Code != http.StatusOK {
		t.Fatalf("expected document upsert 200, got %d: %s", documentRec.Code, documentRec.Body.String())
	}
	documentPath := filepath.Join(projectDir, "documents", "world_rules.md")
	assertPathExists(t, documentPath)
	markdownBody, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatalf("read markdown document failed: %v", err)
	}
	if !bytes.Contains(markdownBody, []byte("遗物会保留死者最后三分钟的执念。")) {
		t.Fatalf("unexpected markdown document body: %s", string(markdownBody))
	}
	var documentMeta struct {
		ProjectID string `json:"project_id"`
		Kind      string `json:"kind"`
		Title     string `json:"title"`
	}
	readJSONFile(t, filepath.Join(projectDir, "documents", "world_rules.meta.json"), &documentMeta)
	if documentMeta.ProjectID != projectPayload.Project.ID || documentMeta.Kind != "world_rules" || documentMeta.Title != "世界规则" {
		t.Fatalf("unexpected document meta: %#v", documentMeta)
	}
	readJSONFile(t, filepath.Join(projectDir, "meta.json"), &projectMeta)
	if projectMeta.DocumentCount != 1 || len(projectMeta.DocumentKinds) != 1 || projectMeta.DocumentKinds[0] != "world_rules" {
		t.Fatalf("expected project meta to include synced document, got %#v", projectMeta)
	}

	// 2. 测试干跑（Dry Run）
	// 构造干跑请求体，指定项目和模型
	runBody := bytes.NewBufferString(`{"project":"都市异能悬疑","model":"deepseek-flash","input":"先做起盘","dry_run":true}`)
	// 创建 HTTP 请求
	runReq := httptest.NewRequest(http.MethodPost, "/v1/runs", runBody)
	runReq.Header.Set("Content-Type", "application/json")
	// 创建响应记录器
	runRec := httptest.NewRecorder()
	// 执行请求
	router.ServeHTTP(runRec, runReq)
	// 验证响应状态码是否为 200 OK
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected dry run 200, got %d: %s", runRec.Code, runRec.Body.String())
	}

	// 解析干跑响应内容
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
	// 验证干跑标志位、RunID 和 RunDir 是否存在
	if !runPayload.DryRun || runPayload.RunID == "" || runPayload.RunDir == "" {
		t.Fatalf("expected dry-run metadata, got %#v", runPayload)
	}
	// 验证响应中包含正确的项目 ID
	if runPayload.Project.ID != "都市异能悬疑" {
		t.Fatalf("expected active project in run response, got %#v", runPayload.Project)
	}
}

// TestHTTPSkills 测试获取技能列表接口。
// 验证接口是否返回 200 状态码，并且响应中包含预置的测试技能信息。
func TestHTTPSkills(t *testing.T) {
	// 初始化测试配置
	cfg := newHTTPTestConfig(t)
	// 构建 HTTP 路由，使用空的 fakeStore
	router := buildHTTPRouter(cfg, false, newFakeStore(), nil, nil)

	// 创建获取技能列表的请求
	req := httptest.NewRequest(http.MethodGet, "/v1/skills", nil)
	rec := httptest.NewRecorder()
	// 执行请求
	router.ServeHTTP(rec, req)
	// 验证响应状态码
	if rec.Code != http.StatusOK {
		t.Fatalf("expected skills 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// 验证响应体中是否包含 "Test Skill" 字符串，确认识别到了测试目录下的技能
	if !bytes.Contains(rec.Body.Bytes(), []byte("Test Skill")) {
		t.Fatalf("expected response to include test skill, got %s", rec.Body.String())
	}
}

// TestHTTPModelCRUDAndDryRunSelection 测试模型配置的增删改查以及干跑时的模型选择。
// 重点验证模型列表接口是否隐藏了敏感的 API Key 原始值，但保留了 api_key_set 标志。
func TestHTTPModelCRUDAndDryRunSelection(t *testing.T) {
	// 初始化测试配置和存储
	cfg := newHTTPTestConfig(t)
	db := newFakeStore()
	router := buildHTTPRouter(cfg, false, db, nil, nil)

	// 1. 测试创建模型配置
	// 构造创建模型的请求体
	body := bytes.NewBufferString(`{"id":"deepseek-flash","name":"DeepSeek Flash","model_id":"deepseek-v4-flash","base_url":"https://api.deepseek.com","api_key":"sk-test","max_output_tokens":8192}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/models", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	// 验证创建成功
	if rec.Code != http.StatusOK {
		t.Fatalf("expected model create 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. 测试列出模型配置
	// 创建获取模型列表的请求
	listReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	// 验证列表请求成功
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected model list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	// 验证列表中包含创建的模型 ID
	if !bytes.Contains(listRec.Body.Bytes(), []byte("deepseek-flash")) {
		t.Fatalf("expected model list to include profile, got %s", listRec.Body.String())
	}
	// 安全测试：验证列表响应中不包含原始的 API Key ("sk-test")
	if bytes.Contains(listRec.Body.Bytes(), []byte("sk-test")) {
		t.Fatalf("expected model list to hide raw api key, got %s", listRec.Body.String())
	}
	// 兼容性测试：验证列表响应中不包含已弃用或内部的 api_key_env 字段
	if bytes.Contains(listRec.Body.Bytes(), []byte("api_key_env")) {
		t.Fatalf("expected model list to hide api_key_env compatibility field, got %s", listRec.Body.String())
	}
	// 验证列表响应中包含 api_key_set 标志，用于前端判断是否配置了密钥
	if !bytes.Contains(listRec.Body.Bytes(), []byte("api_key_set")) {
		t.Fatalf("expected model list to include api_key_set, got %s", listRec.Body.String())
	}

	// 3. 测试使用特定模型进行干跑
	// 构造干跑请求，指定使用 "deepseek-flash" 模型
	runBody := bytes.NewBufferString(`{"model":"deepseek-flash","input":"先做起盘","dry_run":true}`)
	runReq := httptest.NewRequest(http.MethodPost, "/v1/runs", runBody)
	runReq.Header.Set("Content-Type", "application/json")
	runRec := httptest.NewRecorder()
	router.ServeHTTP(runRec, runReq)
	// 验证干跑成功
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected dry run with model 200, got %d: %s", runRec.Code, runRec.Body.String())
	}
	// 验证响应中包含了使用的模型 ID
	if !bytes.Contains(runRec.Body.Bytes(), []byte("deepseek-flash")) {
		t.Fatalf("expected dry run response to include active model, got %s", runRec.Body.String())
	}
}

// TestHTTPDefaultModelSettingAndDeleteGuard 测试默认模型设置及删除保护逻辑。
// 验证：
// 1. 可以设置和获取默认模型。
// 2. 干跑时若不指定模型，会自动使用默认模型。
// 3. 不能删除当前被设为默认的模型（应返回 400 错误）。
// 4. 清除默认模型设置后，可以删除该模型。
func TestHTTPDefaultModelSettingAndDeleteGuard(t *testing.T) {
	// 初始化测试配置和存储
	cfg := newHTTPTestConfig(t)
	db := newFakeStore()
	router := buildHTTPRouter(cfg, false, db, nil, nil)

	// 1. 创建模型配置
	createBody := bytes.NewBufferString(`{"id":"deepseek-flash","name":"DeepSeek Flash","model_id":"deepseek-v4-flash","base_url":"https://api.deepseek.com","api_key":"sk-test","max_output_tokens":8192}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/models", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected model create 200, got %d: %s", createRec.Code, createRec.Body.String())
	}

	// 2. 设置默认模型
	// 发送 PUT 请求设置 "deepseek-flash" 为默认模型
	setReq := httptest.NewRequest(http.MethodPut, "/v1/settings/default-model", bytes.NewBufferString(`{"model":"deepseek-flash"}`))
	setReq.Header.Set("Content-Type", "application/json")
	setRec := httptest.NewRecorder()
	router.ServeHTTP(setRec, setReq)
	// 验证设置成功
	if setRec.Code != http.StatusOK {
		t.Fatalf("expected default model set 200, got %d: %s", setRec.Code, setRec.Body.String())
	}

	// 3. 获取默认模型设置
	// 发送 GET 请求读取当前默认模型
	getReq := httptest.NewRequest(http.MethodGet, "/v1/settings/default-model", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	// 验证读取到的默认模型 ID 正确
	if getRec.Code != http.StatusOK || !bytes.Contains(getRec.Body.Bytes(), []byte(`"default_model_id":"deepseek-flash"`)) {
		t.Fatalf("expected default model readback, got %d: %s", getRec.Code, getRec.Body.String())
	}

	// 4. 测试未指定模型时的干跑（应自动使用默认模型）
	// 构造干跑请求，不指定 model 字段
	runReq := httptest.NewRequest(http.MethodPost, "/v1/runs", bytes.NewBufferString(`{"input":"先做起盘","dry_run":true}`))
	runReq.Header.Set("Content-Type", "application/json")
	runRec := httptest.NewRecorder()
	router.ServeHTTP(runRec, runReq)
	// 验证干跑成功，且响应中包含了默认模型 "deepseek-flash"
	if runRec.Code != http.StatusOK || !bytes.Contains(runRec.Body.Bytes(), []byte("deepseek-flash")) {
		t.Fatalf("expected dry run without explicit model to use default setting, got %d: %s", runRec.Code, runRec.Body.String())
	}

	// 5. 测试删除默认模型的保护机制
	// 尝试删除当前设为默认的模型 "deepseek-flash"
	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/models/deepseek-flash", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	// 验证请求被拒绝，返回 400 Bad Request
	if deleteRec.Code != http.StatusBadRequest {
		t.Fatalf("expected deleting default model to fail, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	// 6. 清除默认模型设置
	// 发送 DELETE 请求清除默认模型设置
	clearReq := httptest.NewRequest(http.MethodDelete, "/v1/settings/default-model", nil)
	clearRec := httptest.NewRecorder()
	router.ServeHTTP(clearRec, clearReq)
	// 验证清除成功
	if clearRec.Code != http.StatusOK {
		t.Fatalf("expected default model clear 200, got %d: %s", clearRec.Code, clearRec.Body.String())
	}

	// 7. 再次尝试删除模型（此时已非默认模型）
	deleteRec = httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	// 验证删除成功，返回 200 OK
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected deleting non-default model to succeed, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

// newHTTPTestConfig 创建一个用于测试的配置对象。
// 它会创建一个临时目录，并在其中设置一个测试用的 Skill 文件结构。
// 使用 t.TempDir() 确保测试结束后自动清理临时文件。
func newHTTPTestConfig(t *testing.T) config.Config {
	t.Helper()
	// 创建临时根目录
	root := t.TempDir()
	// 构建 Skills 目录路径
	skillsDir := filepath.Join(root, "skills")
	// 构建具体测试 Skill 的目录路径
	skillDir := filepath.Join(skillsDir, "test-skill")
	// 创建目录结构
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	// 写入测试用的 SKILL.md 文件，包含 YAML frontmatter 和正文
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
	// 返回配置对象，指向临时目录和测试模型设置
	return config.Config{
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

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist %s: %v", path, err)
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode %s failed: %v", path, err)
	}
}
