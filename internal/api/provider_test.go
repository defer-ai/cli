package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClaudeCodeProviderImplementsProvider verifies that
// *ClaudeCodeProvider satisfies the Provider interface.
func TestClaudeCodeProviderImplementsProvider(t *testing.T) {
	var _ Provider = (*ClaudeCodeProvider)(nil)
}

// TestOpenAIProviderImplementsProvider verifies that
// *OpenAIProvider satisfies the Provider interface.
func TestOpenAIProviderImplementsProvider(t *testing.T) {
	var _ Provider = (*OpenAIProvider)(nil)
}

func TestClaudeCodeProviderGetModel(t *testing.T) {
	p := NewClaudeCodeProvider("haiku")
	if p.GetModel() != "haiku" {
		t.Errorf("GetModel() = %q, want haiku", p.GetModel())
	}
}

func TestOpenAIProviderGetModel(t *testing.T) {
	p := NewOpenAIProvider("openai", "test-key", "gpt-4o")
	if p.GetModel() != "gpt-4o" {
		t.Errorf("GetModel() = %q, want gpt-4o", p.GetModel())
	}
}

func TestClaudeCodeProviderResetSession(t *testing.T) {
	p := NewClaudeCodeProvider("sonnet")
	p.sessionID = "test-session"
	p.ResetSession()
	if p.sessionID != "" {
		t.Errorf("sessionID = %q after ResetSession, want empty", p.sessionID)
	}
}

func TestOpenAIProviderResetSession(t *testing.T) {
	p := NewOpenAIProvider("openai", "key", "gpt-4o")
	// Should not panic
	p.ResetSession()
}

// TestClaudeCodeProviderStrictModeDefault — fresh providers created via
// the public constructors should have StrictMode off. The executor opts
// into strict mode explicitly for execute-phase invocations, and tests
// that construct providers shouldn't inherit behavior they didn't ask for.
func TestClaudeCodeProviderStrictModeDefault(t *testing.T) {
	p := NewClaudeCodeProvider("sonnet")
	if p.StrictMode {
		t.Error("NewClaudeCodeProvider should create provider with StrictMode=false")
	}
	p2 := NewClaudeCodeProviderWithCWD("sonnet", "/tmp")
	if p2.StrictMode {
		t.Error("NewClaudeCodeProviderWithCWD should create provider with StrictMode=false")
	}
}

// TestStrictHookSettingsJSONIsValid — the hook config embedded in the
// binary must be parseable as JSON (Claude Code will silently ignore a
// malformed settings file) and contain the markers that make it do what
// we want.
func TestStrictHookSettingsJSONIsValid(t *testing.T) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strictHookSettingsJSON), &parsed); err != nil {
		t.Fatalf("strictHookSettingsJSON is not valid JSON: %v", err)
	}
	// Must target PreToolUse on the file-modifying tools.
	if !strings.Contains(strictHookSettingsJSON, `"PreToolUse"`) {
		t.Error("hook JSON missing PreToolUse event")
	}
	if !strings.Contains(strictHookSettingsJSON, "Write|Edit|MultiEdit|NotebookEdit") {
		t.Error("hook JSON missing the write-family matcher")
	}
	// The reminder must use hookSpecificOutput/additionalContext — plain
	// stdout from a hook is discarded by Claude Code, so anything else
	// won't reach the model.
	if !strings.Contains(strictHookSettingsJSON, "hookSpecificOutput") {
		t.Error("hook JSON must use hookSpecificOutput format")
	}
	if !strings.Contains(strictHookSettingsJSON, "additionalContext") {
		t.Error("hook JSON must use additionalContext to inject the reminder")
	}
	if !strings.Contains(strictHookSettingsJSON, "DECIDED:") {
		t.Error("hook JSON reminder must reference the DECIDED protocol")
	}
}

// TestEnsureStrictHookFile — writes the hook file to HOME/.defer/strict-hook.json
// on first call, returns the existing path on subsequent calls.
func TestEnsureStrictHookFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// First call writes the file.
	path, err := ensureStrictHookFile()
	if err != nil {
		t.Fatalf("first ensureStrictHookFile() error: %v", err)
	}
	want := filepath.Join(tmpHome, ".defer", "strict-hook.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read hook file: %v", err)
	}
	if string(data) != strictHookSettingsJSON {
		t.Error("hook file contents do not match strictHookSettingsJSON constant")
	}

	// Second call returns the existing path without rewriting.
	origStat, _ := os.Stat(path)
	path2, err := ensureStrictHookFile()
	if err != nil {
		t.Fatalf("second ensureStrictHookFile() error: %v", err)
	}
	if path2 != path {
		t.Errorf("second call returned different path: %q vs %q", path2, path)
	}
	newStat, _ := os.Stat(path2)
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Error("second call rewrote the file (modtime changed) — should have reused the existing one")
	}
}

// TestStrictAppendSystemPromptMentionsRestrictions — guard so the appendix
// keeps explaining which tools are gone. Without this, reformatting could
// accidentally strip the explanation and leave the model confused when it
// tries to call a tool that isn't there or invoke a plugin skill.
func TestStrictAppendSystemPromptMentionsRestrictions(t *testing.T) {
	for _, tool := range []string{"Bash", "Skill", "Task", "AskUserQuestion"} {
		if !strings.Contains(strictAppendSystemPrompt, tool) {
			t.Errorf("strictAppendSystemPrompt should mention %q explicitly", tool)
		}
	}
	if !strings.Contains(strictAppendSystemPrompt, "Write") || !strings.Contains(strictAppendSystemPrompt, "Edit") {
		t.Error("strictAppendSystemPrompt should point the model at Write/Edit tools")
	}
	if !strings.Contains(strictAppendSystemPrompt, "DECIDED") {
		t.Error("strictAppendSystemPrompt should reinforce the DECIDED protocol")
	}
	if !strings.Contains(strictAppendSystemPrompt, "plugin skills") {
		t.Error("strictAppendSystemPrompt should explicitly call out plugin skills to prevent brainstorming/TDD/etc. intrusions")
	}
}

// TestStrictDisallowedToolsIncludesAllProblemTools — regression guard so
// the disallow list stays comprehensive. Each of these tools is a known
// source of executor-phase contamination documented on the constant.
func TestStrictDisallowedToolsIncludesAllProblemTools(t *testing.T) {
	required := []string{"Bash", "Skill", "Task", "AskUserQuestion", "EnterPlanMode"}
	for _, tool := range required {
		if !strings.Contains(strictDisallowedTools, tool) {
			t.Errorf("strictDisallowedTools should block %q", tool)
		}
	}
}

func TestMapModel(t *testing.T) {
	tests := []struct {
		model    string
		provider string
		want     string
	}{
		{"sonnet", "openai", "gpt-4o"},
		{"haiku", "openai", "gpt-4o-mini"},
		{"gpt-4o", "openai", "gpt-4o"},
		{"sonnet", "groq", "llama-3.3-70b-versatile"},
		{"haiku", "groq", "llama-3.1-8b-instant"},
		{"sonnet", "unknown", "sonnet"},
	}
	for _, tt := range tests {
		t.Run(tt.model+"/"+tt.provider, func(t *testing.T) {
			got := mapModel(tt.model, tt.provider)
			if got != tt.want {
				t.Errorf("mapModel(%q, %q) = %q, want %q", tt.model, tt.provider, got, tt.want)
			}
		})
	}
}
