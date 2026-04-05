package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HookEvent identifies when a hook fires.
type HookEvent string

const (
	PreDecision     HookEvent = "pre-decision"     // before auto-deciding
	PostDecision    HookEvent = "post-decision"     // after decision confirmed
	PreExecute      HookEvent = "pre-execute"       // before executor starts
	PostExecute     HookEvent = "post-execute"      // after executor completes
	DecisionChanged HookEvent = "decision-changed"  // when user revises
)

// HookConfig defines a hook action.
type HookConfig struct {
	Command string `json:"command,omitempty"` // bash command
	URL     string `json:"url,omitempty"`     // webhook URL
}

// webhookPayload is the JSON body sent to webhook URLs.
type webhookPayload struct {
	Event string            `json:"event"`
	Env   map[string]string `json:"env"`
}

// RunHooks executes all hooks for an event.
// Environment variables: DEFER_EVENT, DEFER_DECISION_ID, DEFER_DECISION_ANSWER, DEFER_CWD
// All hooks are run in order; errors are collected and returned as a combined error.
func RunHooks(event HookEvent, hooks []HookConfig, env map[string]string) error {
	var errs []error

	for _, h := range hooks {
		if h.Command == "" && h.URL == "" {
			continue // skip empty hooks silently
		}

		if h.Command != "" {
			if err := runCommandHook(event, h.Command, env); err != nil {
				errs = append(errs, fmt.Errorf("command hook %q: %w", h.Command, err))
			}
		}

		if h.URL != "" {
			if err := runURLHook(event, h.URL, env); err != nil {
				errs = append(errs, fmt.Errorf("url hook %q: %w", h.URL, err))
			}
		}
	}

	return errors.Join(errs...)
}

// runCommandHook runs a bash command with a 10s timeout and injected env vars.
func runCommandHook(event HookEvent, command string, env map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Inherit current environment and add hook-specific vars
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("DEFER_EVENT=%s", string(event)))
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out after 10s (output: %s)", strings.TrimSpace(string(output)))
	}
	if err != nil {
		return fmt.Errorf("%w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// runURLHook POSTs JSON to a webhook URL with a 5s timeout.
func runURLHook(event HookEvent, url string, env map[string]string) error {
	payload := webhookPayload{
		Event: string(event),
		Env:   env,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("POST %s: status %d", url, resp.StatusCode)
	}

	return nil
}
