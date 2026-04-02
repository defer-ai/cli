package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

// --- helpers ---

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func keyEsc() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEscape}
}

func keyTab() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

func keyCtrlC() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

func strPtr(s string) *string { return &s }

func fakeDecisions() []decision.Decision {
	return []decision.Decision{
		{
			ID: "@STA-0001", Category: "Stack", Question: "Backend language?",
			Options: []decision.DecisionOption{
				{Key: "A", Label: "TypeScript"},
				{Key: "B", Label: "Python"},
				{Key: "C", Label: "Choose for me"},
			},
			Source: "user",
		},
		{
			ID: "@STA-0002", Category: "Stack", Question: "Frontend framework?",
			Options: []decision.DecisionOption{
				{Key: "A", Label: "React"},
				{Key: "B", Label: "Vue"},
				{Key: "C", Label: "Choose for me"},
			},
			Source: "user",
		},
		{
			ID: "@UIX-0001", Category: "UI", Question: "CSS approach?",
			Options: []decision.DecisionOption{
				{Key: "A", Label: "Tailwind"},
				{Key: "B", Label: "CSS Modules"},
				{Key: "C", Label: "Choose for me"},
			},
			Source: "user",
		},
	}
}

// processCmd executes a tea.Cmd and returns the resulting message.
func processCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return cmd()
}

// updateModel is a convenience that calls Update and returns the model + cmd.
func updateModel(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	result, cmd := m.Update(msg)
	return result.(Model), cmd
}

// setupAtPriorities creates a model in the ViewPriorities state with a real
// manager and agent, without triggering network calls.
func setupAtPriorities(t *testing.T, decs []decision.Decision) Model {
	t.Helper()
	dir := t.TempDir()
	m := NewModel("", nil, dir)
	m.task = "test task"

	// Create manager with a pre-populated agent
	m.manager = agent.NewManager(nil, m.cwd)
	mgrAgent := agent.NewAgent("test task", nil, m.cwd)
	agent.SetAgentDecisions(mgrAgent, decs)
	agent.SetManagerAgent(m.manager, mgrAgent)

	// Move to priorities view
	m.tree.decisions = make([]decision.Decision, len(decs))
	copy(m.tree.decisions, decs)
	m.view = ViewPriorities
	m.priorities = NewPrioritiesModel(decs)
	m.priorities.width = 120
	m.priorities.height = 40

	return m
}

// setupAtTree creates a model at ViewTree state with decisions and priorities applied.
// It immediately cancels the context to prevent background goroutines from racing.
func setupAtTree(t *testing.T, decs []decision.Decision, priorities map[string]agent.CareLevel) Model {
	t.Helper()
	m := setupAtPriorities(t, decs)
	m, _ = updateModel(t, m, PrioritiesConfirmedMsg{Priorities: priorities})
	// Cancel context to stop any executor goroutines launched by PrioritiesConfirmedMsg.
	// This prevents data races from background goroutines accessing shared state.
	m.cancel()
	// Create a fresh context for subsequent operations in the test.
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// setupAtTreeNoExecutors creates a model at ViewTree state without triggering
// AutoDecide or executor launches. Decisions are set directly. Use this when
// you need pending decisions and don't want the auto-decide/executor logic.
func setupAtTreeNoExecutors(t *testing.T, decs []decision.Decision) Model {
	t.Helper()
	dir := t.TempDir()
	m := NewModel("", nil, dir)
	m.task = "test task"
	m.view = ViewTree
	m.tree.decisions = make([]decision.Decision, len(decs))
	copy(m.tree.decisions, decs)
	m.tree.width = 120
	m.tree.height = 40
	m.width = 120
	m.height = 40

	// Create a manager with the decisions
	m.manager = agent.NewManager(nil, m.cwd)
	mgrAgent := agent.NewAgent("test task", nil, m.cwd)
	agent.SetAgentDecisions(mgrAgent, decs)
	agent.SetManagerAgent(m.manager, mgrAgent)

	return m
}

// --- tests ---

func TestConversationStartsInChatMode(t *testing.T) {
	m := NewModel("", nil, t.TempDir())

	if m.view != ViewConversation {
		t.Fatalf("initial view = %d, want ViewConversation (%d)", m.view, ViewConversation)
	}

	// Chat mode should be active by default
	if m.tree.mode != tmChat {
		t.Fatalf("tree.mode = %d, want tmChat", m.tree.mode)
	}

	// Send window size
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 || m.height != 40 {
		t.Fatalf("dimensions = %dx%d, want 120x40", m.width, m.height)
	}

	// Type "hey" into the chat input
	for _, ch := range "hey" {
		m, _ = updateModel(t, m, keyRunes(string(ch)))
	}

	if m.tree.chatInput.Value() != "hey" {
		t.Fatalf("chatInput = %q, want %q", m.tree.chatInput.Value(), "hey")
	}

	// Press enter -- should produce ChatMessageMsg (goes to agent, not decomposition)
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, keyEnter())

	msg := processCmd(t, cmd)
	cmm, ok := msg.(ChatMessageMsg)
	if !ok {
		t.Fatalf("expected ChatMessageMsg, got %T", msg)
	}
	if cmm.Text != "hey" {
		t.Fatalf("text = %q, want %q", cmm.Text, "hey")
	}

	// The message should appear in chat log
	found := false
	for _, entry := range m.tree.chatLog {
		if entry.Type == "user" && entry.Text == "hey" {
			found = true
		}
	}
	if !found {
		t.Fatal("chat log should contain the user message")
	}

	// Task should NOT be set (casual chat doesn't trigger decomposition)
	if m.task != "" {
		t.Errorf("task should be empty after casual chat, got %q", m.task)
	}
}

