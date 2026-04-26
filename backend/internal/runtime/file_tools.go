package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"novel-agent-runtime/internal/model"
	"novel-agent-runtime/internal/runstore"
	"novel-agent-runtime/internal/skill"
)

const (
	skillReadToolName  = "Read"
	skillWriteToolName = "Write"
	skillEditToolName  = "Edit"
	skillGlobToolName  = "Glob"
)

type skillFileReadState struct {
	Content   string
	Timestamp int64
	Full      bool
}

type skillFileToolSession struct {
	WorkspaceRoot     string
	DocumentOutputDir string
	ReadState         map[string]skillFileReadState
	Store             *runstore.Store
	StorePrefix       string
}

func newSkillFileToolSession(cfg RuntimeConfigView, store *runstore.Store, prefix string) *skillFileToolSession {
	return &skillFileToolSession{
		WorkspaceRoot:     filepath.Clean(cfg.WorkspaceRoot),
		DocumentOutputDir: filepath.Clean(cfg.DocumentOutputDir),
		ReadState:         map[string]skillFileReadState{},
		Store:             store,
		StorePrefix:       prefix,
	}
}

type RuntimeConfigView struct {
	WorkspaceRoot     string
	DocumentOutputDir string
}

func (s *skillFileToolSession) toolSpecs(allowed []string) []model.ToolSpec {
	enabled := canonicalSkillLocalTools(allowed)
	var specs []model.ToolSpec
	for _, name := range enabled {
		switch name {
		case skillReadToolName:
			specs = append(specs, readToolSpec())
		case skillWriteToolName:
			specs = append(specs, writeToolSpec())
		case skillEditToolName:
			specs = append(specs, editToolSpec())
		case skillGlobToolName:
			specs = append(specs, globToolSpec())
		case skillBashToolName:
			specs = append(specs, bashToolSpec())
		case skillPowerShellToolName:
			specs = append(specs, powerShellToolSpec())
		}
	}
	return specs
}

func canonicalSkillLocalTools(allowed []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range allowed {
		name, ok := canonicalSkillLocalToolName(item)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func canonicalSkillLocalToolName(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case strings.ToLower(skillReadToolName), "read_file", "file_read":
		return skillReadToolName, true
	case strings.ToLower(skillWriteToolName), "write_file", "file_write":
		return skillWriteToolName, true
	case strings.ToLower(skillEditToolName), "edit_file", "file_edit":
		return skillEditToolName, true
	case strings.ToLower(skillGlobToolName), "glob_files", "file_glob":
		return skillGlobToolName, true
	case strings.ToLower(skillBashToolName), "bash_tool", "shell":
		return skillBashToolName, true
	case strings.ToLower(skillPowerShellToolName), "powershell_tool", "pwsh":
		return skillPowerShellToolName, true
	default:
		return "", false
	}
}

func readToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillReadToolName,
			Description: "Read a text file from the workspace. Supports optional line offset and line limit for large files.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string", "description": "Absolute path or workspace-relative path to the file."},
					"offset":    map[string]any{"type": "integer", "description": "1-based line number to start reading from."},
					"limit":     map[string]any{"type": "integer", "description": "Number of lines to read."},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

func writeToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillWriteToolName,
			Description: "Create or fully overwrite a text file in the workspace. If the target file already exists, you must read it first in the current skill execution.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string", "description": "Absolute path or workspace-relative path to the file."},
					"content":   map[string]any{"type": "string", "description": "Full file content to write."},
				},
				"required": []string{"file_path", "content"},
			},
		},
	}
}

func editToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillEditToolName,
			Description: "Edit an existing text file by replacing a specific string. You must read the file first in the current skill execution.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path":   map[string]any{"type": "string", "description": "Absolute path or workspace-relative path to the file."},
					"old_string":  map[string]any{"type": "string", "description": "Exact text to replace. Use empty string only when creating a new file through Edit."},
					"new_string":  map[string]any{"type": "string", "description": "Replacement text."},
					"replace_all": map[string]any{"type": "boolean", "description": "Whether to replace every occurrence of old_string."},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			},
		},
	}
}

func globToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillGlobToolName,
			Description: "Find files by glob pattern. Returns workspace-relative paths.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{"type": "string", "description": "Glob pattern such as **/*.md or docs/**/*.md."},
					"path":    map[string]any{"type": "string", "description": "Optional absolute or workspace-relative directory to search in."},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (s *skillFileToolSession) handleToolCall(tc model.ToolCall) (string, error) {
	_ = s.Store.WriteJSON(filepath.ToSlash(filepath.Join(s.StorePrefix, "tools", fmt.Sprintf("%s-%s-call.json", tc.Function.Name, tc.ID))), tc)
	var (
		payload map[string]any
		err     error
	)
	switch tc.Function.Name {
	case skillReadToolName:
		payload, err = s.handleRead(tc.Function.Arguments)
	case skillWriteToolName:
		payload, err = s.handleWrite(tc.Function.Arguments)
	case skillEditToolName:
		payload, err = s.handleEdit(tc.Function.Arguments)
	case skillGlobToolName:
		payload, err = s.handleGlob(tc.Function.Arguments)
	case skillBashToolName:
		payload, err = s.handleBash(tc.Function.Arguments)
	case skillPowerShellToolName:
		payload, err = s.handlePowerShell(tc.Function.Arguments)
	default:
		err = fmt.Errorf("unknown skill file tool: %s", tc.Function.Name)
	}
	if err != nil {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["error"] = err.Error()
	}
	_ = s.Store.WriteJSON(filepath.ToSlash(filepath.Join(s.StorePrefix, "tools", fmt.Sprintf("%s-%s-result.json", tc.Function.Name, tc.ID))), payload)
	if err != nil {
		return model.MustJSON(payload), err
	}
	return model.MustJSON(payload), nil
}

func (s *skillFileToolSession) handleRead(raw string) (map[string]any, error) {
	var args struct {
		FilePath string `json:"file_path"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	abs, rel, err := s.resolvePath(args.FilePath, false)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory: %s", rel)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	content := normalizeTextForFileTools(string(data))
	lines := splitFileLines(content)
	totalLines := len(lines)
	startLine := args.Offset
	if startLine <= 0 {
		startLine = 1
	}
	if startLine > totalLines && totalLines > 0 {
		startLine = totalLines
	}
	startIdx := 0
	if startLine > 0 {
		startIdx = startLine - 1
	}
	endIdx := totalLines
	if args.Limit > 0 && startIdx+args.Limit < endIdx {
		endIdx = startIdx + args.Limit
	}
	selected := ""
	if totalLines > 0 && startIdx < totalLines {
		selected = strings.Join(lines[startIdx:endIdx], "\n")
	}
	fullRead := args.Offset == 0 && args.Limit == 0
	if !fullRead && startLine == 1 && (args.Limit == 0 || endIdx >= totalLines) {
		fullRead = true
	}
	s.ReadState[abs] = skillFileReadState{
		Content:   content,
		Timestamp: info.ModTime().UnixMilli(),
		Full:      fullRead,
	}
	return map[string]any{
		"type":          "text",
		"file_path":     rel,
		"absolute_path": abs,
		"content":       selected,
		"start_line":    startLine,
		"num_lines":     maxInt(0, endIdx-startIdx),
		"total_lines":   totalLines,
		"full_read":     fullRead,
	}, nil
}

func (s *skillFileToolSession) handleWrite(raw string) (map[string]any, error) {
	var args struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	abs, rel, err := s.resolvePath(args.FilePath, true)
	if err != nil {
		return nil, err
	}
	oldContent := ""
	fileType := "create"
	if info, err := os.Stat(abs); err == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("cannot write directory: %s", rel)
		}
		state, ok := s.ReadState[abs]
		if !ok || !state.Full {
			return nil, fmt.Errorf("file has not been fully read yet. Use %s first before overwriting %s", skillReadToolName, rel)
		}
		current, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		oldContent = normalizeTextForFileTools(string(current))
		if currentMtime := info.ModTime().UnixMilli(); currentMtime > state.Timestamp && oldContent != state.Content {
			return nil, fmt.Errorf("file has changed since it was read. Use %s again before writing %s", skillReadToolName, rel)
		}
		fileType = "update"
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, []byte(args.Content), 0o644); err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	s.ReadState[abs] = skillFileReadState{
		Content:   normalizeTextForFileTools(args.Content),
		Timestamp: info.ModTime().UnixMilli(),
		Full:      true,
	}
	return map[string]any{
		"type":          fileType,
		"file_path":     rel,
		"absolute_path": abs,
		"bytes":         len(args.Content),
		"old_content":   oldContent,
	}, nil
}

func (s *skillFileToolSession) handleEdit(raw string) (map[string]any, error) {
	var args struct {
		FilePath   string `json:"file_path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	abs, rel, err := s.resolvePath(args.FilePath, true)
	if err != nil {
		return nil, err
	}
	current := ""
	fileExists := false
	if info, err := os.Stat(abs); err == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("cannot edit directory: %s", rel)
		}
		state, ok := s.ReadState[abs]
		if !ok || !state.Full {
			return nil, fmt.Errorf("file has not been fully read yet. Use %s first before editing %s", skillReadToolName, rel)
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		current = normalizeTextForFileTools(string(body))
		if info.ModTime().UnixMilli() > state.Timestamp && current != state.Content {
			return nil, fmt.Errorf("file has changed since it was read. Use %s again before editing %s", skillReadToolName, rel)
		}
		fileExists = true
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if !fileExists {
		if args.OldString != "" {
			return nil, fmt.Errorf("file does not exist: %s", rel)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(abs, []byte(args.NewString), 0o644); err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		s.ReadState[abs] = skillFileReadState{
			Content:   normalizeTextForFileTools(args.NewString),
			Timestamp: info.ModTime().UnixMilli(),
			Full:      true,
		}
		return map[string]any{
			"type":          "create",
			"file_path":     rel,
			"absolute_path": abs,
			"match_count":   0,
		}, nil
	}

	if args.OldString == args.NewString {
		return nil, fmt.Errorf("old_string and new_string are identical for %s", rel)
	}
	matchCount := strings.Count(current, args.OldString)
	if matchCount == 0 {
		return nil, fmt.Errorf("old_string was not found in %s", rel)
	}
	if matchCount > 1 && !args.ReplaceAll {
		return nil, fmt.Errorf("old_string matched %d times in %s. Set replace_all=true or provide a more specific old_string", matchCount, rel)
	}
	updated := current
	if args.ReplaceAll {
		updated = strings.ReplaceAll(current, args.OldString, args.NewString)
	} else {
		updated = strings.Replace(current, args.OldString, args.NewString, 1)
	}
	if err := os.WriteFile(abs, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	s.ReadState[abs] = skillFileReadState{
		Content:   updated,
		Timestamp: info.ModTime().UnixMilli(),
		Full:      true,
	}
	return map[string]any{
		"type":          "update",
		"file_path":     rel,
		"absolute_path": abs,
		"match_count":   matchCount,
		"replace_all":   args.ReplaceAll,
	}, nil
}

func (s *skillFileToolSession) handleGlob(raw string) (map[string]any, error) {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Pattern) == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	baseAbs := s.WorkspaceRoot
	baseRel := "."
	var err error
	if strings.TrimSpace(args.Path) != "" {
		baseAbs, baseRel, err = s.resolvePath(args.Path, false)
		if err != nil {
			return nil, err
		}
	}
	info, err := os.Stat(baseAbs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("glob path is not a directory: %s", baseRel)
	}
	fsys := os.DirFS(baseAbs)
	matches, err := doublestar.Glob(fsys, filepath.ToSlash(args.Pattern))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) > 100 {
		matches = matches[:100]
	}
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		abs := filepath.Join(baseAbs, filepath.FromSlash(match))
		rel, err := filepath.Rel(s.WorkspaceRoot, abs)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			continue
		}
		stat, err := fs.Stat(fsys, match)
		if err == nil && stat.IsDir() {
			continue
		}
		files = append(files, rel)
	}
	return map[string]any{
		"pattern":   args.Pattern,
		"base_path": filepath.ToSlash(baseRel),
		"num_files": len(files),
		"files":     files,
	}, nil
}

