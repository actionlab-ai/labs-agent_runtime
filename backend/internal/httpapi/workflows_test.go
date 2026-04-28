package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"novel-agent-runtime/internal/store"
)

func TestHTTPProjectKickoffWorkflowDryRunReturnsFixedPlan(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	writeWorkflowTestSkill(t, cfg.Runtime.SkillsDir, "novel-project-kickoff", `---
name: Project Kickoff
description: Fixed kickoff workflow step
when_to_use: Initialize project direction
user_invocable: true
---
Return project kickoff docs.
`)

	db := newFakeStore()
	db.projects["urban-safe-growth"] = store.Project{
		ID:              "urban-safe-growth",
		Name:            "都市安全成长流",
		Status:          "active",
		StorageProvider: "filesystem",
		StoragePrefix:   "urban-safe-growth",
	}
	db.models["deepseek-flash"] = store.ModelProfile{
		ID:              "deepseek-flash",
		Name:            "DeepSeek Flash",
		Provider:        "openai_compatible",
		ModelID:         "deepseek-v4-flash",
		BaseURL:         "https://example.invalid",
		APIKey:          "sk-test",
		APIKeySet:       true,
		ContextWindow:   131072,
		MaxOutputTokens: 8192,
		Temperature:     0.7,
		TimeoutSeconds:  30,
		Status:          "active",
	}

	router := buildHTTPRouter(cfg, false, db, nil, nil)
	body := bytes.NewBufferString(`{"project":"urban-safe-growth","model":"deepseek-flash","input":"先把这本书的项目调性定下来","dry_run":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/project-kickoff", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workflow dry run 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		DryRun     bool             `json:"dry_run"`
		WorkflowID string           `json:"workflow_id"`
		Stage      string           `json:"stage"`
		Arguments  map[string]any   `json:"arguments"`
		Steps      []map[string]any `json:"steps"`
		RunDir     string           `json:"run_dir"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode workflow dry run payload failed: %v", err)
	}
	if !payload.DryRun {
		t.Fatalf("expected dry_run=true, got %#v", payload)
	}
	if payload.WorkflowID != "project-kickoff" || payload.Stage != "kickoff" {
		t.Fatalf("unexpected workflow plan identity: %#v", payload)
	}
	if len(payload.Steps) != 1 || payload.Steps[0]["skill_id"] != "novel-project-kickoff" {
		t.Fatalf("expected fixed kickoff step, got %#v", payload.Steps)
	}
	if payload.Arguments["question_mode"] != "clarify_first" {
		t.Fatalf("expected fixed clarify_first argument, got %#v", payload.Arguments)
	}
	assertPathExists(t, filepath.Join(payload.RunDir, "workflow", "plan.json"))
}

