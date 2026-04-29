package project

import (
	"strings"
	"testing"
)

func TestSlugKeepsReadableProjectIDs(t *testing.T) {
	if got := Slug("都市 异能/悬疑"); got != "都市-异能-悬疑" {
		t.Fatalf("unexpected slug: %q", got)
	}
}

func TestSlugFallsBackForPunctuationOnlyName(t *testing.T) {
	if got := Slug("/// ..."); got != "novel-project" {
		t.Fatalf("expected fallback slug, got %q", got)
	}
}

func TestBuildContextIncludesProjectAndDocuments(t *testing.T) {
	text := BuildContext(Project{
		ID:          "case-file",
		Name:        "Case File",
		Description: "遗物悬疑",
		Status:      "active",
	}, []Document{{
		Kind:  "world_rules",
		Title: "世界规则",
		Body:  "遗物会保留死者最后三分钟的执念。",
	}})

	if !strings.Contains(text, "project_id: case-file") {
		t.Fatalf("expected project id, got %q", text)
	}
	if !strings.Contains(text, "遗物会保留死者最后三分钟的执念") {
		t.Fatalf("expected document body, got %q", text)
	}
}

func TestBuildContextAlwaysPrioritizesNovelCore(t *testing.T) {
	text := BuildContext(Project{
		ID:     "case-file",
		Name:   "Case File",
		Status: "active",
	}, []Document{
		{Kind: "world_rules", Title: "World Rules", Body: strings.Repeat("world ", 6000)},
		{Kind: "world_engine", Title: "World Engine", Body: "World pressure engine."},
		{Kind: "novel_core", Title: "Novel Core", Body: "Core promise: regain dignity."},
	})

	if !strings.Contains(text, "novel_core_status: present") {
		t.Fatalf("expected novel_core to be marked present, got %q", text)
	}
	corePos := strings.Index(text, "Novel Core (novel_core)")
	worldPos := strings.Index(text, "World Rules (world_rules)")
	if corePos == -1 {
		t.Fatalf("expected novel_core body in context, got %q", text)
	}
	if worldPos != -1 && worldPos < corePos {
		t.Fatalf("expected novel_core before world_rules, got %q", text)
	}
	if !strings.Contains(text, "Core promise: regain dignity.") {
		t.Fatalf("expected novel_core content to survive truncation, got %q", text)
	}
}

func TestBuildContextOrdersWorldEngineBeforeWorldDetails(t *testing.T) {
	text := BuildContext(Project{
		ID:     "case-file",
		Status: "active",
	}, []Document{
		{Kind: "world_rules", Title: "World Rules", Body: "World rules."},
		{Kind: "power_system", Title: "Power System", Body: "Power system."},
		{Kind: "world_engine", Title: "World Engine", Body: "World pressure engine."},
		{Kind: "novel_core", Title: "Novel Core", Body: "Core promise."},
	})

	corePos := strings.Index(text, "Novel Core (novel_core)")
	enginePos := strings.Index(text, "World Engine (world_engine)")
	rulesPos := strings.Index(text, "World Rules (world_rules)")
	powerPos := strings.Index(text, "Power System (power_system)")
	if corePos == -1 || enginePos == -1 || rulesPos == -1 || powerPos == -1 {
		t.Fatalf("expected all documents in context, got %q", text)
	}
	if !(corePos < enginePos && enginePos < rulesPos && rulesPos < powerPos) {
		t.Fatalf("unexpected context order: core=%d engine=%d rules=%d power=%d\n%s", corePos, enginePos, rulesPos, powerPos, text)
	}
}

func TestBuildContextUsesConfiguredDocumentPolicy(t *testing.T) {
	policy, err := ParseDocumentPolicy(`{
	  "documents": [
	    {"kind":"novel_core","title":"Core","priority":0},
	    {"kind":"arc_plan","title":"Arc Plan","priority":5},
	    {"kind":"world_engine","title":"World Engine","priority":10}
	  ],
	  "skill_documents": {
	    "arc-skill": ["novel_core", "arc_plan"]
	  }
	}`)
	if err != nil {
		t.Fatalf("ParseDocumentPolicy failed: %v", err)
	}
	text := BuildContextWithPolicy(Project{ID: "case-file"}, []Document{
		{Kind: "world_engine", Title: "World Engine", Body: "World pressure."},
		{Kind: "arc_plan", Title: "Arc Plan", Body: "Arc plan."},
		{Kind: "novel_core", Title: "Core", Body: "Core promise."},
	}, policy)

	corePos := strings.Index(text, "Core (novel_core)")
	arcPos := strings.Index(text, "Arc Plan (arc_plan)")
	enginePos := strings.Index(text, "World Engine (world_engine)")
	if corePos == -1 || arcPos == -1 || enginePos == -1 {
		t.Fatalf("expected configured documents in context, got %q", text)
	}
	if !(corePos < arcPos && arcPos < enginePos) {
		t.Fatalf("unexpected configured order: core=%d arc=%d engine=%d\n%s", corePos, arcPos, enginePos, text)
	}
	if got := strings.Join(policy.RequiredDocumentsForSkill("arc-skill"), ","); got != "novel_core,arc_plan" {
		t.Fatalf("unexpected skill required docs: %q", got)
	}
}

func TestDocumentPolicyDoesNotReAddDefaultSkillDocumentsWhenConfiguredEmpty(t *testing.T) {
	policy, err := ParseDocumentPolicy(`{
	  "documents": [
	    {"kind":"novel_core","title":"Core","priority":0}
	  ],
	  "skill_documents": {}
	}`)
	if err != nil {
		t.Fatalf("ParseDocumentPolicy failed: %v", err)
	}
	if got := policy.RequiredDocumentsForSkill("novel-world-engine"); len(got) != 0 {
		t.Fatalf("expected configured empty skill documents to stay empty, got %#v", got)
	}
}

func TestBuildContextMarksMissingNovelCore(t *testing.T) {
	text := BuildContext(Project{ID: "case-file"}, []Document{
		{Kind: "world_rules", Title: "World Rules", Body: "rules"},
	})
	if !strings.Contains(text, "novel_core_status: missing") {
		t.Fatalf("expected missing novel_core status, got %q", text)
	}
}

func TestExtractDocumentDraftsParsesStructuredWorkflowOutput(t *testing.T) {
	drafts := ExtractDocumentDrafts(`## project_brief

项目一句话定位。

## reader_contract

读者会在前二十章拿到稳定的安全感反馈。

## world_engine

世界持续压迫主角，同时保留破局口。

## ignored_section

这个分节不应该被持久化。

## taboo

不要苦大仇深。`)

	if len(drafts) != 4 {
		t.Fatalf("expected four persistable drafts, got %#v", drafts)
	}
	if drafts[0].Kind != "project_brief" || drafts[1].Kind != "reader_contract" || drafts[2].Kind != "world_engine" || drafts[3].Kind != "taboo" {
		t.Fatalf("unexpected draft kinds: %#v", drafts)
	}
	if drafts[0].Title != "项目简报" || drafts[1].Title != "读者承诺" || drafts[2].Title != "小说世界压力引擎" || drafts[3].Title != "禁区与避坑" {
		t.Fatalf("unexpected default titles: %#v", drafts)
	}
}
