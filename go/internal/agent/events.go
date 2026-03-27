package agent

import "github.com/defer-ai/cli/internal/decision"

// EventType classifies agent/executor events for the TUI.
type EventType int

const (
	AgentStateChanged  EventType = iota // decomposition agent state updated
	AgentDecisionsReady                 // decomposition complete, decisions available
	ExecStateChanged                    // domain executor state changed
	ExecDecisionStored                  // executor logged a new decision
	AllExecutorsDone                    // all domain executors finished
	SwarmComplete                       // swarm expansion finished
)

// Event is sent from agent goroutines to the TUI.
type Event struct {
	Type       EventType
	ExecutorID string              // for executor events
	Decisions  []decision.Decision // for DecisionsReady / DecisionStored
	Error      error
}
