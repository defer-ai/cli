package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

// Agent lifecycle messages
type AgentStateChangedMsg struct{}
type AgentDecisionsReadyMsg struct{ Decisions []decision.Decision }
type ExecutorStateChangedMsg struct{ ExecutorID string }
type ExecutorDecisionStoredMsg struct {
	ExecutorID string
	Decisions  []decision.Decision
}
type AllExecutorsDoneMsg struct{}
type ToolActivityMsg struct{ Description string }

// UI messages
type TaskSubmittedMsg struct{ Task string }
type PrioritiesConfirmedMsg struct{ Priorities map[string]agent.CareLevel }
type TickMsg struct{ Time time.Time }

// Tree action messages (emitted by TreeModel, handled by app)
type ReviseDecisionMsg struct{ ID, NewAnswer string }
type AskDecisionMsg struct{ ID, Question string }
type WhyDecisionMsg struct{ ID, Label string }
type SuggestDecisionMsg struct{ ID string }
type WhyResponseMsg struct{ Text string }
type ChatMessageMsg struct{ Text string }
type ChatResponseMsg struct{ Text string }
type SuggestResponseMsg struct {
	ID      string
	Options []decision.DecisionOption
}
type TogglePermissionsMsg struct{ Bypass bool }
type CheckAllDecidedMsg struct{}
type ImplicitInvalidationMsg struct {
	IDs    []string
	Reason string
}

// BridgeAgentEvent converts an agent.Event to a tea.Msg.
func BridgeAgentEvent(ev agent.Event) tea.Msg {
	switch ev.Type {
	case agent.AgentStateChanged:
		return AgentStateChangedMsg{}
	case agent.AgentDecisionsReady:
		return AgentDecisionsReadyMsg{Decisions: ev.Decisions}
	case agent.ExecStateChanged:
		return ExecutorStateChangedMsg{ExecutorID: ev.ExecutorID}
	case agent.ExecDecisionStored:
		return ExecutorDecisionStoredMsg{ExecutorID: ev.ExecutorID, Decisions: ev.Decisions}
	case agent.ExecToolActivity:
		return ToolActivityMsg{Description: ev.ToolActivity}
	case agent.AllExecutorsDone:
		return AllExecutorsDoneMsg{}
	default:
		return nil
	}
}

func ListenForEvents(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok || msg == nil {
			return nil
		}
		return msg
	}
}

// safeSend sends a message to the event channel without panicking on closed channel or cancelled context.
func safeSend(ctx context.Context, ch chan<- tea.Msg, msg tea.Msg) {
	select {
	case <-ctx.Done():
		return
	case ch <- msg:
	}
}

func DoTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}
