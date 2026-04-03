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
	cmd.Stderr = &strings.Builder{}

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