func TestChatResponseWithDecisionsTriggersFlow(t *testing.T) {
	m := NewModel("", nil, t.TempDir())

	// Simulate a chat response that contains decisions
	m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "user", Text: "build a todo app"})

	resp := "Here are the decisions:\n\n```defer-decisions\n" +
		`[{"category": "Stack", "question": "Backend?", "options": [{"key": "A", "label": "Go"}, {"key": "B", "label": "Choose for me"}], "context": "Backend choice"}]` +
		"\n```\n"

	var cmd tea.Cmd
	m, cmd = updateModel(t, m, ChatResponseMsg{Text: resp})

	// Should detect decisions and trigger AgentDecisionsReadyMsg
	msg := processCmd(t, cmd)
	adm, ok := msg.(AgentDecisionsReadyMsg)
	if !ok {
		t.Fatalf("expected AgentDecisionsReadyMsg, got %T", msg)
	}
	if len(adm.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(adm.Decisions))
	}

	// Task should be set from the last user message
	if m.task != "build a todo app" {
		t.Errorf("task = %q, want %q", m.task, "build a todo app")
	}

	// We don't process TaskSubmittedMsg further because it triggers decomposition
	// with nil ccProvider. Just verify the message was produced correctly.
}

func TestDecomposingToPrivileges(t *testing.T) {
	m := setupAtPriorities(t, fakeDecisions())

	if m.view != ViewPriorities {
		t.Fatalf("view = %d, want ViewPriorities (%d)", m.view, ViewPriorities)
	}
	if len(m.tree.decisions) != 3 {
		t.Fatalf("tree.decisions = %d, want 3", len(m.tree.decisions))
	}
}

func TestPrioritiesToTree(t *testing.T) {
	// Confirm priorities: skip for Stack, paranoid for UI
	priorities := map[string]agent.CareLevel{
		"Stack": agent.CareLevelSkip,
		"UI":    agent.CareLevelParanoid,
	}
	m := setupAtTree(t, fakeDecisions(), priorities)

	if m.view != ViewConversation {
		t.Fatalf("view = %d, want ViewConversation (%d)", m.view, ViewConversation)
	}

	// Verify Stack decisions are auto-decided
	for _, d := range m.tree.decisions {
		if d.Category == "Stack" {
			if d.Answer == nil {
				t.Errorf("Stack decision %s should be auto-decided", d.ID)
			}
			if d.Source != "auto" {
				t.Errorf("Stack decision %s source = %q, want auto", d.ID, d.Source)
			}
		}
	}

	// Verify UI decisions are still pending
	for _, d := range m.tree.decisions {
		if d.Category == "UI" {
			if d.Answer != nil {
				t.Errorf("UI decision %s should be pending (paranoid), got answer %q", d.ID, *d.Answer)
			}
		}
	}
}

