package runtime

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
	"novel-agent-runtime/internal/skill"
)

func TestToolSpecsEnforceSearchFirstAndActivateSkillTools(t *testing.T) {
	rt := newTestRuntime(t)

	initial := rt.toolSpecs(runState{})
	if len(initial) != 1 || initial[0].Function.Name != "tool_search" {
		t.Fatalf("expected only tool_search before activation, got %#v", initial)
	}

	activated := []activatedSkillRef{{
		SkillID:  "webnovel-opening-sniper",
		ToolName: activeSkillToolName("webnovel-opening-sniper"),
		Name:     "Opening Sniper",
	}}
	afterSearch := rt.toolSpecs(runState{SearchPerformed: true, RetainedSkills: activated})

	var names []string
	for _, spec := range afterSearch {
		names = append(names, spec.Function.Name)
	}
	if len(names) != 3 {
		t.Fatalf("expected tool_search + activated skill tool + skill_call, got %v", names)
	}
	if names[1] != activeSkillToolName("webnovel-opening-sniper") {
		t.Fatalf("expected activated skill tool to appear, got %v", names)
	}
	if names[2] != "skill_call" {
		t.Fatalf("expected compatibility skill_call to remain available, got %v", names)
	}
	if got := afterSearch[1].Function.Parameters["type"]; got != "object" {
		t.Fatalf("expected activated skill tool to expose object schema, got %#v", afterSearch[1].Function.Parameters)
	}
	properties, ok := afterSearch[1].Function.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected activated skill schema properties, got %#v", afterSearch[1].Function.Parameters)
	}
	if _, ok := properties["premise"]; !ok {
		t.Fatalf("expected activated skill schema to include skill-specific field premise, got %#v", properties)
	}
}

func TestBuildRouterAssemblyPrependsDeferredSkillsReminder(t *testing.T) {
	rt := newTestRuntime(t)
	assembly := rt.buildRouterAssembly("write an urban opening", runState{}, []model.Message{
		{Role: "user", Content: "write an urban opening"},
	}, 1)

	if len(assembly.Messages) < 3 {
		t.Fatalf("expected system + deferred reminder + user message, got %#v", assembly.Messages)
	}
	if assembly.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %#v", assembly.Messages[0])
	}
	if assembly.Messages[1].Role != "user" || !strings.Contains(assembly.Messages[1].Content, "<available-deferred-skills>") {
		t.Fatalf("expected deferred skills reminder in second message, got %#v", assembly.Messages[1])
	}
	if !strings.Contains(assembly.Messages[1].Content, "select:webnovel-opening-sniper") {
		t.Fatalf("expected reminder to teach select:<skill_id> usage, got %q", assembly.Messages[1].Content)
	}
}

func TestBuildRouterAssemblyPrependsRetainedSkillToolsReminder(t *testing.T) {
	rt := newTestRuntime(t)
	state := runState{
		SearchPerformed: true,
		SearchCount:     1,
		RetainedSkills: []activatedSkillRef{{
			SkillID:  "webnovel-opening-sniper",
			ToolName: activeSkillToolName("webnovel-opening-sniper"),
			Name:     "Opening Sniper",
		}},
	}
	assembly := rt.buildRouterAssembly("write an urban opening", state, []model.Message{
		{Role: "user", Content: "write an urban opening"},
	}, 2)

	found := false
	for _, msg := range assembly.Messages {
		if msg.Role == "user" && strings.Contains(msg.Content, "<retained-skill-tools>") {
			found = true
			if !strings.Contains(msg.Content, activeSkillToolName("webnovel-opening-sniper")) {
				t.Fatalf("expected retained reminder to include active tool name, got %q", msg.Content)
			}
		}
	}
	if !found {
		t.Fatalf("expected retained skill tools reminder in router messages, got %#v", assembly.Messages)
	}
}

