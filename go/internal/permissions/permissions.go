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
//   - auto:   never prompt
//   - review: prompt for write + execute
func ShouldPrompt(care agent.CareLevel, action ToolAction) bool {
	switch care {
	case agent.CareLevelAuto:
		return false
	case agent.CareLevelReview:
		return action == ActionWrite || action == ActionExecute
	default:
		// Unknown care level: default to auto behavior
		return false
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
