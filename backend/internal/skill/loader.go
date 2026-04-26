package skill

import (
	"bufio"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadSkillMetadata(entryPath string) (map[string]any, string, int, error) {
	f, err := os.Open(entryPath)
	if err != nil {
		return nil, "", 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	data := map[string]any{}
	var yamlLines []string
	var paragraph strings.Builder
	bodyStarted := false
	inFrontmatter := false
	frontmatterDone := false
	bodyLength := 0

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case !bodyStarted:
			bodyStarted = true
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = true
				continue
			}
			appendBodyLine(line, &paragraph, &bodyLength)
		case inFrontmatter:
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				frontmatterDone = true
				continue
			}
			yamlLines = append(yamlLines, line)
		default:
			appendBodyLine(line, &paragraph, &bodyLength)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", 0, err
	}

	if bodyStarted && inFrontmatter {
		return map[string]any{}, strings.TrimSpace(paragraph.String()), bodyLength, nil
	}
	if frontmatterDone && len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &data); err != nil {
			return nil, "", 0, err
		}
	}

	return data, trimSummary(paragraph.String()), bodyLength, nil
}

func loadSkillBody(entryPath string) (string, error) {
	b, err := os.ReadFile(entryPath)
	if err != nil {
		return "", err
	}
	fm, err := SplitFrontmatter(string(b))
	if err != nil {
		return "", err
	}
	return fm.Body, nil
}

func appendBodyLine(line string, paragraph *strings.Builder, bodyLength *int) {
	*bodyLength += len(line) + 1
	if paragraph == nil {
		return
	}
	trimmed := strings.TrimSpace(strings.ReplaceAll(line, "#", ""))
	if trimmed == "" || paragraph.Len() >= 220 {
		return
	}
	if paragraph.Len() > 0 {
		paragraph.WriteByte(' ')
	}
	paragraph.WriteString(trimmed)
}

func trimSummary(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 220 {
		return s[:220]
	}
	return s
}
