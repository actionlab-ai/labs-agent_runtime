package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"novel-agent-runtime/internal/model"
)

const (
	skillBashToolName            = "Bash"
	skillPowerShellToolName      = "PowerShell"
	defaultSkillShellTimeoutMs   = 30_000
	maxSkillShellTimeoutMs       = 120_000
	maxSkillShellOutputChars     = 20_000
	defaultPowerShellCommand     = "powershell"
	defaultPowerShellCoreCommand = "pwsh"
	novelBashPathEnv             = "NOVEL_BASH_PATH"
	bashToolGuidanceLine         = "Use Bash for terminal operations such as git, build, test, packaging, and process inspection. Do NOT use Bash for file search, file reads, file edits, or file writes when Glob, Read, Edit, or Write are available."
	powerShellToolGuidanceLine   = "Use PowerShell for terminal operations such as git, build, test, environment inspection, and process commands. Do NOT use PowerShell for file search, file reads, file edits, or file writes when Glob, Read, Edit, or Write are available."
	fileToolPriorityGuidanceLine = "File search: use Glob instead of find/ls/Get-ChildItem -Recurse. Read files: use Read instead of cat/head/tail/Get-Content. Edit files: use Edit instead of sed/awk/manual replace loops. Write files: use Write instead of echo >/cat <<EOF/Set-Content/Out-File."
)

func bashToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillBashToolName,
			Description: fmt.Sprintf("Execute a Bash command in the workspace root. %s %s Use this for terminal work, not for direct file manipulation.", bashToolGuidanceLine, fileToolPriorityGuidanceLine),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "The Bash command to execute."},
					"timeout_ms": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("Optional timeout in milliseconds. Defaults to %d and is capped at %d.", defaultSkillShellTimeoutMs, maxSkillShellTimeoutMs),
					},
					"description": map[string]any{"type": "string", "description": "Optional short explanation of why this shell command is needed."},
				},
				"required": []string{"command"},
			},
		},
	}
}

func powerShellToolSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        skillPowerShellToolName,
			Description: fmt.Sprintf("Execute a PowerShell command in the workspace root. %s %s Use this for terminal work, not for direct file manipulation.", powerShellToolGuidanceLine, fileToolPriorityGuidanceLine),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "The PowerShell command to execute."},
					"timeout_ms": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("Optional timeout in milliseconds. Defaults to %d and is capped at %d.", defaultSkillShellTimeoutMs, maxSkillShellTimeoutMs),
					},
					"description": map[string]any{"type": "string", "description": "Optional short explanation of why this shell command is needed."},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (s *skillFileToolSession) handleBash(raw string) (map[string]any, error) {
	executable, prefixArgs, err := resolveBashCommand()
	if err != nil {
		return nil, err
	}
	return s.handleShell(raw, skillBashToolName, executable, prefixArgs)
}

func (s *skillFileToolSession) handlePowerShell(raw string) (map[string]any, error) {
	executable, prefixArgs, err := resolvePowerShellCommand()
	if err != nil {
		return nil, err
	}
	return s.handleShell(raw, skillPowerShellToolName, executable, prefixArgs)
}

func (s *skillFileToolSession) handleShell(raw, shellName, executable string, prefixArgs []string) (map[string]any, error) {
	var args struct {
		Command     string `json:"command"`
		TimeoutMS   int    `json:"timeout_ms"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	commandText := strings.TrimSpace(args.Command)
	if commandText == "" {
		return nil, fmt.Errorf("%s command is required", shellName)
	}
	timeoutMs := resolvedShellTimeoutMs(args.TimeoutMS)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmdArgs := append([]string{}, prefixArgs...)
	cmdArgs = append(cmdArgs, commandText)
	cmd := exec.CommandContext(ctx, executable, cmdArgs...)
	cmd.Dir = s.WorkspaceRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stdoutText, stdoutTruncated := truncateShellOutput(stdout.String())
	stderrText, stderrTruncated := truncateShellOutput(stderr.String())
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			exitCode = -1
		} else {
			exitCode = -1
		}
	}
	if cmd.ProcessState != nil && err == nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	payload := map[string]any{
		"shell":             shellName,
		"command":           commandText,
		"description":       strings.TrimSpace(args.Description),
		"working_directory": filepath.ToSlash(s.WorkspaceRoot),
		"timeout_ms":        timeoutMs,
		"exit_code":         exitCode,
		"stdout":            stdoutText,
		"stderr":            stderrText,
		"stdout_truncated":  stdoutTruncated,
		"stderr_truncated":  stderrTruncated,
		"success":           err == nil,
	}
	if ctx.Err() == context.DeadlineExceeded {
		return payload, fmt.Errorf("%s command timed out after %dms", shellName, timeoutMs)
	}
	if err != nil {
		return payload, fmt.Errorf("%s command failed with exit code %d", shellName, exitCode)
	}
	return payload, nil
}

func resolveBashCommand() (string, []string, error) {
	if custom := strings.TrimSpace(os.Getenv(novelBashPathEnv)); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, []string{"-lc"}, nil
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		return path, []string{"-lc"}, nil
	}
	for _, candidate := range []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files\Git\usr\bin\bash.exe`,
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, []string{"-lc"}, nil
		}
	}
	return "", nil, fmt.Errorf("bash executable not found. Install bash or set %s to the full path", novelBashPathEnv)
}

func resolvePowerShellCommand() (string, []string, error) {
	if path, err := exec.LookPath(defaultPowerShellCoreCommand); err == nil {
		return path, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command"}, nil
	}
	if path, err := exec.LookPath(defaultPowerShellCommand); err == nil {
		return path, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command"}, nil
	}
	return "", nil, fmt.Errorf("powershell executable not found")
}

func resolvedShellTimeoutMs(requested int) int {
	if requested <= 0 {
		return defaultSkillShellTimeoutMs
	}
	if requested > maxSkillShellTimeoutMs {
		return maxSkillShellTimeoutMs
	}
	return requested
}

func truncateShellOutput(text string) (string, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if len(text) <= maxSkillShellOutputChars {
		return text, false
	}
	return text[:maxSkillShellOutputChars] + "\n...[truncated]", true
}
