package agent

import "github.com/defer-ai/cli/internal/decision"

// SetAgentDecisions sets the agent's decisions directly (for testing).
func SetAgentDecisions(a *Agent, decs []decision.Decision) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Decisions = make([]decision.Decision, len(decs))
	copy(a.state.Decisions, decs)
	a.state.Status = StatusDone
}

// SetManagerAgent sets the manager's agent directly (for testing).
func SetManagerAgent(m *Manager, a *Agent) {
	m.agent = a
}
