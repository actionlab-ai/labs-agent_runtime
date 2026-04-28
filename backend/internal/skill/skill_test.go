package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRegistryDefersSkillBodyUntilInvocation(t *testing.T) {
	skillsDir := t.TempDir()
	writeTestSkill(t, skillsDir, "webnovel-opening-sniper", `---
name: Opening Sniper
description: Writes a strong 600-word opening
when_to_use: Use when the user asks for a novel opening
aliases:
  - opening-sniper
search_hint: forensic-report urban-power
tool_description: Turn premise and hooks into a publishable opening.
tool_contract: opening_v1
tool_output_contract: opening_prose_v1
tool_input_schema:
  type: object
  properties:
    task:
      type: string
    premise:
      type: string
  required:
    - task
tags:
  - novel
  - opening
---
# Skill Body

This is the real skill body.`)

	reg, err := LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	cmd, ok := reg.Get("webnovel-opening-sniper")
	if !ok {
		t.Fatalf("expected command to be loaded")
	}
	if cmd.MarkdownContent != "" {
		t.Fatalf("expected metadata load to defer body content")
	}
	if cmd.ContentLength == 0 {
		t.Fatalf("expected body length summary to be recorded")
	}
	if cmd.ToolContract != "opening_v1" {
		t.Fatalf("expected tool contract from frontmatter, got %#v", cmd.ToolContract)
	}
	if cmd.ToolOutput != "opening_prose_v1" {
		t.Fatalf("expected tool output contract from frontmatter, got %#v", cmd.ToolOutput)
	}
	if cmd.ToolInputSchema["type"] != "object" {
		t.Fatalf("expected tool input schema to load, got %#v", cmd.ToolInputSchema)
	}

	loaded, err := reg.LoadInvocationCommand("webnovel-opening-sniper")
	if err != nil {
		t.Fatalf("LoadInvocationCommand failed: %v", err)
	}
	if !strings.Contains(loaded.MarkdownContent, "This is the real skill body.") {
		t.Fatalf("expected full skill body to be loaded on invocation")
	}
}

func TestFrontmatterMapNormalizationSupportsToolSchema(t *testing.T) {
	data := map[string]any{
		"tool_input_schema": map[any]any{
			"type": "object",
			"properties": map[any]any{
				"task": map[any]any{
					"type": "string",
				},
			},
		},
	}

	schema := fmMap(data, "tool_input_schema")
	if schema["type"] != "object" {
		t.Fatalf("expected normalized schema type, got %#v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected normalized properties map, got %#v", schema["properties"])
	}
	task, ok := properties["task"].(map[string]any)
	if !ok || task["type"] != "string" {
		t.Fatalf("expected normalized nested schema, got %#v", properties["task"])
	}
}

func TestSearchSupportsSelectAndRequiredTerms(t *testing.T) {
	skillsDir := t.TempDir()
	writeTestSkill(t, skillsDir, "webnovel-opening-sniper", `---
name: Opening Sniper
description: Writes a strong 600-word opening
when_to_use: Use when the user asks for an urban power novel opening
aliases:
  - opening-sniper
search_hint: forensic-report urban-power
tags:
  - novel
  - opening
---
Body`)
	writeTestSkill(t, skillsDir, "generic-outline", `---
name: Outline Builder
description: Generates plot outlines
tags:
  - outline
---
Body`)

	reg, err := LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	selected := reg.Search("select:opening-sniper", 5)
	if len(selected) != 1 || selected[0].ID != "webnovel-opening-sniper" {
		t.Fatalf("expected select syntax to resolve by alias, got %#v", selected)
	}
	if !selected[0].Exact {
		t.Fatalf("expected select hit to be marked exact")
	}

	hits := reg.Search("+forensic urban opening", 5)
	if len(hits) == 0 {
		t.Fatalf("expected search hits for required terms")
	}
	if hits[0].ID != "webnovel-opening-sniper" {
		t.Fatalf("expected best hit to be opening skill, got %#v", hits[0])
	}
	if !strings.Contains(hits[0].Reason, "search-hint") && !strings.Contains(hits[0].Reason, "when-to-use") {
		t.Fatalf("expected reason to mention matched fields, got %q", hits[0].Reason)
	}
}

func TestSearchSupportsBareExactQuery(t *testing.T) {
	skillsDir := t.TempDir()
	writeTestSkill(t, skillsDir, "webnovel-opening-sniper", `---
name: Opening Sniper
aliases:
  - opening-sniper
---
Body`)

	reg, err := LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	hits := reg.Search("Opening Sniper", 5)
	if len(hits) != 1 {
		t.Fatalf("expected one exact hit, got %#v", hits)
	}
	if !hits[0].Exact {
		t.Fatalf("expected bare exact query to mark hit as exact")
	}
	if hits[0].MatchedFields[0] != "exact-name" {
		t.Fatalf("expected exact-name match, got %#v", hits[0].MatchedFields)
	}
}

func TestSearchRanksBootstrapSkillForIdeaQueries(t *testing.T) {
	skillsDir := t.TempDir()
	writeTestSkill(t, skillsDir, "webnovel-opening-sniper", `---
name: Opening Sniper
description: Writes a strong 600-word opening
when_to_use: Use when the user asks for an opening or first chapter
tags:
  - novel
  - opening
  - hook
---
Body`)
	writeTestSkill(t, skillsDir, "novel-idea-bootstrap", `---
name: Idea Bootstrapper
description: Turns a rough novel idea into worldview, power system, and early plot scaffolding
when_to_use: Use when the user gives a rough idea and wants worldbuilding or golden finger design
aliases:
  - world-power-designer
search_hint: idea worldbuilding setting golden finger 世界观 金手指 起盘 配角 主线
tags:
  - 小说
  - 世界观
  - 金手指
  - 创意
---
Body`)

	reg, err := LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	hits := reg.Search("帮我根据一个都市异能idea设计世界观和金手指", 5)
	if len(hits) == 0 {
		t.Fatalf("expected search hits")
	}
	if hits[0].ID != "novel-idea-bootstrap" {
		t.Fatalf("expected bootstrap skill to rank first, got %#v", hits[0])
	}
}

func TestSearchRanksEmotionalCoreSkillForNovelCoreQueries(t *testing.T) {
	skillsDir := t.TempDir()
	writeTestSkill(t, skillsDir, "novel-emotional-core", `---
name: Emotional Core Designer
description: Build the emotional core for a new webnovel before worldbuilding
when_to_use: Use when the user wants novel core, reader desire, recognition, catharsis, or emotional payoff
aliases:
  - emotional-core
  - novel-core
search_hint: emotional core novel core desire recognition catharsis 情感内核 小说内核 爽感 认可感 压迫 渴望
tags:
  - 情感内核
  - 新书内核
  - 爽感
---
Body`)
	writeTestSkill(t, skillsDir, "novel-idea-bootstrap", `---
name: Idea Bootstrapper
description: Turns a rough novel idea into worldview and power system
tags:
  - 世界观
  - 金手指
---
Body`)

	reg, err := LoadRegistry(skillsDir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	hits := reg.Search("我想先设计新书情感内核和中年男人被生活压迫后的爽感", 5)
	if len(hits) == 0 {
		t.Fatalf("expected search hits")
	}
	if hits[0].ID != "novel-emotional-core" {
		t.Fatalf("expected emotional core skill to rank first, got %#v", hits[0])
	}
}

func writeTestSkill(t *testing.T, skillsDir, skillID, content string) {
	t.Helper()
	skillDir := filepath.Join(skillsDir, skillID)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