func TestHandleToolSearchActivatesDynamicSkillTools(t *testing.T) {
	rt := newTestRuntime(t)

	args, _ := json.Marshal(map[string]any{
		"query": "urban opening",
		"limit": 3,
	})
	outcome, err := rt.handleToolCall(
		context.Background(),
		"write an urban power opening",
		model.ToolCall{
			ID:   "call_1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "tool_search",
				Arguments: string(args),
			},
		},
		runState{},
	)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}
	if !outcome.SearchPerformed {
		t.Fatalf("expected tool_search to flip SearchPerformed")
	}
	if len(outcome.RetainedSkills) == 0 {
		t.Fatalf("expected retained skill tools from search hit")
	}
	if outcome.RetainedSkills[0].SkillID != "webnovel-opening-sniper" {
		t.Fatalf("expected opening skill to be retained, got %#v", outcome.RetainedSkills)
	}
	if outcome.RetainedSkills[0].ToolName == "" {
		t.Fatalf("expected activated tool name to be generated")
	}
	var payload struct {
		ToolReferences []callableToolReference `json:"tool_references"`
	}
	if err := json.Unmarshal([]byte(outcome.Content), &payload); err != nil {
		t.Fatalf("expected tool_search payload to stay valid json: %v", err)
	}
	if len(payload.ToolReferences) == 0 {
		t.Fatalf("expected explicit tool references in tool_search payload")
	}
	if payload.ToolReferences[0].Type != "tool_reference_like" {
		t.Fatalf("expected tool_reference_like payload, got %#v", payload.ToolReferences[0])
	}
	if payload.ToolReferences[0].Contract != "opening_v1" {
		t.Fatalf("expected tool contract in payload, got %#v", payload.ToolReferences[0])
	}
}

func TestActivateSkillsUsesScoreWindowPolicy(t *testing.T) {
	rt := newTestRuntime(t)

	plan := rt.activateSkills(runState{}, skill.ExplainQuery("opening"), []skill.SearchHit{
		{ID: "webnovel-opening-sniper", Name: "Opening Sniper", Score: 1.0, Reason: "top-hit"},
		{ID: "second-opening-skill", Name: "Second Skill", Score: 0.62, Reason: "close-enough"},
		{ID: "outline-builder", Name: "Outline Builder", Score: 0.20, Reason: "too-low"},
	})

	if plan.Policy != "score-window" {
		t.Fatalf("expected score-window policy, got %#v", plan)
	}
	if len(plan.WindowSkills) != 2 {
		t.Fatalf("expected only the first two skills in the activation window, got %#v", plan.WindowSkills)
	}
	if plan.WindowSkills[1].SkillID != "second-opening-skill" {
		t.Fatalf("expected second skill to survive score window, got %#v", plan.WindowSkills)
	}
	if len(plan.SkippedSkills) == 0 || plan.SkippedSkills[0] != "outline-builder" {
		t.Fatalf("expected low-score skill to be skipped, got %#v", plan.SkippedSkills)
	}
	if len(plan.RetainedSkills) != 2 {
		t.Fatalf("expected retained pool to match activation window on first search, got %#v", plan.RetainedSkills)
	}
}

func TestActivateSkillsRetainsPreviouslyDiscoveredSkills(t *testing.T) {
	rt := newTestRuntime(t)

	state := runState{
		SearchPerformed: true,
		SearchCount:     1,
		RetainedSkills: []activatedSkillRef{{
			SkillID:  "webnovel-opening-sniper",
			ToolName: activeSkillToolName("webnovel-opening-sniper"),
			Name:     "Opening Sniper",
		}},
	}

	plan := rt.activateSkills(state, skill.ExplainQuery("outline"), []skill.SearchHit{
		{ID: "outline-builder", Name: "Outline Builder", Score: 0.95, Reason: "new-top-hit"},
	})

	if len(plan.WindowSkills) != 1 || plan.WindowSkills[0].SkillID != "outline-builder" {
		t.Fatalf("expected outline builder in the fresh window, got %#v", plan.WindowSkills)
	}
	if len(plan.NewlyActivated) != 1 || plan.NewlyActivated[0].SkillID != "outline-builder" {
		t.Fatalf("expected outline builder to be marked as new, got %#v", plan.NewlyActivated)
	}
	if len(plan.RetainedSkills) != 2 {
		t.Fatalf("expected retained pool to keep the previous skill too, got %#v", plan.RetainedSkills)
	}
	if plan.RetainedSkills[0].SkillID != "outline-builder" || plan.RetainedSkills[1].SkillID != "webnovel-opening-sniper" {
		t.Fatalf("expected fresh window first and previous pool after it, got %#v", plan.RetainedSkills)
	}
}