func TestAutoDecideSkipsParanoid(t *testing.T) {
	decs := []decision.Decision{
		{ID: "@SKI-0001", Category: "Infra", Question: "CDN?",
			Options: []decision.DecisionOption{{Key: "A", Label: "CloudFront"}}, Source: "user"},
		{ID: "@LOW-0001", Category: "Logging", Question: "Logger?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Winston"}}, Source: "user"},
		{ID: "@MED-0001", Category: "Data", Question: "ORM?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Prisma"}}, Source: "user"},
		{ID: "@HIG-0001", Category: "Auth", Question: "Provider?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Auth0"}}, Source: "user"},
		{ID: "@PAR-0001", Category: "Security", Question: "Encryption?",
			Options: []decision.DecisionOption{{Key: "A", Label: "AES-256"}}, Source: "user"},
	}

	priorities := map[string]agent.CareLevel{
		"Infra":    agent.CareLevelSkip,
		"Logging":  agent.CareLevelLow,
		"Data":     agent.CareLevelMedium,
		"Auth":     agent.CareLevelHigh,
		"Security": agent.CareLevelParanoid,
	}
	m := setupAtTree(t, decs, priorities)

	tests := []struct {
		id       string
		wantAuto bool
		desc     string
	}{
		{"@SKI-0001", true, "skip domain should be auto-decided"},
		{"@LOW-0001", true, "low domain should be auto-decided"},
		{"@MED-0001", false, "medium domain keeps first decision pending"},
		{"@HIG-0001", false, "high domain should stay pending"},
		{"@PAR-0001", false, "paranoid domain should stay pending"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, d := range m.tree.decisions {
				if d.ID == tt.id {
					if tt.wantAuto && d.Answer == nil {
						t.Errorf("%s: expected auto-decided, got pending", tt.id)
					}
					if !tt.wantAuto && d.Answer != nil {
						t.Errorf("%s: expected pending, got %q", tt.id, *d.Answer)
					}
					if tt.wantAuto && d.Answer != nil && d.Source != "auto" {
						t.Errorf("%s: source = %q, want auto", tt.id, d.Source)
					}
					return
				}
			}
			t.Errorf("decision %s not found", tt.id)
		})
	}
}

func TestReviseDecision(t *testing.T) {
	m := setupAtTreeNoExecutors(t, fakeDecisions())

	// Revise STA-0001
	m, _ = updateModel(t, m, ReviseDecisionMsg{ID: "@STA-0001", NewAnswer: "TypeScript"})

	var found bool
	for _, d := range m.tree.decisions {
		if d.ID == "@STA-0001" {
			found = true
			if d.Answer == nil || *d.Answer != "TypeScript" {
				t.Errorf("answer = %v, want TypeScript", d.Answer)
			}
			if d.Source != "user" {
				t.Errorf("source = %q, want user", d.Source)
			}
			if d.Delegated {
				t.Error("delegated should be false")
			}
		}
	}
	if !found {
		t.Fatal("@STA-0001 not found in tree decisions")
	}
}

func TestReviseTriggersExecutorsWhenAllAnswered(t *testing.T) {
	// Need at least one skip/medium decision so autoIDs is not nil, which
	// prevents the "nil = auto-decide all" behavior in Agent.AutoDecide.
	decs := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Language?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}},
			Source:  "user"},
		{ID: "@UIX-0001", Category: "UI", Question: "Framework?",
			Options: []decision.DecisionOption{{Key: "A", Label: "React"}},
			Source:  "user"},
		{ID: "@UIX-0002", Category: "UI", Question: "State management?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Redux"}},
			Source:  "user"},
	}

	priorities := map[string]agent.CareLevel{
		"Stack": agent.CareLevelSkip,
		"UI":    agent.CareLevelParanoid,
	}
	m := setupAtTree(t, decs, priorities)

	// Stack is auto-decided (skip), but UI decisions are pending (paranoid).
	// Since autoIDs has STA-0001, the Agent.AutoDecide call only auto-decides
	// Stack, leaving UI decisions pending.
	if m.executorsLaunched {
		t.Fatal("executors should not be launched yet (UI decisions pending)")
	}

	// Answer one of the pending decisions
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, ReviseDecisionMsg{ID: "@UIX-0001", NewAnswer: "React"})

	// Process CheckAllDecidedMsg
	msg := processCmd(t, cmd)
	if _, ok := msg.(CheckAllDecidedMsg); !ok {
		t.Fatalf("expected CheckAllDecidedMsg, got %T", msg)
	}
	m, _ = updateModel(t, m, msg)

	// Still one more pending (UIX-0002)
	if m.executorsLaunched {
		t.Fatal("executors should not be launched yet (UIX-0002 still pending)")
	}

	// Answer the last pending decision
	m, cmd = updateModel(t, m, ReviseDecisionMsg{ID: "@UIX-0002", NewAnswer: "Redux"})
	msg = processCmd(t, cmd)
	if _, ok := msg.(CheckAllDecidedMsg); !ok {
		t.Fatalf("expected CheckAllDecidedMsg, got %T", msg)
	}
	m, _ = updateModel(t, m, msg)

	if !m.executorsLaunched {
		t.Error("executorsLaunched should be true after all decisions answered")
	}
	if m.tree.overallStatus != "executing" {
		t.Errorf("overallStatus = %q, want executing", m.tree.overallStatus)
	}
}

