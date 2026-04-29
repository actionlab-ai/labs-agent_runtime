package project

import "strings"

func PersistableDocumentKinds() []string {
	return DefaultDocumentPolicy().Kinds()
}

func IsPersistableDocumentKind(kind string) bool {
	return DefaultDocumentPolicy().IsPersistable(kind)
}

func DefaultDocumentTitle(kind string) string {
	return DefaultDocumentPolicy().Title(kind)
}

func ExtractDocumentDrafts(text string) []Document {
	return ExtractDocumentDraftsWithPolicy(text, DefaultDocumentPolicy())
}

func ExtractDocumentDraftsWithPolicy(text string, policy DocumentPolicy) []Document {
	policy = policy.Normalize()
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
				Title: policy.Title(currentKind),
				Body:  body,
			})
		}
		currentKind = ""
		currentLines = nil
	}

	for _, line := range lines {
		if kind, ok := parseDocumentHeadingWithPolicy(line, policy); ok {
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
	return parseDocumentHeadingWithPolicy(line, DefaultDocumentPolicy())
}

func parseDocumentHeadingWithPolicy(line string, policy DocumentPolicy) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "# ") {
		return "", false
	}
	heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
	heading = strings.Trim(heading, "`")
	kind := normalizeDocumentKind(heading)
	if !policy.IsPersistable(kind) {
		return "", false
	}
	return kind, true
}

func normalizeDocumentKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