func TestActivateSkillsMarksRepeatedSearchAsUnchanged(t *testing.T) {
	rt := newTestRuntime(t)

	retained := []activatedSkillRef{{
		SkillID:  "webnovel-opening-sniper",
		ToolName: activeSkillToolName("webnovel-opening-sniper"),
		Name:     "Opening Sniper",
	}}
	state := runState{
		SearchPerformed:       true,
		SearchCount:           1,
		LastSearchFingerprint: searchFingerprint(skill.ExplainQuery("urban opening")),
		RetainedSkills:        retained,
	}

	plan := rt.activateSkills(state, skill.ExplainQuery("urban opening"), []skill.SearchHit{
		{ID: "webnovel-opening-sniper", Name: "Opening Sniper", Score: 1.0, Reason: "same-top-hit"},
	})

	if !plan.Unchanged {
		t.Fatalf("expected repeated search to be marked unchanged, got %#v", plan)
	}
	if len(plan.ReusedSkills) != 1 || plan.ReusedSkills[0].SkillID != "webnovel-opening-sniper" {
		t.Fatalf("expected retained skill to be marked reused, got %#v", plan.ReusedSkills)
	}
	if len(plan.NewlyActivated) != 0 {
		t.Fatalf("expected no newly activated skills, got %#v", plan.NewlyActivated)
	}
}

func TestActivateSkillsEvictsOldestRetainedSkillWhenPoolIsFull(t *testing.T) {
	rt := newTestRuntime(t)
	rt.Config.Runtime.MaxRetainedSkills = 2

	state := runState{
		SearchPerformed: true,
		SearchCount:     2,
		RetainedSkills: []activatedSkillRef{
			{
				SkillID:  "webnovel-opening-sniper",
				ToolName: activeSkillToolName("webnovel-opening-sniper"),
				Name:     "Opening Sniper",
			},
			{
				SkillID:  "second-opening-skill",
				ToolName: activeSkillToolName("second-opening-skill"),
				Name:     "Second Skill",
			},
		},
	}

	plan := rt.activateSkills(state, skill.ExplainQuery("outline"), []skill.SearchHit{
		{ID: "outline-builder", Name: "Outline Builder", Score: 0.99, Reason: "new-top-hit"},
	})

	if len(plan.EvictedSkills) != 1 || plan.EvictedSkills[0].SkillID != "second-opening-skill" {
		t.Fatalf("expected the oldest retained tail skill to be evicted, got %#v", plan.EvictedSkills)
	}
	if len(plan.RetainedSkills) != 2 {
		t.Fatalf("expected retained pool to obey max_retained_skills, got %#v", plan.RetainedSkills)
	}
	if plan.RetainedSkills[0].SkillID != "outline-builder" || plan.RetainedSkills[1].SkillID != "webnovel-opening-sniper" {
		t.Fatalf("expected newest window first and previous survivor second, got %#v", plan.RetainedSkills)
	}
}

func TestEffectiveSkillMaxTokensClampsDeepSeek(t *testing.T) {
	rt := newTestRuntime(t)
	rt.ModelConfig.BaseURL = "https://api.deepseek.com"
	rt.ModelConfig.MaxOutput = 409600

	if got := rt.effectiveSkillMaxTokens(); got != 393216 {
		t.Fatalf("expected deepseek max token clamp, got %d", got)
	}
}

func TestNormalizeSkillOutputExtractsOpeningSection(t *testing.T) {
	raw := strings.TrimSpace(`
## 本次开篇弹种
悬疑迷题弹

## 开篇正文（600字以内）
第一句。
第二句。

## 自检结果
- 通过
`)

	got := normalizeSkillOutput(raw)
	want := "第一句。\n第二句。"
	if got != want {
		t.Fatalf("expected normalized body %q, got %q", want, got)
	}
}

func TestNormalizeSkillOutputStripsPresentationHeading(t *testing.T) {
	raw := strings.TrimSpace(`
# 网文开篇狙击手·正文输出

第一句。
第二句。
`)

	got := normalizeSkillOutput(raw)
	want := "第一句。\n第二句。"
	if got != want {
		t.Fatalf("expected presentation heading to be stripped, got %q", got)
	}
}

