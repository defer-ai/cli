package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// View represents which screen is active.
type View int

const (
	ViewWelcome View = iota
	ViewDecomposing
	ViewPriorities
	ViewTree
)

// Model is the root Bubbletea model.
type Model struct {
	view   View
	task   string
	width  int
	height int

	// Sub-models
	welcome    WelcomeModel
	priorities PrioritiesModel
	tree       TreeModel

	// Backend
	manager    *agent.Manager
	ccProvider *api.ClaudeCodeProvider
	cwd        string
	eventChan  chan tea.Msg
	ctx        context.Context
	cancel     context.CancelFunc

	// State
	mascotTick      int
	reasoningLines  []string
	executorsLaunched bool
	lastCtrlC       time.Time

	// Priorities (persisted)
	domainPriorities map[string]agent.CareLevel
}

// NewModel creates the root model.
func NewModel(task string, ccProvider *api.ClaudeCodeProvider, cwd string) Model {
	ctx, cancel := context.WithCancel(context.Background())
	m := Model{
		task:             task,
		ccProvider:       ccProvider,
		cwd:              cwd,
		welcome:          NewWelcomeModel(),
		tree:             NewTreeModel(),
		eventChan:        make(chan tea.Msg, 100),
		ctx:              ctx,
		cancel:           cancel,
		domainPriorities: make(map[string]agent.CareLevel),
	}

	// Always create manager upfront (Init can't modify the model)
	m.manager = agent.NewManager(ccProvider, cwd)

	if task != "" {
		m.view = ViewDecomposing
	} else {
		m.view = ViewWelcome
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{DoTick()}

	if m.task != "" {
		// Manager already created in NewModel, just start decomposition
		if m.manager != nil {
			ch := m.eventChan
			m.manager.StartDecomposition(m.ctx, m.task, func(ev agent.Event) {
				ch <- BridgeAgentEvent(ev)
			})
			cmds = append(cmds, ListenForEvents(m.eventChan))
		}
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.welcome.width = msg.Width
		m.welcome.height = msg.Height
		m.priorities.width = msg.Width
		m.priorities.height = msg.Height
		m.tree.width = msg.Width
		m.tree.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Double Ctrl+C to quit
		if msg.String() == "ctrl+c" {
			now := time.Now()
			if now.Sub(m.lastCtrlC) < 1500*time.Millisecond {
				m.cancel()
				return m, tea.Quit
			}
			m.lastCtrlC = now
			m.reasoningLines = append(m.reasoningLines, "Press Ctrl+C again to exit.")
			if len(m.reasoningLines) > 20 {
				m.reasoningLines = m.reasoningLines[len(m.reasoningLines)-20:]
			}
			return m, nil
		}

	case TickMsg:
		m.mascotTick++
		m.welcome.mascotTick = m.mascotTick
		m.tree.mascotTick = m.mascotTick
		return m, DoTick()

	case TaskSubmittedMsg:
		m.task = msg.Task
		m.view = ViewDecomposing
		m.manager = agent.NewManager(m.ccProvider, m.cwd)
		ch := m.eventChan
		m.manager.StartDecomposition(m.ctx, m.task, func(ev agent.Event) {
			ch <- BridgeAgentEvent(ev)
		})
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AgentStateChangedMsg:
		if m.manager != nil && m.manager.Agent() != nil {
			st := m.manager.Agent().State()
			if st.Output != "" {
				cleaned := stripJSONBlocks(st.Output)
				if cleaned != "" {
					lines := strings.Split(cleaned, "\n")
					if len(lines) > 6 {
						lines = lines[len(lines)-6:]
					}
					m.reasoningLines = lines
				}
			}
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AgentDecisionsReadyMsg:
		m.tree.decisions = msg.Decisions
		// Go straight to priorities (no swarm step)
		m.view = ViewPriorities
		m.priorities = NewPrioritiesModel(msg.Decisions)
		m.priorities.width = m.width
		m.priorities.height = m.height
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case PrioritiesConfirmedMsg:
		m.domainPriorities = msg.Priorities
		m.view = ViewTree

		if m.manager != nil {
			m.manager.AutoDecide(msg.Priorities)
			m.tree.decisions = m.manager.Agent().Decisions()

			// Only launch executors if ALL visible decisions are answered
			allAnswered := true
			for _, d := range m.tree.decisionItems() {
				if d.IsPending() {
					allAnswered = false
					break
				}
			}
			if allAnswered {
				ch := m.eventChan
				m.manager.LaunchExecutors(m.ctx, m.task, m.tree.decisions, msg.Priorities, func(ev agent.Event) {
					ch <- BridgeAgentEvent(ev)
				})
				m.executorsLaunched = true
				m.tree.overallStatus = "executing"
			} else {
				m.tree.overallStatus = "thinking"
			}
		}

		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ExecutorStateChangedMsg:
		if m.manager != nil {
			var lines []string
			var feedLines []string
			for _, exec := range m.manager.Executors() {
				st := exec.State()
				if st.Status == agent.DomainExecuting || st.Status == agent.DomainPlanning || st.Status == agent.DomainVerifying {
					status := extractShortStatus(st.Output, st.Status.String())
					lines = append(lines, "["+st.Domain+"] "+status)
					// Feed: last few lines of raw output
					if st.Output != "" {
						outLines := strings.Split(st.Output, "\n")
						for _, ol := range outLines[max(0, len(outLines)-3):] {
							if strings.TrimSpace(ol) != "" {
								feedLines = append(feedLines, "["+st.Domain+"] "+strings.TrimSpace(ol))
							}
						}
					}
				}
			}
			if len(lines) > 0 {
				m.reasoningLines = lines
			}
			if len(feedLines) > 0 {
				m.tree.feedLines = append(m.tree.feedLines, feedLines...)
				if len(m.tree.feedLines) > 200 {
					m.tree.feedLines = m.tree.feedLines[len(m.tree.feedLines)-200:]
				}
			}
			m.tree.overallStatus = m.computeOverallStatus()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ExecutorDecisionStoredMsg:
		// Merge new decisions without overwriting user changes
		m.manager.SyncDecisions(m.tree.decisions) // push user changes first
		allDecs := m.manager.AllDecisions()
		// Add any new decisions not already in tree
		existing := make(map[string]bool)
		for _, d := range m.tree.decisions {
			existing[d.ID] = true
		}
		for _, d := range allDecs {
			if !existing[d.ID] {
				m.tree.decisions = append(m.tree.decisions, d)
				existing[d.ID] = true
			}
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AllExecutorsDoneMsg:
		m.tree.overallStatus = "done"
		m.reasoningLines = append(m.reasoningLines, "All domains complete.")
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case WhyResponseMsg:
		m.tree.whyText = msg.Text
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ReviseDecisionMsg:
		var changedDecision string
		for i := range m.tree.decisions {
			if m.tree.decisions[i].ID == msg.ID {
				answer := msg.NewAnswer
				m.tree.decisions[i].Answer = &answer
				m.tree.decisions[i].Delegated = false
				m.tree.decisions[i].Source = "user"
				changedDecision = m.tree.decisions[i].Question
				break
			}
		}
		if m.manager != nil {
			m.manager.SyncDecisions(m.tree.decisions)
		}
		store, _ := decision.LoadStore(m.cwd)
		if store != nil {
			store.Decisions = m.tree.decisions
			_ = decision.SaveStore(m.cwd, store)
		}

		// If executors already ran, re-execute with updated decisions
		if m.executorsLaunched && changedDecision != "" {
			m.executorsLaunched = false // allow re-launch
			m.tree.overallStatus = "executing"
			m.reasoningLines = append(m.reasoningLines, fmt.Sprintf("Decision changed: %s. Re-implementing...", trunc(changedDecision, 40)))
			// Re-launch executor with updated decisions
			ch := m.eventChan
			m.manager.LaunchExecutors(m.ctx, m.task, m.tree.decisions, m.domainPriorities, func(ev agent.Event) {
				ch <- BridgeAgentEvent(ev)
			})
			m.executorsLaunched = true
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}

		return m, func() tea.Msg { return CheckAllDecidedMsg{} }

	case CheckAllDecidedMsg:
		if !m.executorsLaunched && m.manager != nil {
			allAnswered := true
			for _, d := range m.tree.decisionItems() {
				if d.IsPending() {
					allAnswered = false
					break
				}
			}
			if allAnswered && len(m.tree.decisions) > 0 {
				ch := m.eventChan
				m.manager.LaunchExecutors(m.ctx, m.task, m.tree.decisions, m.domainPriorities, func(ev agent.Event) {
					ch <- BridgeAgentEvent(ev)
				})
				m.executorsLaunched = true
				m.tree.overallStatus = "executing"
				m.reasoningLines = append(m.reasoningLines, "All decisions answered. Launching executors...")
				cmds = append(cmds, ListenForEvents(m.eventChan))
				return m, tea.Batch(cmds...)
			}
		}
		return m, nil

	case SuggestResponseMsg:
		// Replace options on the target decision
		for i := range m.tree.decisions {
			if m.tree.decisions[i].ID == msg.ID {
				m.tree.decisions[i].Options = msg.Options
				m.tree.decisions[i].Answer = nil // reset answer so user can re-pick
				m.tree.decisions[i].Delegated = false
				break
			}
		}
		m.tree.whyText = ""
		m.tree.optCursor = 0
		return m, nil

	case TogglePermissionsMsg:
		// permissions bypass is implicit, no toggle needed
		return m, nil

	case WhyDecisionMsg:
		// Launch async completion via subprocess
		ch := m.eventChan
		if m.ccProvider != nil {
			go func() {
				events := make(chan api.Event, 100)
				go m.ccProvider.RunCompletion(m.ctx,
					"Explain tradeoffs concisely.",
					"Explain tradeoffs of choosing \""+msg.Label+"\" for decision "+msg.ID,
					events)
				var text string
				for ev := range events {
					if ev.Type == api.EventTextDelta {
						text += ev.Text
					}
					if ev.Type == api.EventDone || ev.Type == api.EventError {
						break
					}
				}
				if text == "" {
					text = "No response."
				}
				ch <- WhyResponseMsg{Text: text}
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AskDecisionMsg:
		ch := m.eventChan
		if m.ccProvider != nil {
			go func() {
				events := make(chan api.Event, 100)
				go m.ccProvider.RunCompletion(m.ctx,
					"Answer concisely.",
					"Question about decision "+msg.ID+": "+msg.Question,
					events)
				var text string
				for ev := range events {
					if ev.Type == api.EventTextDelta {
						text += ev.Text
					}
					if ev.Type == api.EventDone || ev.Type == api.EventError {
						break
					}
				}
				if text == "" {
					text = "No response."
				}
				ch <- WhyResponseMsg{Text: text}
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case SuggestDecisionMsg:
		ch := m.eventChan
		var question string
		var currentOpts []string
		for _, d := range m.tree.decisions {
			if d.ID == msg.ID {
				question = d.Question
				for _, o := range d.Options {
					currentOpts = append(currentOpts, o.Label)
				}
				break
			}
		}
		prompt := fmt.Sprintf(`For the decision "%s", suggest exactly 4 COMPLETELY DIFFERENT alternatives.

Current options (do NOT repeat these): %s

Output ONLY a JSON array with 4 new, creative alternatives:
[{"key": "A", "label": "new option 1"}, {"key": "B", "label": "new option 2"}, {"key": "C", "label": "new option 3"}, {"key": "D", "label": "new option 4"}]`, question, strings.Join(currentOpts, ", "))

		suggestID := msg.ID
		doSuggest := func(text string) {
			opts := parseSuggestedOptions(text)
			if len(opts) > 0 {
				ch <- SuggestResponseMsg{ID: suggestID, Options: opts}
			} else {
				ch <- WhyResponseMsg{Text: text}
			}
		}

		if m.ccProvider != nil {
			go func() {
				events := make(chan api.Event, 100)
				go m.ccProvider.RunCompletion(m.ctx,
					"You output JSON arrays of options. Nothing else.",
					prompt,
					events)
				var text string
				for ev := range events {
					if ev.Type == api.EventTextDelta {
						text += ev.Text
					}
					if ev.Type == api.EventDone || ev.Type == api.EventError {
						break
					}
				}
				if text == "" {
					text = "No response."
				}
				doSuggest(text)
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)
	}

	// Delegate to active sub-model
	switch m.view {
	case ViewWelcome:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.welcome, cmd = m.welcome.Update(keyMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewPriorities:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.priorities, cmd = m.priorities.Update(keyMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewTree:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.tree, cmd = m.tree.Update(keyMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewWelcome:
		return m.welcome.View()

	case ViewDecomposing:
		// Show mascot + "Analyzing..."
		mascot := RenderMascot(MoodThinking, m.mascotTick)
		return mascot + "\n\n  " + AccentStyle.Render("Analyzing task...") + "\n"

	case ViewPriorities:
		return m.priorities.View()

	case ViewTree:
		m.tree.height = m.height
		m.tree.width = m.width
		return m.tree.View()
	}

	return ""
}

func (m Model) computeOverallStatus() string {
	if m.manager == nil {
		return "idle"
	}

	// Check decomposition agent
	if m.manager.Agent() != nil {
		agentSt := m.manager.Agent().State()
		if agentSt.Status == agent.StatusThinking {
			return "thinking"
		}
	}

	// Check executors
	execs := m.manager.Executors()
	if len(execs) == 0 {
		// Executors not launched yet -- check if we have pending decisions
		for _, d := range m.tree.decisionItems() {
			if d.IsPending() {
				return "thinking" // waiting for user input
			}
		}
		if m.executorsLaunched {
			return "executing" // just launched, waiting for first event
		}
		return "idle"
	}

	anyActive := false
	allDone := true
	for _, e := range execs {
		st := e.State()
		switch st.Status {
		case agent.DomainExecuting, agent.DomainPlanning, agent.DomainVerifying, agent.DomainPending:
			anyActive = true
			allDone = false
		case agent.DomainDone, agent.DomainError:
			// done or errored
		default:
			allDone = false
		}
	}

	if anyActive {
		return "executing"
	}
	if allDone {
		return "done"
	}
	return "executing" // default to executing if executors exist but not all done
}

func extractShortStatus(output, fallback string) string {
	if output == "" {
		return fallback + "..."
	}
	lower := strings.ToLower(output)
	// Check last 500 chars
	if len(lower) > 500 {
		lower = lower[len(lower)-500:]
	}
	for _, pair := range [][2]string{
		{"scaffold", "Scaffolding..."},
		{"install", "Installing..."},
		{"creat", "Creating..."},
		{"config", "Configuring..."},
		{"writ", "Writing..."},
		{"test", "Testing..."},
		{"build", "Building..."},
		{"migrat", "Migrating..."},
		{"generat", "Generating..."},
		{"setup", "Setting up..."},
		{"implement", "Implementing..."},
		{"fix", "Fixing..."},
		{"add", "Adding..."},
		{"updat", "Updating..."},
		{"verif", "Verifying..."},
	} {
		if strings.Contains(lower, pair[0]) {
			return pair[1]
		}
	}
	return "Working..."
}

var jsonBlockRe = regexp.MustCompile("(?s)```(?:defer-decisions|defer-choices|defer-status|json)\\s*\\n.*?\\n```")

func stripJSONBlocks(text string) string {
	cleaned := jsonBlockRe.ReplaceAllString(text, "")
	// Also strip raw JSON arrays/objects that look like decision blocks
	lines := strings.Split(cleaned, "\n")
	var result []string
	inJSON := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[" || trimmed == "{" || strings.HasPrefix(trimmed, `{"key"`) || strings.HasPrefix(trimmed, `{"category"`) {
			inJSON = true
		}
		if inJSON {
			if trimmed == "]" || trimmed == "}" || trimmed == "]," || trimmed == "}," {
				if trimmed == "]" || trimmed == "}" {
					inJSON = false
				}
				continue
			}
			continue
		}
		if trimmed != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func parseSuggestedOptions(text string) []decision.DecisionOption {
	// Try to find a JSON array in the text
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		var raw []struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err == nil {
			var opts []decision.DecisionOption
			for _, r := range raw {
				if r.Label != "" {
					key := r.Key
					if key == "" {
						key = string(rune('A' + len(opts)))
					}
					opts = append(opts, decision.DecisionOption{Key: key, Label: r.Label})
				}
			}
			if len(opts) > 0 {
				return opts
			}
		}
	}

	// Fallback: parse numbered/bulleted lines as options
	var opts []decision.DecisionOption
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		// Match "1. Option" or "- Option" or "A) Option" or "* Option"
		for _, prefix := range []string{"1.", "2.", "3.", "4.", "5.", "A)", "B)", "C)", "D)", "E)", "- ", "* "} {
			if strings.HasPrefix(line, prefix) {
				label := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				label = strings.TrimLeft(label, " *_")
				if label != "" && len(label) > 3 {
					key := string(rune('A' + len(opts)))
					opts = append(opts, decision.DecisionOption{Key: key, Label: label})
					break
				}
			}
		}
		if len(opts) >= 4 {
			break
		}
	}
	return opts
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure fmt is used
var _ = fmt.Sprintf
