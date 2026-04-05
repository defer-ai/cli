package permissions

import (
	"testing"

	"github.com/defer-ai/cli/internal/agent"
)

func TestShouldPromptAuto(t *testing.T) {
	for _, action := range []ToolAction{ActionRead, ActionWrite, ActionExecute} {
		if ShouldPrompt(agent.CareLevelAuto, action) {
			t.Errorf("auto + %s: should never prompt", action)
		}
	}
}

func TestShouldPromptReview(t *testing.T) {
	tests := []struct {
		action ToolAction
		want   bool
	}{
		{ActionRead, false},
		{ActionWrite, true},
		{ActionExecute, true},
	}
	for _, tt := range tests {
		got := ShouldPrompt(agent.CareLevelReview, tt.action)
		if got != tt.want {
			t.Errorf("review + %s: got %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestShouldPromptUnknownCareLevel(t *testing.T) {
	// Unknown care level defaults to auto behavior (never prompt)
	unknown := agent.CareLevel("unknown")

	for _, action := range []ToolAction{ActionRead, ActionWrite, ActionExecute} {
		if ShouldPrompt(unknown, action) {
			t.Errorf("unknown + %s: should not prompt (auto default)", action)
		}
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
		{agent.CareLevelAuto, "Bash", false},
		{agent.CareLevelAuto, "Write", false},
		{agent.CareLevelAuto, "Read", false},
		{agent.CareLevelReview, "Read", false},
		{agent.CareLevelReview, "Edit", true},
		{agent.CareLevelReview, "Glob", false},
		{agent.CareLevelReview, "Grep", false},
		{agent.CareLevelReview, "Bash", true},
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