func TestRunCarriesReasoningContentBetweenToolRounds(t *testing.T) {
	rt := newTestRuntime(t)
	t.Setenv("TEST_API_KEY", "dummy")

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := requestCount.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var req model.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		switch current {
		case 1:
			_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"Searching skills","reasoning_content":"Need tool search first","tool_calls":[{"id":"call_1","type":"function","function":{"name":"tool_search","arguments":"{\"query\":\"urban opening\",\"limit\":3}"}}]},"finish_reason":"tool_calls"}]}`)
		case 2:
			found := false
			for _, msg := range req.Messages {
				if msg.Role == "assistant" && msg.ReasoningContent == "Need tool search first" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected round-2 request to preserve reasoning_content, got %#v", req.Messages)
			}
			_, _ = fmt.Fprint(w, `{"id":"resp-2","choices":[{"index":0,"message":{"role":"assistant","content":"Final answer"},"finish_reason":"stop"}]}`)
		default:
			t.Fatalf("unexpected extra request #%d", current)
		}
	}))
	defer server.Close()

	rt.ModelConfig.BaseURL = server.URL
	rt.ModelConfig.APIKeyEnv = "TEST_API_KEY"
	rt.Model = model.NewOpenAICompatible(server.URL, "TEST_API_KEY", "test-model", 5)

	res, err := rt.Run(context.Background(), "write an urban opening")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := requestCount.Load(); got != 2 {
		t.Fatalf("expected exactly two model requests, got %d", got)
	}
	if !strings.Contains(res.FinalText, "Final answer") {
		t.Fatalf("expected final answer from second round, got %q", res.FinalText)
	}
}

func TestRunReturnsSkillOutputDirectWithoutRouterWrap(t *testing.T) {
	rt := newTestRuntime(t)
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
			_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"Searching skills","reasoning_content":"Need tool search first","tool_calls":[{"id":"call_1","type":"function","function":{"name":"tool_search","arguments":"{\"query\":\"urban opening\",\"limit\":3}"}}]},"finish_reason":"tool_calls"}]}`)
		case 2:
			_, _ = fmt.Fprint(w, `{"id":"resp-2","choices":[{"index":0,"message":{"role":"assistant","content":"Calling skill","reasoning_content":"Use the activated skill tool","tool_calls":[{"id":"call_2","type":"function","function":{"name":"skill_exec_webnovel_opening_sniper_28a1b3d8","arguments":"{\"task\":\"write opening\"}"}}]},"finish_reason":"tool_calls"}]}`)
		case 3:
			if len(req.Messages) < 2 || req.Messages[0].Role != "system" {
				t.Fatalf("expected skill executor request, got %#v", req.Messages)
			}
			_, _ = fmt.Fprint(w, `{"id":"resp-3","choices":[{"index":0,"message":{"role":"assistant","content":"## 本次开篇弹种\n悬疑迷题弹\n\n## 开篇正文（600字以内）\n第一句。\n第二句。\n\n## 自检结果\n- 通过"},"finish_reason":"stop"}]}`)
		default:
			t.Fatalf("unexpected extra request #%d", current)
		}
	}))
	defer server.Close()

	rt.ModelConfig.BaseURL = server.URL
	rt.ModelConfig.APIKeyEnv = "TEST_API_KEY"
	rt.Model = model.NewOpenAICompatible(server.URL, "TEST_API_KEY", "test-model", 5)

	res, err := rt.Run(context.Background(), "write an urban opening")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := requestCount.Load(); got != 3 {
		t.Fatalf("expected router search + router skill call + skill execution only, got %d requests", got)
	}
	if res.FinalText != "第一句。\n第二句。" {
		t.Fatalf("expected direct normalized skill output, got %q", res.FinalText)
	}
}

func TestComposeSkillPromptIncludesStructuredInvocationArgs(t *testing.T) {
	cmd := skill.Command{
		ID:              "webnovel-opening-sniper",
		Name:            "Opening Sniper",
		SkillRoot:       "C:/tmp/opening",
		MarkdownContent: "Body",
		ToolContract:    "opening_v1",
		ToolOutput:      "opening_prose_v1",
	}

	out := ComposeSkillPrompt(cmd, "write an opening", map[string]any{
		"task":        "write an urban opening",
		"premise":     "corpse report reveals hidden data",
		"protagonist": "rookie forensic analyst",
	}, "workspace_root: C:/tmp\npreferred_document_output_dir: C:/tmp/docs")

	if !strings.Contains(out, "# Activated Tool Arguments") {
		t.Fatalf("expected structured argument section, got %q", out)
	}
	if !strings.Contains(out, "\"premise\": \"corpse report reveals hidden data\"") {
		t.Fatalf("expected pretty-printed structured args, got %q", out)
	}
	if !strings.Contains(out, "tool_contract: opening_v1") {
		t.Fatalf("expected tool contract in prompt metadata, got %q", out)
	}
}

