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

// ClaudeCodeProvider runs Claude Code as a subprocess.
// Used when no ANTHROPIC_API_KEY is set (user has Claude Code authenticated via subscription).
type ClaudeCodeProvider struct {
	model     string
	sessionID string
}

// NewClaudeCodeProvider creates a subprocess provider.
func NewClaudeCodeProvider(model string) *ClaudeCodeProvider {
	return &ClaudeCodeProvider{model: model}
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
// This replaces RunAgentLoop when using subprocess mode.
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

	if p.sessionID != "" {
		args = append(args, "--resume", p.sessionID)
	} else {
		args = append(args, "--system-prompt", systemPrompt)
	}

	args = append(args, userPrompt)

	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
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

// ResetSession clears the session ID for a fresh context.
func (p *ClaudeCodeProvider) ResetSession() {
	p.sessionID = ""
}

// GetModel returns the configured model name.
func (p *ClaudeCodeProvider) GetModel() string {
	return p.model
}
