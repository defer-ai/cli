package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ToolCall represents a parsed tool invocation from Claude.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult is the outcome of executing a tool.
type ToolResult struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// HumanDescription generates a readable summary of a tool call.
func (tc *ToolCall) HumanDescription() string {
	switch tc.Name {
	case "Write":
		var in struct{ FilePath string `json:"file_path"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Creating %s", filepath.Base(in.FilePath))
	case "Edit":
		var in struct{ FilePath string `json:"file_path"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Editing %s", filepath.Base(in.FilePath))
	case "Bash":
		var in struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		}
		json.Unmarshal(tc.Input, &in)
		if in.Description != "" {
			desc := in.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			return desc
		}
		cmd := in.Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		return fmt.Sprintf("Running: %s", cmd)
	case "Read":
		var in struct{ FilePath string `json:"file_path"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Reading %s", filepath.Base(in.FilePath))
	case "Glob":
		var in struct{ Pattern string `json:"pattern"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Finding files matching %s", in.Pattern)
	case "Grep":
		var in struct{ Pattern string `json:"pattern"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Searching for \"%s\"", in.Pattern)
	case "WebSearch":
		var in struct{ Query string `json:"query"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Searching the web: %s", in.Query)
	case "WebFetch":
		var in struct{ URL string `json:"url"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Fetching %s", in.URL)
	case "Agent":
		var in struct{ Description string `json:"description"` }
		json.Unmarshal(tc.Input, &in)
		if in.Description != "" {
			return in.Description
		}
		return "Spawning sub-agent..."
	case "ToolSearch":
		var in struct{ Query string `json:"query"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Looking up tools: %s", in.Query)
	case "AskUserQuestion":
		return "Waiting for input..."
	case "EnterPlanMode":
		return "Planning approach..."
	case "ExitPlanMode":
		return "Plan complete, executing..."
	case "TaskCreate":
		var in struct{ Subject string `json:"subject"` }
		json.Unmarshal(tc.Input, &in)
		return fmt.Sprintf("Creating task: %s", in.Subject)
	case "TaskUpdate":
		return "Updating task..."
	case "NotebookEdit":
		return "Editing notebook..."
	case "LSP":
		return "Querying language server..."
	// --- Defer MCP gated-write tools ---
	case "mcp__defer__register_decision":
		var in struct {
			Category string `json:"category"`
			Question string `json:"question"`
			Chosen   string `json:"chosen"`
		}
		json.Unmarshal(tc.Input, &in)
		q := in.Question
		if len(q) > 60 {
			q = q[:57] + "..."
		}
		if q != "" && in.Chosen != "" {
			return fmt.Sprintf("%s: %s → %s", in.Category, q, in.Chosen)
		}
		if q != "" {
			return fmt.Sprintf("Deciding: %s", q)
		}
		return "Registering decision"
	case "mcp__defer__write_file":
		var in struct {
			Path        string   `json:"path"`
			DecisionIDs []string `json:"decision_ids"`
		}
		json.Unmarshal(tc.Input, &in)
		if in.Path != "" {
			return fmt.Sprintf("Writing %s (%d decisions)", filepath.Base(in.Path), len(in.DecisionIDs))
		}
		return "Writing file"
	case "mcp__defer__read_decisions":
		return "Reading decisions"
	case "mcp__defer__list_pending":
		return "Checking pending decisions"
	case "mcp__defer__confirm_decision":
		return "Confirming decision"
	case "mcp__defer__update_decision":
		return "Updating decision"
	case "mcp__defer__get_session_state":
		return "Checking session state"
	case "mcp__defer__get_decision_tree":
		return "Viewing decision tree"
	default:
		// Strip mcp__ prefix for any unrecognized MCP tools so the UI
		// doesn't show raw protocol-internal names.
		if strings.HasPrefix(tc.Name, "mcp__") {
			parts := strings.SplitN(tc.Name, "__", 3)
			if len(parts) == 3 {
				return strings.ReplaceAll(parts[2], "_", " ")
			}
		}
		return tc.Name
	}
}

// IsMajorAction returns true for Write, Edit, Bash (actions that modify state).
func (tc *ToolCall) IsMajorAction() bool {
	return tc.Name == "Write" || tc.Name == "Edit" || tc.Name == "Bash"
}

// ExecuteTool runs a tool call and returns the result.
func ExecuteTool(ctx context.Context, call ToolCall, cwd string) ToolResult {
	switch call.Name {
	case "Read":
		return execRead(call, cwd)
	case "Write":
		return execWrite(call, cwd)
	case "Edit":
		return execEdit(call, cwd)
	case "Bash":
		return execBash(ctx, call, cwd)
	case "Glob":
		return execGlob(call, cwd)
	case "Grep":
		return execGrep(ctx, call, cwd)
	default:
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Unknown tool: %s", call.Name), IsError: true}
	}
}

func resolvePath(path, cwd string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cwd, path)
}

func execRead(call ToolCall, cwd string) ToolResult {
	var in struct {
		FilePath string `json:"file_path"`
		Offset   *int   `json:"offset"`
		Limit    *int   `json:"limit"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	p := resolvePath(in.FilePath, cwd)
	f, err := os.Open(p)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	lineNum := 0
	offset := 0
	limit := 2000
	if in.Offset != nil {
		offset = *in.Offset
	}
	if in.Limit != nil {
		limit = *in.Limit
	}

	for scanner.Scan() {
		if lineNum >= offset && lineNum < offset+limit {
			lines = append(lines, fmt.Sprintf("%6d\t%s", lineNum+1, scanner.Text()))
		}
		lineNum++
		if lineNum >= offset+limit {
			break
		}
	}

	return ToolResult{ToolUseID: call.ID, Content: strings.Join(lines, "\n")}
}

func execWrite(call ToolCall, cwd string) ToolResult {
	var in struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	p := resolvePath(in.FilePath, cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}
	if err := os.WriteFile(p, []byte(in.Content), 0o644); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Wrote %d bytes to %s", len(in.Content), in.FilePath)}
}

func execEdit(call ToolCall, cwd string) ToolResult {
	var in struct {
		FilePath string `json:"file_path"`
		OldText  string `json:"old_text"`
		NewText  string `json:"new_text"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	p := resolvePath(in.FilePath, cwd)
	data, err := os.ReadFile(p)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	content := string(data)
	if !strings.Contains(content, in.OldText) {
		return ToolResult{ToolUseID: call.ID, Content: "old_text not found in file", IsError: true}
	}

	newContent := strings.Replace(content, in.OldText, in.NewText, 1)
	if err := os.WriteFile(p, []byte(newContent), 0o644); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Edited %s", in.FilePath)}
}

func execBash(ctx context.Context, call ToolCall, cwd string) ToolResult {
	var in struct {
		Command string `json:"command"`
		Timeout *int   `json:"timeout"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	timeout := 120 * time.Second
	if in.Timeout != nil && *in.Timeout > 0 {
		timeout = time.Duration(*in.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", in.Command)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()

	result := string(out)
	if len(result) > 50000 {
		result = result[:50000] + "\n...(truncated)"
	}

	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: result + "\nError: " + err.Error(), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: result}
}

func execGlob(call ToolCall, cwd string) ToolResult {
	var in struct {
		Pattern string  `json:"pattern"`
		Path    *string `json:"path"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	dir := cwd
	if in.Path != nil {
		dir = resolvePath(*in.Path, cwd)
	}

	pattern := filepath.Join(dir, in.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	// Make paths relative to cwd
	var relative []string
	for _, m := range matches {
		rel, err := filepath.Rel(cwd, m)
		if err != nil {
			relative = append(relative, m)
		} else {
			relative = append(relative, rel)
		}
	}

	if len(relative) == 0 {
		return ToolResult{ToolUseID: call.ID, Content: "No matches found"}
	}
	return ToolResult{ToolUseID: call.ID, Content: strings.Join(relative, "\n")}
}

func execGrep(ctx context.Context, call ToolCall, cwd string) ToolResult {
	var in struct {
		Pattern string  `json:"pattern"`
		Path    *string `json:"path"`
		Glob    *string `json:"glob"`
	}
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: err.Error(), IsError: true}
	}

	args := []string{"-rn", "--color=never"}
	if in.Glob != nil {
		args = append(args, "--include="+*in.Glob)
	}
	args = append(args, in.Pattern)

	dir := cwd
	if in.Path != nil {
		dir = resolvePath(*in.Path, cwd)
	}
	args = append(args, dir)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "grep", args...)
	out, _ := cmd.CombinedOutput() // grep returns exit 1 for no matches

	result := string(out)
	if len(result) > 50000 {
		result = result[:50000] + "\n...(truncated)"
	}
	if result == "" {
		return ToolResult{ToolUseID: call.ID, Content: "No matches found"}
	}
	return ToolResult{ToolUseID: call.ID, Content: result}
}
