package project

import "strings"

var persistableDocumentKinds = []string{
	"novel_core",
	"project_brief",
	"reader_contract",
	"style_guide",
	"taboo",
	"world_rules",
	"power_system",
	"factions",
	"locations",
	"mainline",
	"current_state",
}

var defaultDocumentTitles = map[string]string{
	"novel_core":      "小说情感内核",
	"project_brief":   "项目简报",
	"reader_contract": "读者承诺",
	"style_guide":     "风格指南",
	"taboo":           "禁区与避坑",
	"world_rules":     "世界规则",
	"power_system":    "能力体系",
	"factions":        "势力关系",
	"locations":       "地点设定",
	"mainline":        "主线规划",
	"current_state":   "当前状态",
}

func PersistableDocumentKinds() []string {
	out := make([]string, 0, len(persistableDocumentKinds))
	out = append(out, persistableDocumentKinds...)
	return out
}

func IsPersistableDocumentKind(kind string) bool {
	kind = normalizeDocumentKind(kind)
	for _, candidate := range persistableDocumentKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

func DefaultDocumentTitle(kind string) string {
	kind = normalizeDocumentKind(kind)
	if title := strings.TrimSpace(defaultDocumentTitles[kind]); title != "" {
		return title
	}
	return kind
}

func ExtractDocumentDrafts(text string) []Document {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	var out []Document
	currentKind := ""
	var currentLines []string
	flush := func() {
		if currentKind == "" {
			return
		}
		body := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if body != "" {
			out = append(out, Document{
				Kind:  currentKind,
				Title: DefaultDocumentTitle(currentKind),
				Body:  body,
			})
		}
		currentKind = ""
		currentLines = nil
	}

	for _, line := range lines {
		if kind, ok := parseDocumentHeading(line); ok {
			flush()
			currentKind = kind
			currentLines = nil
			continue
		}
		if currentKind != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	return out
}

func parseDocumentHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "# ") {
		return "", false
	}
	heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
	heading = strings.Trim(heading, "`")
	kind := normalizeDocumentKind(heading)
	if !IsPersistableDocumentKind(kind) {
		return "", false
	}
	return kind, true
}

func normalizeDocumentKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
