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

// TestHeadlessAppendSystemPromptCoversContaminationSources — the phase-
// agnostic appendix must explicitly forbid invoking plugin skills and
// interactive tools, which are the specific contamination vectors we've
// hit from the superpowers plugin.
func TestHeadlessAppendSystemPromptCoversContaminationSources(t *testing.T) {
	if !strings.Contains(headlessAppendSystemPrompt, "plugin skills") {
		t.Error("headlessAppendSystemPrompt must mention plugin skills")
	}
	for _, forbidden := range []string{"TodoWrite", "AskUserQuestion", "Task"} {
		if !strings.Contains(headlessAppendSystemPrompt, forbidden) {
			t.Errorf("headlessAppendSystemPrompt should tell the model not to use %q", forbidden)
		}
	}
	if !strings.Contains(headlessAppendSystemPrompt, "SessionStart") {
		t.Error("headlessAppendSystemPrompt should explicitly reference SessionStart injections")
	}
	if !strings.Contains(headlessAppendSystemPrompt, "sub-agents") {
		t.Error("headlessAppendSystemPrompt should forbid spawning sub-agents")
	}
}

// TestStrictAppendSystemPromptExtendsHeadless — strict mode's appendix
// must include the entire headless guidance as a prefix AND add the
// DECIDED protocol on top. This catches regressions where a refactor
// might drop one or the other.
func TestStrictAppendSystemPromptExtendsHeadless(t *testing.T) {
	if !strings.HasPrefix(strictAppendSystemPrompt, headlessAppendSystemPrompt) {
		t.Error("strictAppendSystemPrompt should start with the full headlessAppendSystemPrompt text")
	}
	if !strings.Contains(strictAppendSystemPrompt, "DECIDED") {
		t.Error("strictAppendSystemPrompt should reinforce the DECIDED protocol")
	}
	// Still must name the allowed executor tools so the model knows what's
	// available after the tool list restriction.
	for _, tool := range []string{"Write", "Edit", "Read", "Glob", "Grep"} {
		if !strings.Contains(strictAppendSystemPrompt, tool) {
			t.Errorf("strictAppendSystemPrompt should list %q as an available tool", tool)
		}
	}
}

// TestStrictAllowedToolsIncludesEssentials — regression guard so the
// allow list keeps the file-write family and read-only exploration tools
// that defer's executor actually needs. Also checks that contamination
// tools (Skill, Task, ToolSearch, etc.) are NOT included — the whole
// point of switching from denylist to allowlist was that the denylist
// approach let the model bypass via ToolSearch(select:Skill).
func TestStrictAllowedToolsIncludesEssentials(t *testing.T) {
	// Must-have tools for implementation:
	required := []string{"Write", "Edit", "Read", "Glob", "Grep"}
	for _, tool := range required {
		if !strings.Contains(strictAllowedTools, tool) {
			t.Errorf("strictAllowedTools should allow %q — defer executor needs it for implementation", tool)
		}
	}
	// Must-NOT-have tools (contamination sources):
	forbidden := []string{"Bash", "Skill", "Task", "AskUserQuestion", "ToolSearch", "TodoWrite", "EnterPlanMode"}
	for _, tool := range forbidden {
		// Use comma-delimited match so "Task" doesn't match "TaskOutput" etc.
		parts := strings.Split(strictAllowedTools, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == tool {
				t.Errorf("strictAllowedTools must not include %q — it's a contamination source", tool)
			}
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
