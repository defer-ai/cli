package api

import (
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
