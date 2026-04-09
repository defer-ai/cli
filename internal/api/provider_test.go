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

// TestEnsureStrictMCPConfigFile — writes a valid MCP server config to
// HOME/.defer/strict-mcp.json, naming the current defer binary as the
// "defer" server. Regenerated on every call so the binary path stays
// current across upgrades.
//
// Note: os.UserHomeDir() reads HOME on Linux/macOS but USERPROFILE on
// Windows. Set both so the test redirects to the temp dir on every
// platform.
func TestEnsureStrictMCPConfigFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	path, err := ensureStrictMCPConfigFile()
	if err != nil {
		t.Fatalf("ensureStrictMCPConfigFile() error: %v", err)
	}
	want := filepath.Join(tmpHome, ".defer", "strict-mcp.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	// File must be valid JSON with the Claude Code MCP server shape.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read mcp config: %v", err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("strict-mcp.json is not valid JSON: %v\n%s", err, data)
	}
	if _, ok := parsed.MCPServers["defer"]; !ok {
		t.Error(`mcp config must register a server named "defer"`)
	}
	if parsed.MCPServers["defer"].Command == "" {
		t.Error("defer server command must be set to a defer binary path")
	}
	if len(parsed.MCPServers["defer"].Args) < 2 || parsed.MCPServers["defer"].Args[0] != "serve" || parsed.MCPServers["defer"].Args[1] != "--mcp" {
		t.Errorf("defer server args must be [serve, --mcp], got %v", parsed.MCPServers["defer"].Args)
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
// must include the entire headless guidance as a prefix AND describe
// the MCP gated-write protocol that replaced the old DECIDED-inline
// approach. This catches regressions where a refactor might drop one
// or the other.
func TestStrictAppendSystemPromptExtendsHeadless(t *testing.T) {
	if !strings.HasPrefix(strictAppendSystemPrompt, headlessAppendSystemPrompt) {
		t.Error("strictAppendSystemPrompt should start with the full headlessAppendSystemPrompt text")
	}
	// Must describe the new MCP-based write protocol explicitly so the
	// model knows that "Write" is not in its toolkit and mcp__defer__
	// write_file is.
	for _, marker := range []string{
		"mcp__defer__register_decision",
		"mcp__defer__write_file",
		"FILE WRITE PROTOCOL",
		"not available in this environment",
	} {
		if !strings.Contains(strictAppendSystemPrompt, marker) {
			t.Errorf("strictAppendSystemPrompt should contain %q", marker)
		}
	}
	// Must still reference the read-only tools so the model knows it can
	// explore with them.
	for _, tool := range []string{"Read", "Glob", "Grep"} {
		if !strings.Contains(strictAppendSystemPrompt, tool) {
			t.Errorf("strictAppendSystemPrompt should list %q as a read-only tool", tool)
		}
	}
}

// TestStrictAllowedToolsIncludesEssentials — regression guard for the
// --tools allowlist. After switching to the MCP gated-write
// architecture, strict mode keeps only the read-only built-in tools;
// file modification happens exclusively through the MCP tools, which
// Claude Code auto-includes from the loaded --mcp-config servers
// regardless of the --tools filter.
func TestStrictAllowedToolsIncludesEssentials(t *testing.T) {
	// Must-have tools for exploration:
	required := []string{"Read", "Glob", "Grep"}
	for _, tool := range required {
		if !strings.Contains(strictAllowedTools, tool) {
			t.Errorf("strictAllowedTools should allow %q — defer executor needs it for exploration", tool)
		}
	}
	// Must-NOT-have tools:
	//   - native write tools (Write/Edit/NotebookEdit) — replaced by the
	//     MCP gated-write path
	//   - every contamination source (Bash/Skill/Task/AskUserQuestion/
	//     ToolSearch/TodoWrite/EnterPlanMode)
	forbidden := []string{
		"Write", "Edit", "NotebookEdit",
		"Bash", "Skill", "Task", "AskUserQuestion", "ToolSearch", "TodoWrite", "EnterPlanMode",
	}
	for _, tool := range forbidden {
		// Use comma-delimited match so "Task" doesn't match "TaskOutput" etc.
		parts := strings.Split(strictAllowedTools, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == tool {
				t.Errorf("strictAllowedTools must not include %q in strict mode", tool)
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