func TestReviseAfterExecutionRelaunches(t *testing.T) {
	decs := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Language?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}},
			Answer:  strPtr("Go"), Source: "auto"},
	}

	// Set up at tree without executors, then manually mark as launched
	m := setupAtTreeNoExecutors(t, decs)
	m.executorsLaunched = true
	m.tree.overallStatus = "executing"

	// Cancel context first so that when ReviseDecisionMsg triggers
	// LaunchExecutors, the goroutine exits immediately.
	m.cancel()

	// Now revise a decision after execution.
	m, _ = updateModel(t, m, ReviseDecisionMsg{ID: "@STA-0001", NewAnswer: "Rust"})

	// After revision with executors already launched, it should re-launch
	if !m.executorsLaunched {
		t.Error("executorsLaunched should be true (re-launched)")
	}
	if m.tree.overallStatus != "executing" {
		t.Errorf("overallStatus = %q, want executing", m.tree.overallStatus)
	}

	// Verify the decision was updated
	for _, d := range m.tree.decisions {
		if d.ID == "@STA-0001" {
			if d.Answer == nil || *d.Answer != "Rust" {
				t.Errorf("answer = %v, want Rust", d.Answer)
			}
			if d.Source != "user" {
				t.Errorf("source = %q, want user", d.Source)
			}
		}
	}
}

func TestExecutorDecisionsMerge(t *testing.T) {
	m := setupAtTreeNoExecutors(t, fakeDecisions())

	// Add a new decision directly to tree and verify merge doesn't duplicate
	newDec := decision.Decision{
		ID: "@DAT-0001", Category: "Data", Question: "Database choice?",
		Answer: strPtr("PostgreSQL"), Source: "agent",
	}
	m.tree.decisions = append(m.tree.decisions, newDec)

	m, _ = updateModel(t, m, ExecutorDecisionStoredMsg{
		ExecutorID: "domain-Implementation",
		Decisions:  []decision.Decision{newDec},
	})

	// Should not have duplicated
	countWithID := 0
	for _, d := range m.tree.decisions {
		if d.ID == "@DAT-0001" {
			countWithID++
		}
	}
	if countWithID > 1 {
		t.Errorf("@DAT-0001 appears %d times, should be 1 (no duplication)", countWithID)
	}

	// Original decisions should still be there
	for _, orig := range fakeDecisions() {
		found := false
		for _, d := range m.tree.decisions {
			if d.ID == orig.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("original decision %s was lost during merge", orig.ID)
		}
	}
}

func TestExecutorDecisionsDontOverwriteUserChanges(t *testing.T) {
	m := setupAtTreeNoExecutors(t, fakeDecisions())

	// User revises a decision
	m, _ = updateModel(t, m, ReviseDecisionMsg{ID: "@STA-0001", NewAnswer: "Rust"})

	// Verify user change
	for _, d := range m.tree.decisions {
		if d.ID == "@STA-0001" {
			if d.Answer == nil || *d.Answer != "Rust" {
				t.Fatalf("user revision not applied")
			}
			if d.Source != "user" {
				t.Fatalf("source should be user")
			}
		}
	}

	// Now an executor decision stored msg comes in (shouldn't overwrite)
	m, _ = updateModel(t, m, ExecutorDecisionStoredMsg{
		ExecutorID: "domain-Implementation",
		Decisions:  []decision.Decision{},
	})

	// User's revision should be preserved
	for _, d := range m.tree.decisions {
		if d.ID == "@STA-0001" {
			if d.Answer == nil || *d.Answer != "Rust" {
				t.Errorf("user revision was overwritten, got %v", d.Answer)
			}
			if d.Source != "user" {
				t.Errorf("source changed from user to %q", d.Source)
			}
		}
	}
}