func (s *skillFileToolSession) resolvePath(input string, allowCreate bool) (string, string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", "", fmt.Errorf("file path is required")
	}
	cleaned := raw
	if !filepath.IsAbs(cleaned) {
		cleaned = filepath.Join(s.WorkspaceRoot, cleaned)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", "", err
	}
	abs = filepath.Clean(abs)
	if !pathWithinRoot(s.WorkspaceRoot, abs) {
		return "", "", fmt.Errorf("path is outside workspace root: %s", raw)
	}
	if !allowCreate {
		if _, err := os.Stat(abs); err != nil {
			return "", "", err
		}
	}
	rel, err := filepath.Rel(s.WorkspaceRoot, abs)
	if err != nil {
		rel = abs
	}
	return abs, filepath.ToSlash(rel), nil
}

func pathWithinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if strings.EqualFold(root, target) {
		return true
	}
	rootWithSep := root
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}
	return strings.HasPrefix(strings.ToLower(target), strings.ToLower(rootWithSep))
}

func normalizeTextForFileTools(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func splitFileLines(text string) []string {
	text = normalizeTextForFileTools(text)
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func skillDocumentHint(cmd skill.Command, session *skillFileToolSession) string {
	specs := session.toolSpecs(cmd.AllowedTools)
	if len(specs) == 0 {
		return ""
	}
	root := filepath.ToSlash(session.WorkspaceRoot)
	defaultDir := filepath.ToSlash(session.DocumentOutputDir)
	var names []string
	for _, spec := range specs {
		names = append(names, spec.Function.Name)
	}
	guidance := "When the task produces a durable novel artifact, prefer writing a markdown document under preferred_document_output_dir. If document_path is provided in the tool arguments, use that path. After writing, return a short summary with file path instead of repeating the whole document unless the user explicitly asked for chat-only output."
	available := strings.Join(names, ", ")
	if strings.Contains(available, skillBashToolName) || strings.Contains(available, skillPowerShellToolName) {
		guidance += " Use Bash or PowerShell only for terminal operations such as git, build, test, environment inspection, or process commands. Do not use shell tools for file search, file reads, file edits, or file writes when Glob, Read, Edit, or Write are available."
	}
	return fmt.Sprintf(
		"workspace_root: %s\npreferred_document_output_dir: %s\nallowed_local_tools: %s\n%s",
		root,
		defaultDir,
		available,
		guidance,
	)
}
