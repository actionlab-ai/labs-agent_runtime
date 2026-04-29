package project

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const maxContextChars = 24000
const NovelCoreKind = "novel_core"

type Project struct {
	ID          string
	Name        string
	Description string
	Status      string
}

type Document struct {
	Kind  string
	Title string
	Body  string
}

func Slug(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r) || r == '/' || r == '\\' || r == '.':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		return "novel-project"
	}
	return id
}

func BuildContext(p Project, docs []Document) string {
	docs = orderContextDocuments(docs)
	hasNovelCore := hasDocumentKind(docs, NovelCoreKind)

	var b strings.Builder
	b.WriteString("# Active Novel Project Context\n\n")
	b.WriteString(fmt.Sprintf("- project_id: %s\n", p.ID))
	if strings.TrimSpace(p.Name) != "" {
		b.WriteString(fmt.Sprintf("- project_name: %s\n", p.Name))
	}
	if strings.TrimSpace(p.Description) != "" {
		b.WriteString(fmt.Sprintf("- description: %s\n", p.Description))
	}
	if strings.TrimSpace(p.Status) != "" {
		b.WriteString(fmt.Sprintf("- status: %s\n", p.Status))
	}
	b.WriteString("\n## Project Asset Policy\n\n")
	b.WriteString("- always_load: novel_core\n")
	if hasNovelCore {
		b.WriteString("- novel_core_status: present\n")
		b.WriteString("- rule: Treat novel_core as mandatory canon for every downstream creative task. Do not contradict its emotional promise unless the user explicitly asks to revise novel_core.\n")
	} else {
		b.WriteString("- novel_core_status: missing\n")
		b.WriteString("- rule: The project does not have a saved emotional core yet. If the task depends on story direction, create or ask for novel_core first.\n")
	}
	b.WriteString("\nUse this project context as the baseline for this session. Treat the database-backed documents below as current canon. If a key fact is missing, ask for it or return a proposal that can be saved back into project documents.\n")

	written := len([]rune(b.String()))
	for _, doc := range docs {
		body := strings.TrimSpace(doc.Body)
		if body == "" {
			continue
		}
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			title = doc.Kind
		}
		section := fmt.Sprintf("\n\n## %s (%s)\n\n%s", title, doc.Kind, body)
		if written+len([]rune(section)) > maxContextChars {
			remaining := maxContextChars - written
			if remaining <= 0 {
				break
			}
			section = truncateRunes(section, remaining)
		}
		b.WriteString(section)
		written += len([]rune(section))
		if written >= maxContextChars {
			break
		}
	}
	return b.String()
}

func orderContextDocuments(docs []Document) []Document {
	out := append([]Document(nil), docs...)
	sort.SliceStable(out, func(i, j int) bool {
		left := contextDocumentPriority(out[i].Kind)
		right := contextDocumentPriority(out[j].Kind)
		if left != right {
			return left < right
		}
		return strings.TrimSpace(out[i].Kind) < strings.TrimSpace(out[j].Kind)
	})
	return out
}

func hasDocumentKind(docs []Document, kind string) bool {
	kind = normalizeDocumentKind(kind)
	for _, doc := range docs {
		if normalizeDocumentKind(doc.Kind) == kind && strings.TrimSpace(doc.Body) != "" {
			return true
		}
	}
	return false
}

func contextDocumentPriority(kind string) int {
	switch normalizeDocumentKind(kind) {
	case NovelCoreKind:
		return 0
	case "project_brief":
		return 10
	case "reader_contract":
		return 20
	case "style_guide":
		return 30
	case "taboo":
		return 40
	case "world_rules":
		return 50
	case "power_system":
		return 60
	case "factions":
		return 70
	case "locations":
		return 80
	case "mainline":
		return 90
	case "current_state":
		return 100
	default:
		return 1000
	}
}

func truncateRunes(text string, maxChars int) string {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	if maxChars <= 20 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-16]) + "\n\n...[truncated]"
}
