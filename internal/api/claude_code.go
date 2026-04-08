package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/defer-ai/cli/internal/decision"
)

// EventType classifies agent loop events.
type EventType int

const (
	EventTextDelta         EventType = iota // Claude produced text
	EventToolCallStart                      // Claude wants to call a tool
	EventToolCallDone                       // Tool execution finished
	EventDecisionFound                      // An implicit decision was logged
	EventPermissionRequest                  // Subprocess needs permission to use a tool
	EventDone                               // Agent loop finished
	EventError                              // Something went wrong
)

// PermissionRequest represents a tool permission request from the Claude subprocess.
type PermissionRequest struct {
	RequestID  string
	ToolName   string
	ToolUseID  string
	Input      json.RawMessage
	ResponseCh chan PermissionResponse // caller writes here to approve/deny
}

// PermissionResponse is the user's decision on a permission request.
type PermissionResponse struct {
	Allow   bool
	Message string // reason for denial (used when Allow is false)
}

// Event is emitted by the agent loop.
type Event struct {
	Type          EventType
	Text          string             // for TextDelta
	ToolCall      *ToolCall          // for ToolCallStart
	ToolResult    *ToolResult        // for ToolCallDone
	Decision      *decision.Decision // for DecisionFound
	PermissionReq *PermissionRequest // for PermissionRequest
	Error         error              // for Error
}

// ClaudeCodeProvider runs Claude Code as a subprocess.
// Used when no ANTHROPIC_API_KEY is set (user has Claude Code authenticated via subscription).
type ClaudeCodeProvider struct {
	model        string
	cwd          string   // working directory for the subprocess
	sessionID    string   // Claude session ID (persisted in .defer/)
	AllowedTools []string // if set, restricts which tools the subprocess can use
	Effort       string   // "low", "medium", "high", "max" — passed via --effort
	// StrictMode enables the executor-hardening bundle: Bash removed from
	// the toolkit, a PreToolUse hook reminder on every Write/Edit, and a
	// system-prompt appendix explaining the restriction. Enabled by default
	// for the executor phase (via executor.freshProvider) after a 5×2 bench
	// on a Flask task showed it improves inline narration from 0% to 14%
	// tool-anchored with a 20% mean inline increase and a 14% speed-up.
	// Env var overrides (DEFER_CLAUDE_SETTINGS, DEFER_CLAUDE_DISALLOWED_TOOLS,
	// DEFER_CLAUDE_APPEND_SYSTEM_PROMPT) take precedence over strict's
	// defaults when set.
	StrictMode bool
}

// strictAppendSystemPrompt is the per-invocation appendix the executor adds
// when StrictMode is on. It explains which tools are gone and reinforces
// the DECIDED-between-writes protocol. Without this, the model sometimes
// flails looking for a tool that was removed, or tries to invoke plugin
// skills (brainstorming, TDD, etc.) that were auto-injected by SessionStart
// hooks from the user's ~/.claude/plugins.
const strictAppendSystemPrompt = "IMPORTANT: You are a defer executor running in strict, headless mode. The following tools are NOT available: Bash, Skill, Task, AskUserQuestion, EnterPlanMode. Do not attempt to invoke plugin skills (brainstorming, TDD, debugging, etc.) — implement the decisions directly. Do not spawn sub-agents. Do not ask the user questions. Use only Write, Edit, Read, Glob, and Grep. After any Write or Edit you must emit DECIDED lines before calling another tool."

// strictDisallowedTools is the comma-separated list of tool names removed
// from the Claude Code executor when StrictMode is on. Each one is a known
// source of executor-phase contamination:
//
//   Bash             — file-write bypass via cat>EOF heredoc, sed -i, etc.
//                      Also lets the model run arbitrary shell commands
//                      that aren't tracked as decisions.
//   Skill            — loads plugin skills from ~/.claude/plugins (e.g.
//                      superpowers' brainstorming skill). These are
//                      designed for interactive use and interrupt defer
//                      with visual demos and questions.
//   Task             — spawns a nested Claude Code sub-agent, adding a
//                      second layer of autonomy defer doesn't track.
//   AskUserQuestion  — interactive prompts are antithetical to defer's
//                      zero-autonomy contract.
//   EnterPlanMode    — plan mode is interactive; the executor has already
//                      been given confirmed decisions to implement.
const strictDisallowedTools = "Bash,Skill,Task,AskUserQuestion,EnterPlanMode"