func TestSuggestReplacesOptions(t *testing.T) {
	m := setupAtTreeNoExecutors(t, fakeDecisions())

	// Answer a decision first
	m, _ = updateModel(t, m, ReviseDecisionMsg{ID: "@STA-0001", NewAnswer: "TypeScript"})

	// Now send SuggestResponseMsg with new options
	newOpts := []decision.DecisionOption{
		{Key: "A", Label: "Rust"},
		{Key: "B", Label: "Elixir"},
		{Key: "C", Label: "Zig"},
		{Key: "D", Label: "Haskell"},
	}
	m, _ = updateModel(t, m, SuggestResponseMsg{ID: "@STA-0001", Options: newOpts})

	for _, d := range m.tree.decisions {
		if d.ID == "@STA-0001" {
			if len(d.Options) != 4 {
				t.Errorf("options count = %d, want 4", len(d.Options))
			}
			if d.Options[0].Label != "Rust" {
				t.Errorf("first option = %q, want Rust", d.Options[0].Label)
			}
			if d.Answer != nil {
				t.Errorf("answer should be nil after suggest, got %q", *d.Answer)
			}
			return
		}
	}
	t.Fatal("@STA-0001 not found")
}

func TestWhyResponse(t *testing.T) {
	m := NewModel("", nil, t.TempDir())
	m.view = ViewTree
	m.tree.decisions = fakeDecisions()

	m, _ = updateModel(t, m, WhyResponseMsg{Text: "TypeScript has better tooling."})

	if m.tree.whyText != "TypeScript has better tooling." {
		t.Errorf("whyText = %q, want %q", m.tree.whyText, "TypeScript has better tooling.")
	}
}

func TestAllExecutorsDone(t *testing.T) {
	m := NewModel("", nil, t.TempDir())
	m.view = ViewTree
	m.tree.decisions = fakeDecisions()

	m, _ = updateModel(t, m, AllExecutorsDoneMsg{})

	if m.tree.overallStatus != "done" {
		t.Errorf("overallStatus = %q, want done", m.tree.overallStatus)
	}
}

func TestDoubleCtrlCQuits(t *testing.T) {
	m := NewModel("", nil, t.TempDir())
	m.view = ViewTree
	m.tree.decisions = fakeDecisions()

	// First Ctrl+C
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, keyCtrlC())
	if cmd != nil {
		msg := processCmd(t, cmd)
		if msg != nil {
			t.Errorf("first ctrl+c should not produce quit, got %T", msg)
		}
	}

	// Second Ctrl+C within 1.5s
	m, cmd = updateModel(t, m, keyCtrlC())
	if cmd == nil {
		t.Fatal("second ctrl+c should produce tea.Quit cmd")
	}

	msg := processCmd(t, cmd)
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestDoubleCtrlCSlow(t *testing.T) {
	m := NewModel("", nil, t.TempDir())
	m.view = ViewTree
	m.tree.decisions = fakeDecisions()

	// First Ctrl+C
	m, _ = updateModel(t, m, keyCtrlC())

	// Simulate time passing beyond 1.5s
	m.lastCtrlC = time.Now().Add(-2 * time.Second)

	// Second Ctrl+C after delay - should NOT quit (new "first" press)
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, keyCtrlC())

	if cmd != nil {
		msg := processCmd(t, cmd)
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("ctrl+c after long delay should not quit")
		}
	}
}

func TestMiscCategoryFiltered(t *testing.T) {
	decs := fakeDecisions()
	miscAnswer := "(catch-all category)"
	decs = append(decs, decision.Decision{
		ID:       "MIS-0001",
		Category: "Misc",
		Question: "Uncategorized implementation decisions",
		Answer:   &miscAnswer,
		Source:   "auto",
	})

	m := NewModel("", nil, t.TempDir())
	m.view = ViewTree
	m.tree.decisions = decs

	items := m.tree.decisionItems()
	for _, item := range items {
		if item.Question == "Uncategorized implementation decisions" && item.Source == "auto" {
			t.Error("Misc placeholder should be filtered out")
		}
	}

	// decisionCount should not include the misc placeholder
	expectedCount := len(decs) - 1
	if m.tree.decisionCount() != expectedCount {
		t.Errorf("decisionCount() = %d, want %d", m.tree.decisionCount(), expectedCount)
	}
}