func TestHTTPProjectKickoffWorkflowPersistsStructuredDocuments(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	writeWorkflowTestSkill(t, cfg.Runtime.SkillsDir, "novel-project-kickoff", `---
name: Project Kickoff
description: Fixed kickoff workflow step
when_to_use: Initialize project direction
user_invocable: true
---
Return project kickoff docs.
`)

	db := newFakeStore()
	db.projects["urban-safe-growth"] = store.Project{
		ID:              "urban-safe-growth",
		Name:            "都市安全成长流",
		Status:          "active",
		StorageProvider: "filesystem",
		StoragePrefix:   "urban-safe-growth",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	var requestCount atomic.Int32
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected model path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"## project_brief\n\n这本书是男频都市安全成长流，先卖安全感，再卖持续升级。\n\n## reader_contract\n\n前二十章给读者确定的安全区和稳定变强反馈。\n\n## style_guide\n\n开篇直接，信息释放清晰，避免绕弯。\n\n## taboo\n\n不要苦大仇深，不要一上来把主角打进绝境。"},"finish_reason":"stop"}]}`)
	}))
	defer modelServer.Close()

	db.models["deepseek-flash"] = store.ModelProfile{
		ID:              "deepseek-flash",
		Name:            "DeepSeek Flash",
		Provider:        "openai_compatible",
		ModelID:         "deepseek-v4-flash",
		BaseURL:         modelServer.URL,
		APIKey:          "sk-test",
		APIKeySet:       true,
		ContextWindow:   131072,
		MaxOutputTokens: 8192,
		Temperature:     0.7,
		TimeoutSeconds:  30,
		Status:          "active",
	}

	router := buildHTTPRouter(cfg, false, db, nil, nil)
	body := bytes.NewBufferString(`{"project":"urban-safe-growth","model":"deepseek-flash","input":"男频 番茄 都市重生 安全感 成长流，先做第一步定调"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/project-kickoff", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workflow run 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		WorkflowID       string `json:"workflow_id"`
		Stage            string `json:"stage"`
		ResponseMode     string `json:"response_mode"`
		NeedsInput       bool   `json:"needs_input"`
		FinalText        string `json:"final_text"`
		RunDir           string `json:"run_dir"`
		UpdatedDocuments []struct {
			Kind string `json:"kind"`
		} `json:"updated_documents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode workflow payload failed: %v", err)
	}
	if payload.WorkflowID != "project-kickoff" || payload.Stage != "kickoff" {
		t.Fatalf("unexpected workflow response: %#v", payload)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected direct skill execution with one model request, got %d", requestCount.Load())
	}
	if strings.TrimSpace(payload.FinalText) == "" || !strings.Contains(payload.FinalText, "## project_brief") {
		t.Fatalf("expected final text to come from kickoff skill, got %q", payload.FinalText)
	}
	if len(payload.UpdatedDocuments) != 4 {
		t.Fatalf("expected four persisted project docs, got %#v", payload.UpdatedDocuments)
	}
	kinds := map[string]bool{}
	for _, item := range payload.UpdatedDocuments {
		kinds[item.Kind] = true
	}
	for _, kind := range []string{"project_brief", "reader_contract", "style_guide", "taboo"} {
		if !kinds[kind] {
			t.Fatalf("expected updated documents to contain %q, got %#v", kind, payload.UpdatedDocuments)
		}
	}
	if _, err := os.Stat(filepath.Join(payload.RunDir, "router")); !os.IsNotExist(err) {
		t.Fatalf("expected fixed workflow path to bypass router artifacts, stat err=%v", err)
	}
	assertPathExists(t, filepath.Join(payload.RunDir, "skill-calls", "novel-project-kickoff", "compiled-prompt.md"))
	assertPathExists(t, filepath.Join(cfg.Runtime.WorkspaceRoot, "projects", "urban-safe-growth", "documents", "project_brief.md"))
}

func TestHTTPProjectKernelWorkflowRunsEmotionalCoreAndPersistsNovelCore(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	writeWorkflowTestSkill(t, cfg.Runtime.SkillsDir, "novel-emotional-core", `---
name: Emotional Core
description: Fixed kernel workflow step
when_to_use: Initialize story emotional core
user_invocable: true
---
Return one durable novel_core document.
`)

	db := newFakeStore()
	db.projects["urban-safe-growth"] = store.Project{
		ID:              "urban-safe-growth",
		Name:            "都市安全成长流",
		Status:          "active",
		StorageProvider: "filesystem",
		StoragePrefix:   "urban-safe-growth",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	var requestCount atomic.Int32
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected model path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"# 小说情感内核\n\n## 核心一句话\n\n一个长期被轻视的小人物，在现实压迫下，通过建立绝对安全区，一步步夺回尊严与掌控感。\n\n## 读者情绪入口\n\n被生活压着走的人，也想先拥有一块不会再被夺走的立足点。"},"finish_reason":"stop"}]}`)
	}))
	defer modelServer.Close()

	db.models["deepseek-flash"] = store.ModelProfile{
		ID:              "deepseek-flash",
		Name:            "DeepSeek Flash",
		Provider:        "openai_compatible",
		ModelID:         "deepseek-v4-flash",
		BaseURL:         modelServer.URL,
		APIKey:          "sk-test",
		APIKeySet:       true,
		ContextWindow:   131072,
		MaxOutputTokens: 8192,
		Temperature:     0.7,
		TimeoutSeconds:  30,
		Status:          "active",
	}

	router := buildHTTPRouter(cfg, false, db, nil, nil)
	body := bytes.NewBufferString(`{"project":"urban-safe-growth","model":"deepseek-flash","input":"先把这本书真正的情感内核定下来：读者要的是安全感、尊严和持续变强。"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/project-kernel", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected kernel workflow run 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		WorkflowID       string `json:"workflow_id"`
		Stage            string `json:"stage"`
		ResponseMode     string `json:"response_mode"`
		NeedsInput       bool   `json:"needs_input"`
		FinalText        string `json:"final_text"`
		RunDir           string `json:"run_dir"`
		UpdatedDocuments []struct {
			Kind string `json:"kind"`
		} `json:"updated_documents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode kernel workflow payload failed: %v", err)
	}
	if payload.WorkflowID != "project-kernel" || payload.Stage != "kernel" {
		t.Fatalf("unexpected kernel workflow response: %#v", payload)
	}
	if payload.ResponseMode != workflowResponseModeDocument || payload.NeedsInput {
		t.Fatalf("expected document response mode without needs_input, got %#v", payload)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected direct emotional-core execution with one model request, got %d", requestCount.Load())
	}
	if !strings.Contains(payload.FinalText, "# 小说情感内核") {
		t.Fatalf("expected emotional-core text, got %q", payload.FinalText)
	}
	if len(payload.UpdatedDocuments) != 1 || payload.UpdatedDocuments[0].Kind != "novel_core" {
		t.Fatalf("expected one novel_core document update, got %#v", payload.UpdatedDocuments)
	}
	assertPathExists(t, filepath.Join(payload.RunDir, "skill-calls", "novel-emotional-core", "compiled-prompt.md"))
	assertPathExists(t, filepath.Join(cfg.Runtime.WorkspaceRoot, "projects", "urban-safe-growth", "documents", "novel_core.md"))
	projectDocs := db.docs["urban-safe-growth"]
	if len(projectDocs) != 1 || projectDocs[0].Kind != "novel_core" {
		t.Fatalf("expected fake store novel_core doc, got %#v", projectDocs)
	}
}