func TestBuildSkillAssemblyIncludesLocalToolPack(t *testing.T) {
	rt := newTestRuntime(t)
	cmd, err := rt.Registry.LoadInvocationCommand("webnovel-opening-sniper")
	if err != nil {
		t.Fatalf("LoadInvocationCommand failed: %v", err)
	}
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     rt.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: rt.Config.Runtime.DocumentOutputDir,
	}, store, "skill-calls/test")
	specs := session.toolSpecs(cmd.AllowedTools)
	assembly := rt.buildSkillAssembly(cmd, "compiled-prompt", 1, []model.Message{{Role: "user", Content: "compiled-prompt"}}, specs)

	if len(assembly.LocalTools) == 0 {
		t.Fatalf("expected local tool descriptors in skill assembly")
	}
	if assembly.Messages[0].Role != "system" {
		t.Fatalf("expected system prompt to be first skill assembly message, got %#v", assembly.Messages[0])
	}
	names := make([]string, 0, len(assembly.LocalTools))
	for _, tool := range assembly.LocalTools {
		names = append(names, tool.Name)
	}
	if !containsString(names, "Read") || !containsString(names, "PowerShell") {
		t.Fatalf("expected skill assembly local tools to include file and shell tools, got %v", names)
	}
}

func TestSkillDocumentHintIncludesProjectID(t *testing.T) {
	rt := newTestRuntime(t)
	cmd, err := rt.Registry.LoadInvocationCommand("webnovel-opening-sniper")
	if err != nil {
		t.Fatalf("LoadInvocationCommand failed: %v", err)
	}
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     rt.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: filepath.Join(rt.Config.Runtime.WorkspaceRoot, "docs", "generated"),
		ProjectID:         "case-file",
	}, store, "skill-calls/test")

	hint := skillDocumentHint(cmd, session)
	if !strings.Contains(hint, "active_project_id: case-file") {
		t.Fatalf("expected active project id in hint, got %q", hint)
	}
	if strings.Contains(hint, "allowed_local_tools: Read, Write, Edit, Glob, Bash, PowerShell, WriteProjectDocument") {
		t.Fatalf("did not expect project document tool to be exposed when provider is absent, got %q", hint)
	}
}

func TestProjectDocumentToolIsBuiltInWhenProjectWriterExists(t *testing.T) {
	rt := newTestRuntime(t)
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	provider := &recordingProjectDocumentProvider{}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     rt.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: rt.Config.Runtime.DocumentOutputDir,
		ProjectID:         "case-file",
		ProjectDocs:       provider,
	}, store, "skill-calls/test")

	specs := session.toolSpecs(nil)
	names := toolSpecNames(specs)
	if !containsString(names, skillListProjectDocumentsToolName) {
		t.Fatalf("expected built-in project document list tool, got %v", names)
	}
	if !containsString(names, skillReadProjectDocumentToolName) {
		t.Fatalf("expected built-in project document read tool, got %v", names)
	}
	if !containsString(names, skillWriteProjectDocumentToolName) {
		t.Fatalf("expected built-in project document write tool, got %v", names)
	}
	hint := skillDocumentHint(skill.Command{}, session)
	if !strings.Contains(hint, "WriteProjectDocument") || !strings.Contains(hint, "project document provider") {
		t.Fatalf("expected project document hint, got %q", hint)
	}

	args, _ := json.Marshal(map[string]any{
		"kind":  "novel_core",
		"title": "小说情感内核",
		"body":  "# 小说情感内核\n\n读者想要被看见。",
	})
	content, err := session.handleToolCallWithContext(context.Background(), model.ToolCall{
		ID:   "call_project_doc_1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillWriteProjectDocumentToolName,
			Arguments: string(args),
		},
	})
	if err != nil {
		t.Fatalf("expected project document write to succeed: %v", err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("expected one project document write, got %#v", provider.requests)
	}
	if provider.requests[0].ProjectID != "case-file" || provider.requests[0].Kind != "novel_core" {
		t.Fatalf("unexpected write request: %#v", provider.requests[0])
	}
	if !strings.Contains(content, `"synced": true`) {
		t.Fatalf("expected tool payload to report sync, got %s", content)
	}

	listContent, err := session.handleToolCallWithContext(context.Background(), model.ToolCall{
		ID:   "call_project_doc_2",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillListProjectDocumentsToolName,
			Arguments: `{}`,
		},
	})
	if err != nil {
		t.Fatalf("expected project document list to succeed: %v", err)
	}
	if !strings.Contains(listContent, `"count": 1`) || !strings.Contains(listContent, `"novel_core"`) {
		t.Fatalf("expected list payload to include written doc, got %s", listContent)
	}

	readContent, err := session.handleToolCallWithContext(context.Background(), model.ToolCall{
		ID:   "call_project_doc_3",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillReadProjectDocumentToolName,
			Arguments: `{"kind":"novel_core"}`,
		},
	})
	if err != nil {
		t.Fatalf("expected project document read to succeed: %v", err)
	}
	if !strings.Contains(readContent, "读者想要被看见") {
		t.Fatalf("expected read payload to include document body, got %s", readContent)
	}
}

