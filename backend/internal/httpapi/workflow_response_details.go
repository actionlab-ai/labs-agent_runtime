package httpapi

import (
	"strings"
)

func buildWorkflowResponseDetails(plan fixedWorkflowPlan, persistence workflowPersistenceResult, rawText string) (string, *workflowClarification) {
	finalText := normalizeWorkflowFinalText(plan, persistence.ResponseMode, rawText)
	if persistence.ResponseMode != workflowResponseModeClarification {
		return finalText, nil
	}
	return finalText, parseWorkflowClarification(plan, finalText)
}

func normalizeWorkflowFinalText(plan fixedWorkflowPlan, responseMode, rawText string) string {
	rawText = normalizeMarkdownText(rawText)
	if rawText == "" {
		return ""
	}

	switch responseMode {
	case workflowResponseModeClarification:
		if extracted := extractMarkdownBlockFromHeading(rawText, plan.ClarifyHeading); extracted != "" {
			return extracted
		}
	case workflowResponseModeDocument:
		if extracted := extractMarkdownBlockFromHeading(rawText, firstNonEmpty(plan.PersistHeading, plan.PersistTitle)); extracted != "" {
			return extracted
		}
	}

	return rawText
}

func parseWorkflowClarification(plan fixedWorkflowPlan, text string) *workflowClarification {
	text = normalizeMarkdownText(text)
	if text == "" {
		return nil
	}

	if extracted := extractMarkdownBlockFromHeading(text, plan.ClarifyHeading); extracted != "" {
		text = extracted
	}
	sections := splitMarkdownSections(text)

	out := &workflowClarification{
		MissingFields:   uniquePreserveOrder(parseSimpleList(sections["缺失字段"])),
		ConfirmedFacts:  parseSimpleList(sections["当前已确认"]),
		BlockingReasons: parseSimpleList(sections["还不能定稿的原因"]),
		Questions:       parseClarificationQuestions(sections["请先回答以下问题"]),
	}
	if len(out.MissingFields) == 0 {
		for _, q := range out.Questions {
			if field := normalizeClarificationField(q.Field); field != "" {
				out.MissingFields = append(out.MissingFields, field)
			}
		}
		out.MissingFields = uniquePreserveOrder(out.MissingFields)
	}

	if len(out.MissingFields) == 0 && len(out.ConfirmedFacts) == 0 && len(out.BlockingReasons) == 0 && len(out.Questions) == 0 {
		return nil
	}
	return out
}

func splitMarkdownSections(text string) map[string]string {
	text = normalizeMarkdownText(text)
	lines := strings.Split(text, "\n")
	sections := map[string]string{}
	currentHeading := ""
	var currentLines []string

	flush := func() {
		if currentHeading == "" {
			return
		}
		sections[currentHeading] = strings.TrimSpace(strings.Join(currentLines, "\n"))
		currentHeading = ""
		currentLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			currentHeading = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if currentHeading != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	return sections
}

func parseSimpleList(text string) []string {
	text = normalizeMarkdownText(text)
	if text == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "- "):
			out = append(out, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		case hasOrderedListPrefix(trimmed):
			out = append(out, trimOrderedListPrefix(trimmed))
		}
	}
	return out
}

func parseClarificationQuestions(text string) []workflowClarificationQuestion {
	text = normalizeMarkdownText(text)
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var out []workflowClarificationQuestion
	current := workflowClarificationQuestion{}

	flush := func() {
		if strings.TrimSpace(current.Prompt) == "" {
			return
		}
		current.Field = normalizeClarificationField(current.Field)
		current.Options = uniquePreserveOrder(current.Options)
		out = append(out, current)
		current = workflowClarificationQuestion{}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "### "):
			flush()
			current.Field, current.Prompt = parseClarificationQuestionHeading(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))
		case hasOrderedListPrefix(trimmed):
			flush()
			current.Prompt = trimOrderedListPrefix(trimmed)
		case strings.HasPrefix(trimmed, "- "):
			option := normalizeClarificationOption(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			if option != "" {
				current.Options = append(current.Options, option)
			}
		}
	}
	flush()

	return out
}

func parseClarificationQuestionHeading(heading string) (string, string) {
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return "", ""
	}
	for _, sep := range []string{"|", "｜", ":"} {
		if parts := strings.SplitN(heading, sep, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return "", heading
}

func normalizeClarificationOption(option string) string {
	option = strings.TrimSpace(strings.ReplaceAll(option, "**", ""))
	return option
}

func normalizeClarificationField(field string) string {
	field = strings.TrimSpace(strings.Trim(field, "`"))
	field = strings.TrimPrefix(field, "field=")
	field = strings.TrimSpace(field)
	return field
}

func hasOrderedListPrefix(line string) bool {
	if len(line) < 3 {
		return false
	}
	idx := strings.Index(line, ".")
	if idx <= 0 || idx > 3 {
		return false
	}
	for _, r := range line[:idx] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func trimOrderedListPrefix(line string) string {
	idx := strings.Index(line, ".")
	if idx == -1 {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(line[idx+1:])
}

func extractMarkdownBlockFromHeading(text, heading string) string {
	text = normalizeMarkdownText(text)
	target := normalizeWorkflowHeading(heading)
	if target == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		candidate := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if normalizeWorkflowHeading(candidate) == target {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:], "\n"))
}

func normalizeMarkdownText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
}

func uniquePreserveOrder(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
