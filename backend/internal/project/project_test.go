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