func TestSkillContextHintInjectsProjectContext(t *testing.T) {
	rt := newTestRuntime(t)
	cmd, err := rt.Registry.LoadInvocationCommand("webnovel-opening-sniper")
	if err != nil {
		t.Fatalf("LoadInvocationCommand failed: %v", err)
	}
	rt.ProjectID = "case-file"
	rt.ProjectContext = "# Active Novel Project Context\n\n- project_id: case-file\n\n## 世界规则\n\n遗物会保留死者最后三分钟的执念。"

	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     rt.Config.Runtime.WorkspaceRoot,
		DocumentOutputDir: rt.Config.Runtime.DocumentOutputDir,
		ProjectID:         rt.ProjectID,
	}, rt.Store, "skill-calls/test")

	hint := rt.skillContextHint(cmd, session)
	if !strings.Contains(hint, "Active Novel Project Context") {
		t.Fatalf("expected injected project context, got %q", hint)
	}
	if !strings.Contains(hint, "遗物会保留死者最后三分钟的执念") {
		t.Fatalf("expected canon file content in hint, got %q", hint)
	}
}

func TestSkillFileWriteRequiresReadForExistingFile(t *testing.T) {
	workspaceRoot := t.TempDir()
	filePath := filepath.Join(workspaceRoot, "docs", "sample.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     workspaceRoot,
		DocumentOutputDir: filepath.Join(workspaceRoot, "docs", "generated"),
	}, store, "skill-calls/test-skill")

	writeArgs, _ := json.Marshal(map[string]any{
		"file_path": "docs/sample.md",
		"content":   "new",
	})
	if _, err := session.handleToolCall(model.ToolCall{
		ID:   "call_write_1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillWriteToolName,
			Arguments: string(writeArgs),
		},
	}); err == nil {
		t.Fatalf("expected existing file overwrite to require prior full read")
	}

	readArgs, _ := json.Marshal(map[string]any{
		"file_path": "docs/sample.md",
	})
	if _, err := session.handleToolCall(model.ToolCall{
		ID:   "call_read_1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillReadToolName,
			Arguments: string(readArgs),
		},
	}); err != nil {
		t.Fatalf("expected full read to succeed: %v", err)
	}

	if _, err := session.handleToolCall(model.ToolCall{
		ID:   "call_write_2",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillWriteToolName,
			Arguments: string(writeArgs),
		},
	}); err != nil {
		t.Fatalf("expected write after full read to succeed: %v", err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("expected file content to be updated, got %q", string(got))
	}
}

func TestSkillToolSpecsIncludeShellToolsWhenAllowed(t *testing.T) {
	workspaceRoot := t.TempDir()
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     workspaceRoot,
		DocumentOutputDir: filepath.Join(workspaceRoot, "docs", "generated"),
	}, store, "skill-calls/test-skill")

	specs := session.toolSpecs([]string{"Read", "Write", "Edit", "Glob", "Bash", "PowerShell"})
	names := toolSpecNames(specs)
	want := []string{"Read", "Write", "Edit", "Glob", "Bash", "PowerShell"}
	for _, expected := range want {
		if !containsString(names, expected) {
			t.Fatalf("expected tool spec %q to be present, got %v", expected, names)
		}
	}
	descriptions := map[string]string{}
	for _, spec := range specs {
		descriptions[spec.Function.Name] = spec.Function.Description
	}
	if !strings.Contains(descriptions["Bash"], "Glob") || !strings.Contains(descriptions["Bash"], "Read") {
		t.Fatalf("expected Bash description to steer model toward file tools, got %q", descriptions["Bash"])
	}
	if !strings.Contains(descriptions["PowerShell"], "Write") || !strings.Contains(descriptions["PowerShell"], "Edit") {
		t.Fatalf("expected PowerShell description to steer model toward file tools, got %q", descriptions["PowerShell"])
	}
}