func TestWindowSizePropagates(t *testing.T) {
	m := NewModel("", nil, t.TempDir())
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 200, Height: 50})

	if m.width != 200 || m.height != 50 {
		t.Errorf("root: %dx%d, want 200x50", m.width, m.height)
	}
	if m.priorities.width != 200 || m.priorities.height != 50 {
		t.Errorf("priorities: %dx%d, want 200x50", m.priorities.width, m.priorities.height)
	}
	if m.tree.width != 200 || m.tree.height != 50 {
		t.Errorf("tree: %dx%d, want 200x50", m.tree.width, m.tree.height)
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	views := []struct {
		name string
		view View
	}{
		{"Conversation", ViewConversation},
		{"Priorities", ViewPriorities},
		{"Tree", ViewTree},
	}

	for _, vv := range views {
		t.Run(vv.name, func(t *testing.T) {
			m := NewModel("", nil, t.TempDir())
			m.view = vv.view
			m.width = 80
			m.height = 24
			m.tree.decisions = fakeDecisions()
			m.priorities = NewPrioritiesModel(fakeDecisions())

			output := m.View()
			_ = output // no panic = pass
		})
	}
}

func TestTaskSubmittedFromCLI(t *testing.T) {
	m := NewModel("build something", nil, t.TempDir())
	if m.view != ViewConversation {
		t.Errorf("view = %d, want ViewConversation when task provided", m.view)
	}
	if m.task != "build something" {
		t.Errorf("task = %q, want %q", m.task, "build something")
	}
	// Should have the task in chat log
	foundTask := false
	for _, entry := range m.tree.chatLog {
		if entry.Type == "user" && entry.Text == "build something" {
			foundTask = true
		}
	}
	if !foundTask {
		t.Error("chat log should contain the task as a user message")
	}
}

func TestCheckAllDecidedWithPending(t *testing.T) {
	// Need at least one non-paranoid decision so autoIDs is not nil.
	decs := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Lang?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}}, Source: "user"},
		{ID: "@UIX-0001", Category: "UI", Question: "CSS?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Tailwind"}}, Source: "user"},
	}

	priorities := map[string]agent.CareLevel{
		"Stack": agent.CareLevelSkip,
		"UI":    agent.CareLevelParanoid,
	}
	m := setupAtTree(t, decs, priorities)

	// Stack is auto-decided, UI is still pending
	if m.executorsLaunched {
		t.Fatal("executors should not have launched (UI pending)")
	}

	// CheckAllDecided when there are still pending decisions
	m, _ = updateModel(t, m, CheckAllDecidedMsg{})

	// Should NOT launch executors
	if m.executorsLaunched {
		t.Error("executors should not launch with pending decisions")
	}
}

func TestParseSuggestedOptions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			"JSON array",
			`[{"key": "A", "label": "Option 1"}, {"key": "B", "label": "Option 2"}]`,
			2,
		},
		{
			"JSON in markdown",
			"Here are options:\n```json\n" + `[{"key": "A", "label": "Foo"}, {"key": "B", "label": "Bar"}]` + "\n```",
			2,
		},
		{
			"Numbered list fallback",
			"1. First option here\n2. Second option here\n3. Third option here\n4. Fourth option here",
			4,
		},
		{
			"Empty input",
			"No options here.",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseSuggestedOptions(tt.input)
			if len(opts) != tt.want {
				t.Errorf("got %d options, want %d", len(opts), tt.want)
			}
		})
	}
}

func TestStripJSONBlocks(t *testing.T) {
	input := "Some reasoning\n```defer-decisions\n[{\"key\": \"A\"}]\n```\nMore reasoning"
	result := stripJSONBlocks(input)

	if result == "" {
		t.Error("stripJSONBlocks removed everything")
	}
	if len(result) > len(input) {
		t.Error("result should be shorter than input")
	}
}