func TestHTTPProjectKernelWorkflowReturnsClarificationWithoutPersistingNovelCore(t *testing.T) {
	cfg := newHTTPTestConfig(t)
	writeWorkflowTestSkill(t, cfg.Runtime.SkillsDir, "novel-emotional-core", `---
name: Emotional Core
description: Fixed kernel workflow step
when_to_use: Initialize story emotional core
user_invocable: true
---
Ask clarification questions when key information is missing.
`)

	db := newFakeStore()
	db.projects["urban-safe-growth"] = store.Project{
		ID:              "urban-safe-growth",
		Name:            "Urban Safe Growth",
		Status:          "active",
		StorageProvider: "filesystem",
		StoragePrefix:   "urban-safe-growth",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	var requestCount atomic.Int32
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected model path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"id":"resp-clarify","choices":[{"index":0,"message":{"role":"assistant","content":"先不定稿，下面进入澄清模式。\n\n# 需要补充的信息\n\n## 缺失字段\n- protagonist_seed\n- pressure_source\n- avoid\n\n## 当前已确认\n- 这是一本都市成长向男频小说。\n- 读者追求安全感和持续变强。\n\n## 还不能定稿的原因\n- 主角当前处境不明确。\n- 压迫系统和边界条件还缺关键事实。\n\n## 请先回答以下问题\n\n### protagonist_seed | 主角当前最像以下哪类人？\n- A. 刚毕业的廉价合租社畜\n- B. 校园里最没存在感的男生\n- C. 长期被家庭忽视的人\n\n### pressure_source | 压着他的那股力量主要来自哪里？\n- A. 职场和上层规则\n- B. 家庭与债务\n- C. 更强的超凡体系\n\n### avoid | 哪些方向能要，哪些明确不能碰？\n- A. 可以打脸翻盘，但别物化女性\n- B. 可以有暧昧，但别全员倒贴\n- C. 不要太阴暗苦大仇深"},"finish_reason":"stop"}]}`)
	}))
	defer modelServer.Close()

	db.models["deepseek-flash"] = store.ModelProfile{
		ID:              "deepseek-flash",
		Name:            "DeepSeek Flash",
		Provider:        "openai_compatible",
		ModelID:         "deepseek-v4-flash",
		BaseURL:         modelServer.URL,
		APIKey:          "sk-test",
		APIKeySet:       true,
		ContextWindow:   131072,
		MaxOutputTokens: 8192,
		Temperature:     0.7,
		TimeoutSeconds:  30,
		Status:          "active",
	}

	router := buildHTTPRouter(cfg, false, db, nil, nil)
	body := bytes.NewBufferString(`{"project":"urban-safe-growth","model":"deepseek-flash","input":"先做这本书的情感内核，但我现在只知道它是都市成长流，读者想要安全感和持续变强。"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/project-kernel", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected kernel clarification workflow 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		WorkflowID    string `json:"workflow_id"`
		Stage         string `json:"stage"`
		ResponseMode  string `json:"response_mode"`
		NeedsInput    bool   `json:"needs_input"`
		Clarification *struct {
			MissingFields   []string `json:"missing_fields"`
			ConfirmedFacts  []string `json:"confirmed_facts"`
			BlockingReasons []string `json:"blocking_reasons"`
			Questions       []struct {
				Field   string   `json:"field"`
				Prompt  string   `json:"prompt"`
				Options []string `json:"options"`
			} `json:"questions"`
		} `json:"clarification"`
		FinalText        string `json:"final_text"`
		RunDir           string `json:"run_dir"`
		UpdatedDocuments []struct {
			Kind string `json:"kind"`
		} `json:"updated_documents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode kernel clarification workflow payload failed: %v", err)
	}
	if payload.WorkflowID != "project-kernel" || payload.Stage != "kernel" {
		t.Fatalf("unexpected kernel clarification workflow response: %#v", payload)
	}
	if payload.ResponseMode != workflowResponseModeClarification || !payload.NeedsInput {
		t.Fatalf("expected clarification response mode with needs_input, got %#v", payload)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected one model request, got %d", requestCount.Load())
	}
	if !strings.HasPrefix(strings.TrimSpace(payload.FinalText), "# 需要补充的信息") {
		t.Fatalf("expected clarification prompt text, got %q", payload.FinalText)
	}
	if strings.Contains(payload.FinalText, "先不定稿") {
		t.Fatalf("expected final_text to be normalized without preamble, got %q", payload.FinalText)
	}
	if len(payload.UpdatedDocuments) != 0 {
		t.Fatalf("expected no persisted novel_core updates, got %#v", payload.UpdatedDocuments)
	}
	if payload.Clarification == nil {
		t.Fatalf("expected structured clarification payload, got nil")
	}
	if got := strings.Join(payload.Clarification.MissingFields, ","); got != "protagonist_seed,pressure_source,avoid" {
		t.Fatalf("unexpected missing_fields: %q", got)
	}
	if len(payload.Clarification.Questions) != 3 {
		t.Fatalf("expected three clarification questions, got %#v", payload.Clarification.Questions)
	}
	if payload.Clarification.Questions[0].Field != "protagonist_seed" {
		t.Fatalf("expected first clarification field protagonist_seed, got %#v", payload.Clarification.Questions[0])
	}
	assertPathExists(t, filepath.Join(payload.RunDir, "skill-calls", "novel-emotional-core", "compiled-prompt.md"))
	assertPathExists(t, filepath.Join(payload.RunDir, "workflow", "clarification.json"))
	projectDocs := db.docs["urban-safe-growth"]
	if len(projectDocs) != 0 {
		t.Fatalf("expected no project docs persisted during clarification, got %#v", projectDocs)
	}
	if _, err := os.Stat(filepath.Join(cfg.Runtime.WorkspaceRoot, "projects", "urban-safe-growth", "documents", "novel_core.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no novel_core.md to be created during clarification, stat err=%v", err)
	}
}

func writeWorkflowTestSkill(t *testing.T, skillsDir, skillID, body string) {
	t.Helper()
	skillDir := filepath.Join(skillsDir, skillID)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}
}