func TestPowerShellToolCallRunsCommand(t *testing.T) {
	if _, _, err := resolvePowerShellCommand(); err != nil {
		t.Skipf("powershell is unavailable in this environment: %v", err)
	}

	workspaceRoot := t.TempDir()
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	session := newSkillFileToolSession(RuntimeConfigView{
		WorkspaceRoot:     workspaceRoot,
		DocumentOutputDir: filepath.Join(workspaceRoot, "docs", "generated"),
	}, store, "skill-calls/test-skill")

	args, _ := json.Marshal(map[string]any{
		"command":    `Write-Output "hello-from-powershell"`,
		"timeout_ms": 5000,
	})
	content, err := session.handleToolCall(model.ToolCall{
		ID:   "call_pwsh_1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      skillPowerShellToolName,
			Arguments: string(args),
		},
	})
	if err != nil {
		t.Fatalf("expected PowerShell command to succeed: %v", err)
	}
	var payload struct {
		Shell   string `json:"shell"`
		Stdout  string `json:"stdout"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("expected valid json payload, got %q: %v", content, err)
	}
	if payload.Shell != skillPowerShellToolName {
		t.Fatalf("expected shell name %q, got %q", skillPowerShellToolName, payload.Shell)
	}
	if !payload.Success {
		t.Fatalf("expected powershell payload to report success, got %q", content)
	}
	if !strings.Contains(payload.Stdout, "hello-from-powershell") {
		t.Fatalf("expected stdout to contain command output, got %q", payload.Stdout)
	}
}

func TestExecuteSkillCanPersistDocumentWithWriteTool(t *testing.T) {
	rt := newTestRuntime(t)
	t.Setenv("TEST_API_KEY", "dummy")

	var requestCount atomic.Int32
	expectedToolName := activeSkillToolName("webnovel-opening-sniper")
	expectedDocPath := filepath.Join(rt.Config.Runtime.WorkspaceRoot, "docs", "generated", "opening.md")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := requestCount.Add(1)
		var req model.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")

		switch current {
		case 1:
			_, _ = fmt.Fprint(w, `{"id":"resp-1","choices":[{"index":0,"message":{"role":"assistant","content":"Searching skills","reasoning_content":"Need tool search first","tool_calls":[{"id":"call_1","type":"function","function":{"name":"tool_search","arguments":"{\"query\":\"urban opening\",\"limit\":3}"}}]},"finish_reason":"tool_calls"}]}`)
		case 2:
			_, _ = fmt.Fprintf(w, `{"id":"resp-2","choices":[{"index":0,"message":{"role":"assistant","content":"Calling skill","reasoning_content":"Use the activated skill tool","tool_calls":[{"id":"call_2","type":"function","function":{"name":"%s","arguments":"{\"task\":\"write opening\",\"premise\":\"corpse report reveals hidden data\",\"document_path\":\"docs/generated/opening.md\"}"}}]},"finish_reason":"tool_calls"}]}`, expectedToolName)
		case 3:
			_, _ = fmt.Fprint(w, `{"id":"resp-3","choices":[{"index":0,"message":{"role":"assistant","content":"Persisting opening draft","tool_calls":[{"id":"call_3","type":"function","function":{"name":"Write","arguments":"{\"file_path\":\"docs/generated/opening.md\",\"content\":\"# 都市异能开篇\\n\\n法医室的灯刚亮，尸检报告上的第二组血氧数据忽然自己跳了一下。\"}"}}]},"finish_reason":"tool_calls"}]}`)
		case 4:
			_, _ = fmt.Fprint(w, `{"id":"resp-4","choices":[{"index":0,"message":{"role":"assistant","content":"已写入 docs/generated/opening.md，这一版的核心钩子是尸检报告里的异常数据会主动向主角泄密。"},"finish_reason":"stop"}]}`)
		default:
			t.Fatalf("unexpected extra request #%d with tools %#v", current, req.Tools)
		}
	}))
	defer server.Close()

	rt.ModelConfig.BaseURL = server.URL
	rt.ModelConfig.APIKeyEnv = "TEST_API_KEY"
	rt.Model = model.NewOpenAICompatible(server.URL, "TEST_API_KEY", "test-model", 5)

	res, err := rt.Run(context.Background(), "写一个都市异能悬疑开篇，并把成品落到文档里")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := requestCount.Load(); got != 4 {
		t.Fatalf("expected router search + router skill call + skill write roundtrip, got %d requests", got)
	}
	if !strings.Contains(res.FinalText, "docs/generated/opening.md") {
		t.Fatalf("expected final text to mention persisted document path, got %q", res.FinalText)
	}
	content, err := os.ReadFile(expectedDocPath)
	if err != nil {
		t.Fatalf("expected persisted document to exist: %v", err)
	}
	if !strings.Contains(string(content), "尸检报告上的第二组血氧数据忽然自己跳了一下") {
		t.Fatalf("expected persisted document content, got %q", string(content))
	}
}

type recordingProjectDocumentProvider struct {
	requests []ProjectDocumentWriteRequest
}

func (p *recordingProjectDocumentProvider) ListProjectDocuments(_ context.Context, projectID string) ([]ProjectDocumentSummary, error) {
	out := make([]ProjectDocumentSummary, 0, len(p.requests))
	for _, req := range p.requests {
		if req.ProjectID != projectID {
			continue
		}
		out = append(out, ProjectDocumentSummary{
			ProjectID: req.ProjectID,
			Kind:      req.Kind,
			Title:     req.Title,
			BodyBytes: len(req.Body),
		})
	}
	return out, nil
}

func (p *recordingProjectDocumentProvider) ReadProjectDocument(_ context.Context, projectID, kind string) (ProjectDocumentReadResult, error) {
	for _, req := range p.requests {
		if req.ProjectID == projectID && req.Kind == kind {
			return ProjectDocumentReadResult{
				ProjectID: req.ProjectID,
				Kind:      req.Kind,
				Title:     req.Title,
				Body:      req.Body,
				Metadata:  req.Metadata,
			}, nil
		}
	}
	return ProjectDocumentReadResult{}, fmt.Errorf("project document %q not found", kind)
}

func (p *recordingProjectDocumentProvider) WriteProjectDocument(_ context.Context, req ProjectDocumentWriteRequest) (ProjectDocumentWriteResult, error) {
	p.requests = append(p.requests, req)
	return ProjectDocumentWriteResult{
		ProjectID: req.ProjectID,
		Kind:      req.Kind,
		Title:     req.Title,
		BodyBytes: len(req.Body),
		Synced:    true,
	}, nil
}

func newTestRuntime(t *testing.T) *Runtime {
	t.Helper()

	skillsDir := t.TempDir()
	workspaceRoot := t.TempDir()
	writeSkill := func(id, content string) {
		skillDir := filepath.Join(skillsDir, id)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	writeSkill("webnovel-opening-sniper", `---
name: Opening Sniper
description: Writes a strong 600-word opening
when_to_use: Use when the user asks for an urban power novel opening
search_hint: urban opening hook
allowed_tools:
  - Read
  - Write
  - Edit
  - Glob
  - Bash
  - PowerShell
tool_description: Turn premise and hooks into a publishable opening.
tool_contract: opening_v1
tool_output_contract: opening_prose_v1
tool_input_schema:
  type: object
  properties:
    task:
      type: string
      description: Concrete opening request.
    premise:
      type: string
      description: Story setup or hook to anchor the opening.
    protagonist:
      type: string
      description: Main character identity and current state.
    document_path:
      type: string
      description: Optional workspace-relative markdown file path for the final opening.
  required:
    - task
tags:
  - novel
  - opening
---
Body`)
	writeSkill("second-opening-skill", `---
name: Second Skill
description: Another opening specialist
when_to_use: Use for opening scenes with hooks
search_hint: opening hook
tags:
  - opening
---
Body`)
	writeSkill("outline-builder", `---
name: Outline Builder
description: Generates outlines
tags:
  - outline
---
Body`)

	reg, err := skill.LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("runstore.New failed: %v", err)
	}
	return &Runtime{
		Config: config.Config{
			Runtime: config.RuntimeConfig{
				WorkspaceRoot:        workspaceRoot,
				DocumentOutputDir:    filepath.Join(workspaceRoot, "docs", "generated"),
				ForceToolSearchFirst: true,
				MaxToolRounds:        4,
				MaxSkillToolRounds:   6,
				ReturnSkillDirect:    true,
				MaxActivatedSkills:   3,
				MaxRetainedSkills:    6,
				ActivationMinScore:   0.18,
				ActivationScoreRatio: 0.55,
			},
		},
		ModelConfig: ModelConfig{ID: "test-model", BaseURL: "http://example.invalid", APIKeyEnv: "TEST_API_KEY", MaxOutput: 4096, Temperature: 0.7, TimeoutSeconds: 5},
		Registry:    reg,
		Store:       store,
	}
}

func toolSpecNames(specs []model.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec.Function.Name)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
