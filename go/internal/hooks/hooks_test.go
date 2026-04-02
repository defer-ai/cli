package hooks

import (
	"strings"
	"testing"
)

func TestCommandHookSuccess(t *testing.T) {
	hooks := []HookConfig{
		{Command: "echo hello"},
	}

	err := RunHooks(PreDecision, hooks, map[string]string{
		"DEFER_DECISION_ID": "@STA-0001",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCommandHookReceivesEnvVars(t *testing.T) {
	// The command writes env vars to stdout; we verify it doesn't error.
	// The DEFER_EVENT var is always set by RunHooks.
	hooks := []HookConfig{
		{Command: "test \"$DEFER_EVENT\" = \"pre-decision\" && test \"$DEFER_CWD\" = \"/tmp/test\""},
	}

	env := map[string]string{
		"DEFER_CWD": "/tmp/test",
	}

	err := RunHooks(PreDecision, hooks, env)
	if err != nil {
		t.Errorf("env vars not propagated correctly: %v", err)
	}
}

func TestCommandHookTimeout(t *testing.T) {
	hooks := []HookConfig{
		{Command: "sleep 30"},
	}

	err := RunHooks(PreDecision, hooks, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestCommandHookFailure(t *testing.T) {
	hooks := []HookConfig{
		{Command: "exit 1"},
	}

	err := RunHooks(PostDecision, hooks, nil)
	if err == nil {
		t.Fatal("expected error for failing command, got nil")
	}
}

func TestURLHookEmptyURL(t *testing.T) {
	// Empty URL hook should be skipped silently
	hooks := []HookConfig{
		{URL: ""},
	}

	err := RunHooks(PreExecute, hooks, nil)
	if err != nil {
		t.Errorf("empty URL hook should be skipped, got: %v", err)
	}
}

func TestURLHookInvalidURL(t *testing.T) {
	// An unreachable URL should return an error
	hooks := []HookConfig{
		{URL: "http://127.0.0.1:1/nonexistent"},
	}

	err := RunHooks(PostExecute, hooks, map[string]string{
		"DEFER_DECISION_ID": "TES-0001",
	})
	if err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
}

func TestEmptyHookSkipped(t *testing.T) {
	hooks := []HookConfig{
		{}, // both Command and URL are empty
	}

	err := RunHooks(DecisionChanged, hooks, nil)
	if err != nil {
		t.Errorf("empty hook should be skipped, got: %v", err)
	}
}

func TestMultipleHooksRunInOrder(t *testing.T) {
	// Run three hooks: first two succeed, third fails.
	// All should run; we get back the error from the third.
	hooks := []HookConfig{
		{Command: "true"},
		{Command: "true"},
		{Command: "false"},
	}

	err := RunHooks(PreDecision, hooks, nil)
	if err == nil {
		t.Fatal("expected error from third hook")
	}
	// Only one error (from the "false" command)
	if strings.Count(err.Error(), "command hook") != 1 {
		t.Errorf("expected exactly 1 hook error, got: %v", err)
	}
}

func TestMultipleHooksAllSucceed(t *testing.T) {
	hooks := []HookConfig{
		{Command: "true"},
		{Command: "echo ok"},
		{Command: "true"},
	}

	err := RunHooks(PostDecision, hooks, nil)
	if err != nil {
		t.Errorf("all hooks should succeed, got: %v", err)
	}
}

func TestMultipleHooksCollectAllErrors(t *testing.T) {
	hooks := []HookConfig{
		{Command: "false"},
		{Command: "false"},
	}

	err := RunHooks(PreExecute, hooks, nil)
	if err == nil {
		t.Fatal("expected errors from both hooks")
	}
	// Both errors should be present
	if strings.Count(err.Error(), "command hook") != 2 {
		t.Errorf("expected 2 hook errors, got: %v", err)
	}
}

func TestNilHooksSlice(t *testing.T) {
	err := RunHooks(PreDecision, nil, nil)
	if err != nil {
		t.Errorf("nil hooks should return nil error, got: %v", err)
	}
}

func TestNilEnvMap(t *testing.T) {
	hooks := []HookConfig{
		{Command: "true"},
	}

	err := RunHooks(PreDecision, hooks, nil)
	if err != nil {
		t.Errorf("nil env should work, got: %v", err)
	}
}
