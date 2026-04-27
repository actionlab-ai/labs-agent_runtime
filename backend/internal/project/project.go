package project

import (
	"fmt"
	"strings"
	"unicode"
)

const maxContextChars = 24000

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
