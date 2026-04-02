package agent

import (
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// EventType classifies agent/executor events for the TUI.
type EventType int

const (
	AgentStateChanged    EventType = iota // decomposition agent state updated
	AgentDecisionsReady                   // decomposition complete, decisions available
	ExecStateChanged                      // domain executor state changed
	ExecDecisionStored                    // executor logged a new decision
	ExecToolActivity                      // executor tool call (for live feed)
	ExecPermissionRequest                 // executor needs permission for a tool
	AllExecutorsDone                      // all domain executors finished
)

// Event is sent from agent goroutines to the TUI.
type Event struct {
	Type          EventType
	ExecutorID    string              // for executor events
	Decisions     []decision.Decision // for DecisionsReady / DecisionStored
	ToolActivity  string              // human-readable tool description for feed
	PermissionReq *api.PermissionRequest // for ExecPermissionRequest
	Error         error
}
