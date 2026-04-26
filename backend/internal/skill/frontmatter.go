package skill

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Data map[string]any
	Body string
}

func SplitFrontmatter(text string) (Frontmatter, error) {
	trim := strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(trim, "---\n") && !strings.HasPrefix(trim, "---\r\n") {
		return Frontmatter{Data: map[string]any{}, Body: text}, nil
	}
	lines := strings.Split(trim, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return Frontmatter{Data: map[string]any{}, Body: text}, nil
	}
	yamlText := strings.Join(lines[1:end], "\n")
	body := strings.Join(lines[end+1:], "\n")
	data := map[string]any{}
	if strings.TrimSpace(yamlText) != "" {
		if err := yaml.Unmarshal([]byte(yamlText), &data); err != nil {
			return Frontmatter{}, err
		}
	}
	return Frontmatter{Data: data, Body: strings.TrimSpace(body)}, nil
}

func fmString(data map[string]any, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func fmStringSlice(data map[string]any, key string) []string {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	var out []string
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, item := range x {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, part := range strings.Split(x, ",") {
			s := strings.TrimSpace(part)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func fmMap(data map[string]any, key string) map[string]any {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	normalized, ok := normalizeFrontmatterValue(v).(map[string]any)
	if !ok {
		return nil
	}
	return normalized
}

func normalizeFrontmatterValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = normalizeFrontmatterValue(item)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			key := strings.TrimSpace(fmt.Sprint(k))
			if key == "" {
				continue
			}
			out[key] = normalizeFrontmatterValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, normalizeFrontmatterValue(item))
		}
		return out
	default:
		return x
	}
}
