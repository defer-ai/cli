package permissions

import (
	"testing"

	"github.com/defer-ai/cli/internal/agent"
)

func TestShouldPromptSkip(t *testing.T) {
	for _, action := range []ToolAction{ActionRead, ActionWrite, ActionExecute} {
		if ShouldPrompt(agent.CareLevelSkip, action) {
			t.Errorf("skip + %s: should never prompt", action)
		}
	}
}

func TestShouldPromptLow(t *testing.T) {
	for _, action := range []ToolAction{ActionRead, ActionWrite, ActionExecute} {
		if ShouldPrompt(agent.CareLevelLow, action) {
			t.Errorf("low + %s: should never prompt", action)
		}
	}
}

func TestShouldPromptMedium(t *testing.T) {
	tests := []struct {
		action ToolAction
		want   bool
	}{
		{ActionRead, false},
		{ActionWrite, false},
		{ActionExecute, true},
	}
	for _, tt := range tests {
		got := ShouldPrompt(agent.CareLevelMedium, tt.action)
		if got != tt.want {
			t.Errorf("medium + %s: got %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestShouldPromptHigh(t *testing.T) {
	tests := []struct {
		action ToolAction
		want   bool
	}{
		{ActionRead, false},
		{ActionWrite, true},
		{ActionExecute, true},
	}
	for _, tt := range tests {
		got := ShouldPrompt(agent.CareLevelHigh, tt.action)
		if got != tt.want {
			t.Errorf("high + %s: got %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestShouldPromptParanoid(t *testing.T) {
	for _, action := range []ToolAction{ActionRead, ActionWrite, ActionExecute} {
		if !ShouldPrompt(agent.CareLevelParanoid, action) {
			t.Errorf("paranoid + %s: should always prompt", action)
		}
	}
}

func TestShouldPromptUnknownCareLevel(t *testing.T) {
	// Unknown care level defaults to medium behavior
	unknown := agent.CareLevel("unknown")

	if ShouldPrompt(unknown, ActionRead) {
		t.Error("unknown + read: should not prompt (medium default)")
	}
	if ShouldPrompt(unknown, ActionWrite) {
		t.Error("unknown + write: should not prompt (medium default)")
	}
	if !ShouldPrompt(unknown, ActionExecute) {
		t.Error("unknown + execute: should prompt (medium default)")
	}
}

func TestClassifyToolReadTools(t *testing.T) {
	for _, name := range []string{"Glob", "glob", "GLOB", "Grep", "grep", "Read", "read"} {
		got := ClassifyTool(name)
		if got != ActionRead {
			t.Errorf("ClassifyTool(%q) = %q, want %q", name, got, ActionRead)
		}
	}
}

func TestClassifyToolWriteTools(t *testing.T) {
	for _, name := range []string{"Write", "write", "Edit", "edit"} {
		got := ClassifyTool(name)
		if got != ActionWrite {
			t.Errorf("ClassifyTool(%q) = %q, want %q", name, got, ActionWrite)
		}
	}
}

func TestClassifyToolExecuteTools(t *testing.T) {
	for _, name := range []string{"Bash", "bash", "BASH"} {
		got := ClassifyTool(name)
		if got != ActionExecute {
			t.Errorf("ClassifyTool(%q) = %q, want %q", name, got, ActionExecute)
		}
	}
}

func TestClassifyToolUnknownDefaultsToExecute(t *testing.T) {
	for _, name := range []string{"CustomTool", "unknown", "Deploy", ""} {
		got := ClassifyTool(name)
		if got != ActionExecute {
			t.Errorf("ClassifyTool(%q) = %q, want %q (unknown defaults to execute)", name, got, ActionExecute)
		}
	}
}

func TestShouldPromptIntegration(t *testing.T) {
	// End-to-end: classify a tool and check permission
	tests := []struct {
		care     agent.CareLevel
		tool     string
		wantProm bool
	}{
		{agent.CareLevelSkip, "Bash", false},
		{agent.CareLevelLow, "Write", false},
		{agent.CareLevelMedium, "Read", false},
		{agent.CareLevelMedium, "Bash", true},
		{agent.CareLevelHigh, "Edit", true},
		{agent.CareLevelHigh, "Glob", false},
		{agent.CareLevelParanoid, "Grep", true},
		{agent.CareLevelParanoid, "Bash", true},
	}
	for _, tt := range tests {
		action := ClassifyTool(tt.tool)
		got := ShouldPrompt(tt.care, action)
		if got != tt.wantProm {
			t.Errorf("care=%s tool=%s action=%s: ShouldPrompt=%v, want %v",
				tt.care, tt.tool, action, got, tt.wantProm)
		}
	}
}
