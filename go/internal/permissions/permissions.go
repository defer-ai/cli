package permissions

import (
	"strings"

	"github.com/defer-ai/cli/internal/agent"
)

// ToolAction classifies what a tool does.
type ToolAction string

const (
	ActionRead    ToolAction = "read"    // Glob, Grep, Read
	ActionWrite   ToolAction = "write"   // Write, Edit
	ActionExecute ToolAction = "execute" // Bash
)

// ShouldPrompt returns true if the care level requires user confirmation for this action.
//
// Rules:
//   - skip:     never prompt (all auto-allowed)
//   - low:      never prompt
//   - medium:   prompt for execute (Bash)
//   - high:     prompt for write + execute
//   - paranoid: prompt for everything
func ShouldPrompt(care agent.CareLevel, action ToolAction) bool {
	switch care {
	case agent.CareLevelSkip:
		return false
	case agent.CareLevelLow:
		return false
	case agent.CareLevelMedium:
		return action == ActionExecute
	case agent.CareLevelHigh:
		return action == ActionWrite || action == ActionExecute
	case agent.CareLevelParanoid:
		return true
	default:
		// Unknown care level: default to medium behavior
		return action == ActionExecute
	}
}

// ClassifyTool returns the action type for a tool name.
// Tool names are matched case-insensitively.
func ClassifyTool(toolName string) ToolAction {
	switch strings.ToLower(toolName) {
	case "glob", "grep", "read":
		return ActionRead
	case "write", "edit":
		return ActionWrite
	case "bash":
		return ActionExecute
	default:
		// Unknown tools default to execute (most restrictive)
		return ActionExecute
	}
}