// strictHookSettingsJSON is the Claude Code settings payload that installs
// a PreToolUse hook on Write/Edit/MultiEdit/NotebookEdit. The hook's command
// prints a JSON hookSpecificOutput block with an additionalContext reminder
// — the only format Claude Code feeds back into the conversation (plain
// stdout is silently discarded).
const strictHookSettingsJSON = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "printf '%s' '{\"hookSpecificOutput\": {\"hookEventName\": \"PreToolUse\", \"additionalContext\": \"[STOP: Before this file modification executes, emit one or more DECIDED lines covering the choices you are about to materialize. Format: DECIDED: category | question | chosen | alternatives | reason. The file write will proceed after you emit the narration.]\"}}'"
          }
        ]
      }
    ]
  }
}
`

// ensureStrictHookFile writes the strict hook settings to ~/.defer/strict-hook.json
// if it's not already present and returns the absolute path. The file is
// shared across defer invocations — we write it once per host install.
func ensureStrictHookFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".defer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "strict-hook.json")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(strictHookSettingsJSON), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// NewClaudeCodeProvider creates a subprocess provider using the current working directory.
func NewClaudeCodeProvider(model string) *ClaudeCodeProvider {
	cwd, _ := os.Getwd()
	return &ClaudeCodeProvider{model: model, cwd: cwd}
}

// NewClaudeCodeProviderWithCWD creates a subprocess provider with an explicit working directory.
func NewClaudeCodeProviderWithCWD(model, cwd string) *ClaudeCodeProvider {
	return &ClaudeCodeProvider{model: model, cwd: cwd}
}

// SetEffort sets the --effort level for the Claude Code subprocess.
// Valid values: "low", "medium", "high", "max". Empty string omits the flag.
func (p *ClaudeCodeProvider) SetEffort(effort string) {
	p.Effort = effort
}

// IsClaudeInstalled checks if the claude binary is available.
func IsClaudeInstalled() bool {
	return findClaude() != ""
}

func findClaude() string {
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".npm-global", "bin", "claude"),
		"/usr/local/bin/claude",
		"/usr/bin/claude",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	path, err := exec.LookPath("claude")
	if err == nil {
		return path
	}
	return ""
}

// RunCompletion sends a prompt via Claude Code subprocess and emits events.
// Events are sent to the channel as they occur. The channel is closed when done.
func (p *ClaudeCodeProvider) RunCompletion(ctx context.Context, systemPrompt, userPrompt string, events chan<- Event) {
	defer func() {
		if r := recover(); r != nil {
			events <- Event{Type: EventError, Error: fmt.Errorf("provider panic: %v", r)}
		}
	}()

	claudePath := findClaude()
	if claudePath == "" {
		events <- Event{Type: EventError, Error: fmt.Errorf("claude binary not found")}
		return
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--model", p.model,
		"--dangerously-skip-permissions",
	}
	if p.Effort != "" {
		args = append(args, "--effort", p.Effort)
	}
	// Settings file: env var override first, then strict-mode default.
	// DEFER_CLAUDE_SETTINGS forces a specific --settings file (used by bench
	// experiments). When unset and StrictMode is on, we write/reuse the
	// canonical strict-hook.json under ~/.defer and pass that.
	settingsPath := os.Getenv("DEFER_CLAUDE_SETTINGS")
	if settingsPath == "" && p.StrictMode {
		if hookPath, err := ensureStrictHookFile(); err == nil {
			settingsPath = hookPath
		}
	}
	if settingsPath != "" {
		args = append(args, "--settings", settingsPath)
	}
	// Allowed tools: env var first (no-op under --dangerously-skip-permissions
	// but kept for completeness), then provider.AllowedTools. This is the
	// permissive list for decompose/chat phases.
	if envTools := os.Getenv("DEFER_CLAUDE_ALLOWED_TOOLS"); envTools != "" && len(p.AllowedTools) == 0 {
		args = append(args, "--allowedTools", envTools)
	}
	// Disallowed tools: env var override, then strict-mode default.
	// --disallowedTools actually removes the tool from the session even
	// under --dangerously-skip-permissions. See strictDisallowedTools for
	// the rationale on each entry.
	disallowed := os.Getenv("DEFER_CLAUDE_DISALLOWED_TOOLS")
	if disallowed == "" && p.StrictMode {
		disallowed = strictDisallowedTools
	}
	if disallowed != "" {
		args = append(args, "--disallowedTools", disallowed)
	}
	// Append-system-prompt: env var override, then strict-mode default.
	appendPrompt := os.Getenv("DEFER_CLAUDE_APPEND_SYSTEM_PROMPT")
	if appendPrompt == "" && p.StrictMode {
		appendPrompt = strictAppendSystemPrompt
	}
	if appendPrompt != "" {
		args = append(args, "--append-system-prompt", appendPrompt)
	}
	// AllowedTools restricts which tools the subprocess can use per-phase:
	// - Decomposition: read-only (no Write/Edit)
	// - Execution: full access (user confirmed decisions)
	// - Chat: read-only
	if len(p.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(p.AllowedTools, ","))
	}

	args = append(args, "--system-prompt", systemPrompt)

	args = append(args, userPrompt)

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = os.Environ()
	if p.cwd != "" {
		cmd.Dir = p.cwd
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		events <- Event{Type: EventError, Error: err}
		return
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		events <- Event{Type: EventError, Error: err}
		return
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		events <- Event{Type: EventError, Error: err}
		return
	}

	// Timeout: kill if no output for 5 minutes
	lastActivity := time.Now()
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if time.Since(lastActivity) > 5*time.Minute {
					cmd.Process.Kill()
					return
				}
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	textEmitted := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lastActivity = time.Now()

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		typ, _ := event["type"].(string)

		switch typ {
		case "system":
			if sid, ok := event["session_id"].(string); ok {
				p.sessionID = sid
			}

		case "control_request":
			p.handleControlRequest(event, stdinPipe, events)
			continue

		// Log unhandled event types for debugging
		case "":
			continue

		case "content_block_delta":
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if deltaType, _ := delta["type"].(string); deltaType == "text_delta" {
					if text, ok := delta["text"].(string); ok && text != "" {
						events <- Event{Type: EventTextDelta, Text: text}
						textEmitted = true
					}
				}
			}

		case "assistant":
			if msg, ok := event["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].([]interface{}); ok {
					for _, block := range content {
						b, ok := block.(map[string]interface{})
						if !ok {
							continue
						}
						blockType, _ := b["type"].(string)

						if blockType == "text" && !textEmitted {
							if text, ok := b["text"].(string); ok && text != "" {
								events <- Event{Type: EventTextDelta, Text: text}
								textEmitted = true
							}
						}

						// Tool use interception!
						if blockType == "tool_use" {
							name, _ := b["name"].(string)
							id, _ := b["id"].(string)
							inputRaw, _ := json.Marshal(b["input"])
							tc := ToolCall{
								ID:    id,
								Name:  name,
								Input: inputRaw,
							}
							events <- Event{Type: EventToolCallStart, ToolCall: &tc}
						}
					}
				}
			}

		case "result":
			if sid, ok := event["session_id"].(string); ok {
				p.sessionID = sid
			}
			if result, ok := event["result"]; ok && !textEmitted {
				switch r := result.(type) {
				case string:
					if r != "" {
						events <- Event{Type: EventTextDelta, Text: r}
					}
				case map[string]interface{}:
					if text, ok := r["result"].(string); ok && text != "" {
						events <- Event{Type: EventTextDelta, Text: text}
					}
				}
			}

		case "error":
			errMsg := "Unknown error"
			if e, ok := event["error"].(map[string]interface{}); ok {
				if m, ok := e["message"].(string); ok {
					errMsg = m
				}
			}
			events <- Event{Type: EventError, Error: fmt.Errorf("%s", errMsg)}
			close(done)
			cmd.Wait()
			return
		}
	}

	close(done)

	if err := cmd.Wait(); err != nil {
		stderrStr := stderrBuf.String()
		if stderrStr != "" {
			events <- Event{Type: EventError, Error: fmt.Errorf("claude exited: %s", stderrStr)}
			return
		}
	}

	events <- Event{Type: EventDone}
}

// handleControlRequest parses a control_request event, emits a PermissionRequest event,
// and spawns a goroutine that waits for the user response and writes it to stdin.
func (p *ClaudeCodeProvider) handleControlRequest(event map[string]interface{}, stdinPipe io.WriteCloser, events chan<- Event) {
	requestID, _ := event["request_id"].(string)

	// Parse the nested request object
	reqObj, _ := event["request"].(map[string]interface{})
	if reqObj == nil {
		return
	}

	toolName, _ := reqObj["tool_name"].(string)
	toolUseID, _ := reqObj["tool_use_id"].(string)

	var inputRaw json.RawMessage
	if inp, ok := reqObj["input"]; ok {
		inputRaw, _ = json.Marshal(inp)
	}

	responseCh := make(chan PermissionResponse, 1)
	permReq := &PermissionRequest{
		RequestID:  requestID,
		ToolName:   toolName,
		ToolUseID:  toolUseID,
		Input:      inputRaw,
		ResponseCh: responseCh,
	}

	events <- Event{Type: EventPermissionRequest, PermissionReq: permReq}

	// Goroutine waits for the response and writes it to the subprocess stdin.
	// The subprocess BLOCKS until this response is written.
	go func() {
		resp := <-responseCh

		var controlResp map[string]interface{}
		if resp.Allow {
			controlResp = map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"subtype":    "success",
					"request_id": requestID,
					"response": map[string]interface{}{
						"behavior":     "allow",
						"updatedInput": map[string]interface{}{},
						"toolUseID":    toolUseID,
					},
				},
			}
		} else {
			msg := resp.Message
			if msg == "" {
				msg = "User denied"
			}
			controlResp = map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"subtype":    "success",
					"request_id": requestID,
					"response": map[string]interface{}{
						"behavior":  "deny",
						"message":   msg,
						"toolUseID": toolUseID,
					},
				},
			}
		}

		data, err := json.Marshal(controlResp)
		if err != nil {
			return
		}
		data = append(data, '\n')
		stdinPipe.Write(data)
	}()
}

// ResetSession clears the session ID for a fresh context.
func (p *ClaudeCodeProvider) ResetSession() {
	p.sessionID = ""
}

// SessionID returns the current Claude session ID (for persistence).
func (p *ClaudeCodeProvider) SessionID() string {
	return p.sessionID
}

// SetSessionID sets the Claude session ID (loaded from .defer/).
func (p *ClaudeCodeProvider) SetSessionID(id string) {
	p.sessionID = id
}

// GetModel returns the configured model name.
func (p *ClaudeCodeProvider) GetModel() string {
	return p.model
}
