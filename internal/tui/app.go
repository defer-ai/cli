package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/update"
)

// View represents which screen is active.
type View int

const (
	ViewMain       View = iota // side-by-side tree + chat (primary view)
	ViewPriorities             // care level picker (overlay, returns to main)
)

// Backward compatibility aliases.
const (
	ViewChat = ViewMain
	ViewTree = ViewMain
)

// Model is the root Bubbletea model.
// ModelOpts configures the TUI model.
type ModelOpts struct {
	ShowMascot bool
	MascotSize int // display pixels per eye (0 = use default)
	Version    string
	ModelName  string
}

type Model struct {
	view   View
	task   string
	width  int
	height int

	// Header
	showMascot bool
	mascotSize int // display pixels per eye
	version    string
	modelName  string

	// Sub-models
	priorities PrioritiesModel
	tree       TreeModel

	// Backend
	manager  *agent.Manager
	provider api.Provider
	cwd      string
	eventChan chan tea.Msg
	ctx       context.Context
	cancel    context.CancelFunc

	// State
	mascotTick        int
	notifications     *NotificationManager
	executorsLaunched bool
	quitting          bool // set on quit, prevents new goroutine spawns
	updateAvailable   string // empty = up to date, else latest version like "2.0.3"

	// Permission overlay
	pendingPermission *PermissionRequestMsg
	pendingRevise     *ReviseDecisionMsg // queued from chat @ID commands

	// Priorities (persisted)
	domainPriorities map[string]agent.CareLevel
}

// NewModel creates the root model.
func NewModel(task string, provider api.Provider, cwd string, opts ...ModelOpts) Model {
	ctx, cancel := context.WithCancel(context.Background())
	tree := NewTreeModel()
	tree.domainStatuses = make(map[string]string)
	// Chat panel focused by default (right panel in side-by-side)
	tree.focusPanel = FocusChat
	tree.chatFocused = true
	tree.chatInput.Focus()

	// Apply opts
	var o ModelOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	m := Model{
		task:             task,
		provider:         provider,
		cwd:              cwd,
		showMascot:       o.ShowMascot,
		mascotSize:       o.MascotSize,
		version:          o.Version,
		modelName:        o.ModelName,
		tree:             tree,
		eventChan:        make(chan tea.Msg, 100),
		ctx:              ctx,
		cancel:           cancel,
		domainPriorities: make(map[string]agent.CareLevel),
		notifications:    NewNotificationManager(),
		view:             ViewMain,
	}

	m.manager = agent.NewManager(provider, cwd)

	// Load Claude session ID from .defer/ if it exists
	if sessionID := loadSessionID(cwd); sessionID != "" {
		if cc, ok := provider.(*api.ClaudeCodeProvider); ok {
			cc.SetSessionID(sessionID)
		}
	}

	// Check for existing session to resume (only if no new task given)
	if task == "" {
		store, _ := decision.LoadStore(cwd)
		priorities := loadPriorities(cwd)

		if store != nil && len(store.Decisions) > 0 {
			// Resume existing session — never re-ask priorities, never overwrite decisions
			m.tree.decisions = store.Decisions
			m.domainPriorities = priorities
			m.task = store.Task

			// Count state
			answeredCount := 0
			pendingCount := 0
			for _, d := range store.Decisions {
				if d.Answer != nil {
					answeredCount++
				} else {
					pendingCount++
				}
			}

			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
				Type: "system",
				Text: fmt.Sprintf("Resumed session: %s (%d decisions, %d pending)", store.Task, len(store.Decisions), pendingCount),
			})

			m.tree.pendingCount = pendingCount
			if pendingCount > 0 {
				m.tree.overallStatus = "waiting"
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
					Type: "action",
					Text: fmt.Sprintf("%d decisions need your input. Tab to review.", pendingCount),
				})
			} else {
				m.tree.overallStatus = "done"
			}
			// Never show priorities on resume — decisions are already loaded
		}
	} else {
		// Task given via CLI arg — add as first message and start decomposition
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "user", Text: task})
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Analyzing project and identifying decisions..."})
	}

	return m
}



func loadSessionID(cwd string) string {
	data, err := os.ReadFile(filepath.Join(cwd, ".defer", "session_id"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveSessionID(cwd string, id string) {
	if id == "" {
		return
	}
	dir := filepath.Join(cwd, ".defer")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "session_id"), []byte(id), 0o644)
}

func loadPriorities(cwd string) map[string]agent.CareLevel {
	data, err := os.ReadFile(filepath.Join(cwd, ".defer", "priorities.json"))
	if err != nil {
		return nil
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	result := make(map[string]agent.CareLevel)
	for k, v := range raw {
		if k == "_task" {
			continue
		}
		result[k] = agent.CareLevel(v)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func savePriorities(cwd string, priorities map[string]agent.CareLevel, task string) {
	dir := filepath.Join(cwd, ".defer")
	os.MkdirAll(dir, 0o755)
	raw := make(map[string]string)
	raw["_task"] = task
	for k, v := range priorities {
		raw[k] = string(v)
	}
	data, _ := json.MarshalIndent(raw, "", "  ")
	os.WriteFile(filepath.Join(dir, "priorities.json"), data, 0o644)
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{DoTick()}

	// Non-blocking update check
	if m.version != "" && m.version != "dev" {
		version := m.version
		cmds = append(cmds, func() tea.Msg {
			latest, _, err := update.CheckForUpdate(version)
			if err != nil || latest == "" {
				return nil
			}
			return UpdateAvailableMsg{Version: latest}
		})
	}

	if m.task != "" {
		// Manager already created in NewModel, just start decomposition
		if m.manager != nil {
			ch := m.eventChan
			ctx := m.ctx
			m.manager.StartDecomposition(ctx, m.task, func(ev agent.Event) {
				safeSend(ctx, ch, BridgeAgentEvent(ev))
			})
			cmds = append(cmds, ListenForEvents(m.eventChan))
		}
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Drain queued revision from chat @ID commands
	if m.pendingRevise != nil {
		rev := *m.pendingRevise
		m.pendingRevise = nil
		return m.Update(rev)
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.priorities.width = msg.Width
		m.priorities.height = msg.Height
		m.tree.width = msg.Width
		m.tree.height = msg.Height
		return m, nil

	case UpdateAvailableMsg:
		m.updateAvailable = msg.Version
		return m, nil

	case tea.KeyMsg:
		// Ctrl+Q to quit
		if msg.String() == "ctrl+q" {
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		}
		// Ctrl+C shows a warning (avoid duplicates)
		if msg.String() == "ctrl+c" {
			// Only add if last chat entry isn't already this message
			alreadyShown := len(m.tree.chatLog) > 0 && m.tree.chatLog[len(m.tree.chatLog)-1].Text == "Press ctrl+q to quit."
			if !alreadyShown {
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Press ctrl+q to quit."})
			}
			m.notifications.Push("Press ctrl+q to quit.", NotifyMedium, 3*time.Second)
			return m, nil
		}

		// Intercept keys when a permission overlay is active
		if m.pendingPermission != nil {
			switch msg.String() {
			case "y", "enter":
				m.pendingPermission.ResponseCh <- api.PermissionResponse{Allow: true}
				m.pendingPermission = nil
			case "n", "esc":
				m.pendingPermission.ResponseCh <- api.PermissionResponse{Allow: false, Message: "User denied"}
				m.pendingPermission = nil
			}
			// Swallow all key events while permission overlay is shown
			return m, nil
		}

	case PermissionRequestMsg:
		m.pendingPermission = &msg
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case TickMsg:
		m.mascotTick++
		m.tree.mascotTick = m.mascotTick
		m.notifications.Tick()
		return m, DoTick()

	case TaskSubmittedMsg:
		// First message in conversation becomes the task — start decomposition
		m.task = msg.Task
		m.view = ViewMain
		m.manager = agent.NewManager(m.provider, m.cwd)
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Analyzing project and identifying decisions..."})
		ch := m.eventChan
		ctx := m.ctx
		m.manager.StartDecomposition(ctx, m.task, func(ev agent.Event) {
			safeSend(ctx, ch, BridgeAgentEvent(ev))
		})
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AgentStateChangedMsg:
		// Decomposition progress — show in notification bar only, not conversation
		if m.manager != nil && m.manager.Agent() != nil {
			st := m.manager.Agent().State()
			if st.Output != "" {
				cleaned := stripJSONBlocks(st.Output)
				if cleaned != "" {
					lines := strings.Split(cleaned, "\n")
					for i := len(lines) - 1; i >= 0; i-- {
						line := strings.TrimSpace(lines[i])
						if line != "" {
							m.notifications.Push(line, NotifyLow, 3*time.Second)
							break
						}
					}
				}
			}
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AgentDecisionsReadyMsg:
		if len(msg.Decisions) == 0 {
			// Decomposition failed to produce decisions — notify user
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
				Type: "system",
				Text: "Could not identify decisions. Try describing your project in more detail.",
			})
			m.tree.chatThinking = false
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}

		m.tree.decisions = msg.Decisions
		// Persist decisions immediately so resume works
		store, _ := decision.LoadStore(m.cwd)
		if store == nil {
			store, _ = decision.CreateStore(m.cwd, m.task)
		}
		if store != nil {
			store.Decisions = msg.Decisions
			store.Task = m.task
			_ = decision.SaveStore(m.cwd, store)
		}
		// Show summary in conversation
		categories := make(map[string]int)
		for _, d := range msg.Decisions {
			categories[d.Category]++
		}
		var summary strings.Builder
		summary.WriteString(fmt.Sprintf("Found **%d decisions** across %d domains:\n", len(msg.Decisions), len(categories)))
		for cat, count := range categories {
			summary.WriteString(fmt.Sprintf("  - **%s**: %d decisions\n", cat, count))
		}
		summary.WriteString("\nSet care level per domain in the panel below.")
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: summary.String()})
		m.tree.overallStatus = "waiting"

		// Show priorities inline in the resolver (not full-screen)
		cats := []string{}
		seen := map[string]bool{}
		for _, d := range msg.Decisions {
			if !seen[d.Category] {
				cats = append(cats, d.Category)
				seen[d.Category] = true
			}
		}
		m.tree.showingPriorities = true
		m.tree.priorityCategories = cats
		m.tree.priorityLevels = make(map[string]agent.CareLevel)
		for _, c := range cats {
			m.tree.priorityLevels[c] = agent.CareLevelAuto
		}
		m.tree.priorityCursor = 0
		m.tree.focusPanel = FocusChat // focus right panel for interaction
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case PrioritiesConfirmedMsg:
		m.domainPriorities = msg.Priorities
		savePriorities(m.cwd, msg.Priorities, m.task)
		m.view = ViewMain
		m.tree.mode = tmTree
		m.tree.focusPanel = FocusChat
		m.tree.chatFocused = true
		m.tree.chatInput.Focus()

		if m.manager != nil && m.manager.Agent() != nil {
			m.manager.AutoDecide(msg.Priorities)
			m.tree.decisions = m.manager.Agent().Decisions()
		} else {
			// Resume/scan path: auto-decide on tree decisions directly
			today := time.Now().Format("2006-01-02")
			priMap := make(map[string]agent.CareLevel)
			for k, v := range msg.Priorities {
				priMap[strings.ToLower(strings.TrimSpace(k))] = v
			}
			for i := range m.tree.decisions {
				d := &m.tree.decisions[i]
				if d.Answer != nil {
					continue
				}
				level := priMap[strings.ToLower(strings.TrimSpace(d.Category))]
				if level != agent.CareLevelReview {
					var answer string
					for _, opt := range d.Options {
						if !strings.Contains(strings.ToLower(opt.Label), "choose for me") {
							answer = opt.Label
							break
						}
					}
					if answer == "" && len(d.Options) > 0 {
						answer = d.Options[0].Label
					}
					if answer == "" {
						answer = "auto-decided"
					}
					d.SetAnswer(answer, "auto")
					d.Delegated = true
					d.Date = today
				}
			}
			// Save
			store, _ := decision.LoadStore(m.cwd)
			if store != nil {
				store.Decisions = m.tree.decisions
				_ = decision.SaveStore(m.cwd, store)
			}
		}

		// Only launch executors if there's a real task (not scan/resume without task)
		isScanOnly := m.task == "" || m.task == "(scanned project)" || m.task == "(scanning project)"

		allAnswered := true
		for _, d := range m.tree.decisionItems() {
			if d.IsPending() {
				allAnswered = false
				break
			}
		}

		m.tree.pendingCount = m.countPending()

		if allAnswered && !isScanOnly {
			ch := m.eventChan
			ctx := m.ctx
			m.manager.LaunchExecutors(ctx, m.task, m.tree.decisions, msg.Priorities, func(ev agent.Event) {
				safeSend(ctx, ch, BridgeAgentEvent(ev))
			})
			m.executorsLaunched = true
			m.tree.overallStatus = "executing"
			m.tree.chatThinking = true
			m.tree.chatThinkStart = time.Now()
		} else if !allAnswered {
			m.tree.overallStatus = "waiting"
		} else {
			m.tree.overallStatus = "done" // scan complete, all cataloged
		}

		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ExecutorStateChangedMsg:
		// Executor progress — update domain pills and notification bar only
		if m.manager != nil {
			for _, exec := range m.manager.Executors() {
				st := exec.State()
				m.tree.domainStatuses[st.Domain] = st.Status.String()
				if st.Status == agent.DomainExecuting || st.Status == agent.DomainPlanning || st.Status == agent.DomainVerifying {
					status := extractShortStatus(st.Output, st.Status.String())
					m.notifications.Push(status, NotifyLow, 3*time.Second)
				}
			}
			m.tree.overallStatus = m.computeOverallStatus()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ToolActivityMsg:
		m.tree.activityLine = msg.Description

		// Skip internal tool lookups and non-interactive tools
		if strings.Contains(msg.Description, "Waiting for input") ||
			strings.Contains(msg.Description, "AskUserQuestion") ||
			strings.HasPrefix(msg.Description, "Looking up tools:") {
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}

		// Classify: topics (Agent, planning, high-level) vs subtools (Bash, Read, etc.)
		isTopic := isTopicTool(msg.Description)

		if isTopic {
			// New topic — add as a top-level entry
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "topic", Text: msg.Description})
		} else {
			// Subtool — try to attach to the last topic as a child
			attached := false
			for i := len(m.tree.chatLog) - 1; i >= 0; i-- {
				entry := &m.tree.chatLog[i]
				if entry.Type == "topic" {
					// Deduplicate: skip if last child has same text
					if len(entry.Children) > 0 && entry.Children[len(entry.Children)-1].Text == msg.Description {
						attached = true
						break
					}
					entry.Children = append(entry.Children, ChatEntry{Type: "subtool", Text: msg.Description})
					attached = true
					break
				}
				// Stop looking past non-tool entries (agent response, system, user, action)
				if entry.Type != "tool" && entry.Type != "subtool" {
					break
				}
			}
			if !attached {
				// No parent topic found — promote to a topic so subsequent
				// orphan tools nest under it instead of creating a flat wall.
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "topic", Text: msg.Description})
			}
		}

		if len(m.tree.chatLog) > 200 {
			m.tree.chatLog = m.tree.chatLog[len(m.tree.chatLog)-200:]
		}
		m.notifications.Push(msg.Description, NotifyLow, 3*time.Second)
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
		var newDecs []decision.Decision
		for _, d := range allDecs {
			if !existing[d.ID] {
				m.tree.decisions = append(m.tree.decisions, d)
				existing[d.ID] = true
				newDecs = append(newDecs, d)
			}
		}
		// Show new decisions in chat (both auto and pending)
		for _, d := range newDecs {
			if d.Answer != nil {
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
					Type: "system",
					Text: fmt.Sprintf("Decided: %s → %s", d.Question, *d.Answer),
				})
			}
		}
		// Notify about pending decisions
		newPending := m.countPending()
		if newPending > m.tree.pendingCount && newPending > 0 {
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
				Type: "action",
				Text: fmt.Sprintf("%d decisions need your input. Tab to review.", newPending),
			})
		}
		m.tree.pendingCount = newPending
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ExecWaitingMsg:
		// Executor paused — waiting for pending decisions
		m.tree.pendingCount = m.countPending()
		m.tree.overallStatus = "waiting"
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
			Type: "action",
			Text: fmt.Sprintf("Paused — %d decisions need your input. Tab to review, then come back.", m.tree.pendingCount),
		})
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ResearchRequestMsg:
		// Spawn the chat agent to research the topic
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
			Type: "system",
			Text: fmt.Sprintf("Researching: %s", msg.Query),
		})
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx
			responseCh := msg.ResponseCh
			go func() {
				resp := runSimpleChat(ctx, m.provider,
					"You are a research assistant. Investigate the question thoroughly using Read, Glob, Grep tools. Be concise but complete.",
					msg.Query)
				responseCh <- resp
				safeSend(ctx, ch, ChatResponseMsg{Text: "Research: " + resp})
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AllExecutorsDoneMsg:
		m.tree.overallStatus = "done"
		m.tree.chatThinking = false
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Implementation complete."})
		m.notifications.Push("Implementation complete.", NotifyHigh, 0)
		// Persist Claude session ID for future resume
		if cc, ok := m.provider.(*api.ClaudeCodeProvider); ok {
			saveSessionID(m.cwd, cc.SessionID())
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case WhyResponseMsg:
		m.tree.whyText = msg.Text
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case ReviseDecisionMsg:
		var changedDecision *decision.Decision
		for i := range m.tree.decisions {
			if m.tree.decisions[i].ID == msg.ID {
				m.tree.decisions[i].SetAnswer(msg.NewAnswer, "user")
				m.tree.decisions[i].Delegated = false
				changedDecision = &m.tree.decisions[i]
				break
			}
		}

		// Cascade: invalidate dependent decisions
		if changedDecision != nil {
			dependents := decision.FindTransitiveDependents(changedDecision.ID, m.tree.decisions)
			if len(dependents) > 0 {
				var invalidatedIDs []string
				for _, dep := range dependents {
					for i := range m.tree.decisions {
						if m.tree.decisions[i].ID == dep.ID {
							decision.InvalidateDependent(&m.tree.decisions[i])
							invalidatedIDs = append(invalidatedIDs, dep.ID)
							break
						}
					}
				}
				// Show cascade in conversation
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
					Type: "action",
					Text: fmt.Sprintf("Changed %s → %s", changedDecision.ID, msg.NewAnswer),
				})
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
					Type: "action",
					Text: fmt.Sprintf("Invalidated %d dependent decisions: %s", len(invalidatedIDs), strings.Join(invalidatedIDs, ", ")),
				})
			} else {
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
					Type: "action",
					Text: fmt.Sprintf("Changed %s → %s", changedDecision.ID, msg.NewAnswer),
				})
			}

			// Show thinking indicator while processing the change
			m.tree.chatThinking = true
			m.tree.chatThinkStart = time.Now()

			// For high-impact changes (>= 8), check for implicit invalidation
			// across the full decision set using the agent
			if changedDecision.Impact >= 8 && m.provider != nil && !m.quitting {
				ch := m.eventChan
				ctx := m.ctx
				allDecJSON := decisionSummaryForAgent(m.tree.decisions)
				go func() {
					prompt := fmt.Sprintf(`Decision %s ("%s") changed to "%s". This is a high-impact foundational decision (impact %d/10).

Review ALL these decisions and identify which ones are now INCOMPATIBLE or need to change due to this foundational shift:

%s

For each incompatible decision, output ONLY a JSON array of IDs that should be invalidated:
["STA-0001", "DAT-0002"]

If no decisions are incompatible, output: []`, changedDecision.ID, changedDecision.Question, msg.NewAnswer, changedDecision.Impact, allDecJSON)

					resp := runSimpleChat(ctx, m.provider, "You identify incompatible decisions. Output only a JSON array of decision IDs.", prompt)
					// Parse the response for IDs
					start := strings.Index(resp, "[")
					end := strings.LastIndex(resp, "]")
					if start >= 0 && end > start {
						var ids []string
						if err := json.Unmarshal([]byte(resp[start:end+1]), &ids); err == nil && len(ids) > 0 {
							safeSend(ctx, ch, ImplicitInvalidationMsg{IDs: ids, Reason: fmt.Sprintf("Incompatible with %s = %s", changedDecision.ID, msg.NewAnswer)})
						}
					}
				}()
				cmds = append(cmds, ListenForEvents(m.eventChan))
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

		// Update pending count
		m.tree.pendingCount = m.countPending()

		// If executor is waiting for decisions and all are now answered, signal continue
		if m.executorsLaunched && m.tree.pendingCount == 0 && m.tree.overallStatus == "waiting" {
			m.signalExecutorContinue()
			m.tree.overallStatus = "executing"
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "All decisions resolved. Continuing..."})
			return m, nil
		}

		// If executors already ran and a decision changed, re-execute
		if m.executorsLaunched && changedDecision != nil && m.tree.overallStatus != "waiting" {
			m.executorsLaunched = false
			m.tree.overallStatus = "executing"
			ch := m.eventChan
			ctx := m.ctx
			m.manager.LaunchExecutors(ctx, m.task, m.tree.decisions, m.domainPriorities, func(ev agent.Event) {
				safeSend(ctx, ch, BridgeAgentEvent(ev))
			})
			m.executorsLaunched = true
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}

		return m, func() tea.Msg { return CheckAllDecidedMsg{} }

	case CheckAllDecidedMsg:
		allAnswered := true
		for _, d := range m.tree.decisionItems() {
			if d.IsPending() {
				allAnswered = false
				break
			}
		}

		if allAnswered && m.executorsLaunched && m.tree.overallStatus == "waiting" {
			// Executor is waiting — signal it to continue
			m.signalExecutorContinue()
			m.tree.overallStatus = "executing"
			m.tree.pendingCount = 0
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "All decisions resolved. Continuing..."})
			return m, nil
		}

		if allAnswered && !m.executorsLaunched && m.manager != nil && len(m.tree.decisions) > 0 {
			ch := m.eventChan
			ctx := m.ctx
			m.manager.LaunchExecutors(ctx, m.task, m.tree.decisions, m.domainPriorities, func(ev agent.Event) {
				safeSend(ctx, ch, BridgeAgentEvent(ev))
			})
			m.executorsLaunched = true
			m.tree.overallStatus = "executing"
			m.tree.chatThinking = true
			m.tree.chatThinkStart = time.Now()
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "All decisions answered. Launching..."})
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case ImplicitInvalidationMsg:
		var invalidated []string
		for _, id := range msg.IDs {
			for i := range m.tree.decisions {
				if m.tree.decisions[i].ID == id && m.tree.decisions[i].Source != "invalidated" {
					decision.InvalidateDependent(&m.tree.decisions[i])
					invalidated = append(invalidated, id)
					break
				}
			}
		}
		if len(invalidated) > 0 {
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
				Type: "action",
				Text: fmt.Sprintf("Invalidated %d dependent decisions: %s (%s)", len(invalidated), strings.Join(invalidated, ", "), msg.Reason),
			})
			// Persist
			if store, _ := decision.LoadStore(m.cwd); store != nil {
				store.Decisions = m.tree.decisions
				_ = decision.SaveStore(m.cwd, store)
			}
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case SuggestResponseMsg:
		// Replace options on the target decision and reset to pending
		for i := range m.tree.decisions {
			if m.tree.decisions[i].ID == msg.ID {
				wasAnswered := m.tree.decisions[i].Answer != nil
				m.tree.decisions[i].Options = msg.Options
				m.tree.decisions[i].Answer = nil
				m.tree.decisions[i].Delegated = false
				if wasAnswered {
					m.tree.decisions[i].Source = "invalidated"
					m.tree.decisions[i].RevisionCount++
				}
				break
			}
		}
		m.tree.whyText = ""
		m.tree.optCursor = 0
		m.tree.pendingCount = m.countPending()

		// Persist
		if store, _ := decision.LoadStore(m.cwd); store != nil {
			store.Decisions = m.tree.decisions
			_ = decision.SaveStore(m.cwd, store)
		}
		return m, nil

	case TogglePermissionsMsg:
		return m, nil

	case StopAgentMsg:
		// Cancel all running agents and create a fresh context
		m.cancel()
		m.ctx, m.cancel = context.WithCancel(context.Background())
		m.tree.chatThinking = false
		m.tree.overallStatus = "ready"
		m.executorsLaunched = false
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Agent stopped."})
		return m, nil

	case ChatMessageMsg:
		text := msg.Text

		// Check for direct change commands: "STA-0001 change to Go with Gin"
		if changed := m.tryParseChangeCommand(text); changed {
			return m, nil
		}

		// Send to agent — system prompt adapts based on whether we have a task
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx

			var sysPrompt string
			if m.task == "" {
				// No task yet — conversational mode with inline decomposition
				sysPrompt = `You are defer, a zero-autonomy AI assistant.

When the user describes a project, feature, or task they want built:
1. FIRST, use Read, Glob, and Grep to scan the existing codebase (if any).
   Check for: package manager files, config files, framework choices, project structure.
2. THEN, start your response with: TASK: <one-line description>
3. THEN, output a ` + "```defer-decisions" + ` JSON block with ALL decisions:

` + "```defer-decisions" + `
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [
      {"key": "A", "label": "Go with Gin"},
      {"key": "B", "label": "Node.js with Express"},
      {"key": "C", "label": "Choose for me"}
    ],
    "answer": "A",
    "context": "Already using Go (detected from go.mod)",
    "features": ["api", "backend"],
    "impact": 9,
    "dependsOn": []
  }
]
` + "```" + `

Rules:
- For decisions ALREADY made in the codebase: include answer field with the key of the chosen option
- For NEW decisions: omit answer field, include "Choose for me" as last option
- Every uncertainty = a decision with options. NEVER ask questions as text.
- Order by impact (highest first). Tag with features.
- If the codebase is empty, all decisions are new.
- Be THOROUGH. Aim for 20-40 decisions. Cover: language, framework, file structure, data models, API design, auth, error handling, UI patterns, testing, deployment, config, naming conventions, dependencies.
- Think like a senior developer planning every detail before writing code.

If the user is just chatting or asking questions — respond naturally WITHOUT TASK: prefix and WITHOUT defer-decisions block.`
			} else {
				// Task active — decision management mode with full context
				var decContext strings.Builder
				pendingCount := 0
				answeredCount := 0
				invalidatedCount := 0
				for _, d := range m.tree.decisions {
					status := ""
					if d.Answer != nil {
						answeredCount++
						status = *d.Answer
						if d.Source == "invalidated" {
							invalidatedCount++
							status += " [INVALIDATED - needs new answer]"
						}
					} else {
						pendingCount++
						status = "(PENDING - user must decide)"
					}
					decContext.WriteString(fmt.Sprintf("%s [%s]: %s → %s\n", d.ID, d.Category, d.Question, status))
				}

				execStatus := "not started"
				if m.executorsLaunched {
					execStatus = m.tree.overallStatus
				}

				// Build recent conversation context (last 10 user/agent messages)
				var convContext strings.Builder
				msgCount := 0
				for i := len(m.tree.chatLog) - 1; i >= 0 && msgCount < 10; i-- {
					entry := m.tree.chatLog[i]
					if entry.Type == "user" {
						convContext.WriteString(fmt.Sprintf("User: %s\n", entry.Text))
						msgCount++
					} else if entry.Type == "agent" {
						text := entry.Text
						if len(text) > 200 {
							text = text[:200] + "..."
						}
						convContext.WriteString(fmt.Sprintf("You: %s\n", text))
						msgCount++
					}
				}

				sysPrompt = fmt.Sprintf(`You are the defer assistant managing project decisions.

Status: %d answered, %d pending, %d invalidated. Execution: %s.
Task: %s

CRITICAL — WHEN THE USER ASKS FOR A NEW FEATURE OR CHANGE:
You MUST decompose it into decisions. Start your response with:
TASK: <one-line description of the new feature>
Then output a ` + "```defer-decisions" + ` JSON block with decisions for the new feature.
NEVER implement directly. ALWAYS go through the decision process first.

WHEN THE USER WANTS TO CHANGE A DECISION (even without using @ID):
Output a REVISE line: REVISE: ID | new answer
Example: if user says "actually use PostgreSQL for the database", output:
REVISE: DAT-0001 | PostgreSQL
You can output multiple REVISE lines. Include a brief explanation too.

GENERAL RULES:
- You are part of an ongoing conversation. Read the recent history below.
- An executor agent is running in parallel, implementing based on confirmed decisions.
- If all decisions are answered and execution is running, the project IS being built.
- Only reference decisions by their CURRENT state shown below
- Be concise
- Do NOT write code. Only manage decisions.

Recent conversation:
%s
Current decisions:
%s`, answeredCount, pendingCount, invalidatedCount, execStatus, m.task, convContext.String(), decContext.String())
			}

			// Use read-only provider when no task is set (pre-decomposition)
			chatProvider := m.provider
			if m.task == "" {
				if cc, ok := m.provider.(*api.ClaudeCodeProvider); ok {
					restricted := api.NewClaudeCodeProviderWithCWD(cc.GetModel(), m.cwd)
					restricted.AllowedTools = []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch"}
					chatProvider = restricted
				}
			}
			go func() {
				resp := runStreamingChat(ctx, chatProvider, sysPrompt, text, ch)
				safeSend(ctx, ch, ChatResponseMsg{Text: resp})
			}()
			cmds = append(cmds, ListenForEvents(m.eventChan))
		}
		return m, tea.Batch(cmds...)

	case ChatResponseMsg:
		m.tree.chatThinking = false

		if m.task == "" {
			trimmed := strings.TrimSpace(msg.Text)

			// Extract task from TASK: prefix if present
			hasTaskPrefix := strings.HasPrefix(trimmed, "TASK:")
			if hasTaskPrefix {
				firstLine := strings.SplitN(trimmed, "\n", 2)[0]
				m.task = strings.TrimSpace(strings.TrimPrefix(firstLine, "TASK:"))
				if m.task == "" {
					m.task = "(from conversation)"
				}
			}

			// Check if response contains inline decisions
			decs := agent.ParseDecisionsFromText(msg.Text)

			if len(decs) > 0 {
				// Got decisions inline — use them directly (best case)
				if m.task == "" {
					// No TASK: prefix but got decisions — extract task from last user message
					for i := len(m.tree.chatLog) - 1; i >= 0; i-- {
						if m.tree.chatLog[i].Type == "user" {
							m.task = m.tree.chatLog[i].Text
							break
						}
					}
					if m.task == "" {
						m.task = "(from conversation)"
					}
				}
				// Show the non-decision, non-internal text
				cleaned := cleanAgentResponse(msg.Text, hasTaskPrefix)
				if strings.TrimSpace(cleaned) != "" {
					m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: cleaned})
				}
				return m, func() tea.Msg {
					return AgentDecisionsReadyMsg{Decisions: decs}
				}
			}

			if hasTaskPrefix {
				// Got TASK: but no decisions — fallback to separate decomposition
				cleaned := cleanAgentResponse(msg.Text, true)
				if strings.TrimSpace(cleaned) != "" {
					m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: cleaned})
				}
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Identifying decisions..."})
				m.tree.chatThinking = true
				m.tree.chatThinkStart = time.Now()

				m.manager = agent.NewManager(m.provider, m.cwd)
				ch := m.eventChan
				ctx := m.ctx
				m.manager.StartDecomposition(ctx, m.task, func(ev agent.Event) {
					safeSend(ctx, ch, BridgeAgentEvent(ev))
				})
				cmds = append(cmds, ListenForEvents(m.eventChan))
				return m, tea.Batch(cmds...)
			}
		}

		// Parse REVISE: lines from response
		var revisions []ReviseDecisionMsg
		cleaned := msg.Text
		for _, line := range strings.Split(msg.Text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "REVISE:") {
				parts := strings.SplitN(strings.TrimPrefix(line, "REVISE:"), "|", 2)
				if len(parts) == 2 {
					id := strings.TrimSpace(parts[0])
					answer := strings.TrimSpace(parts[1])
					if id != "" && answer != "" {
						revisions = append(revisions, ReviseDecisionMsg{ID: id, NewAnswer: answer})
					}
				}
				// Strip the REVISE line from displayed text
				cleaned = strings.Replace(cleaned, line, "", 1)
			}
		}

		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: cleaned})
		}
		if len(m.tree.chatLog) > 200 {
			m.tree.chatLog = m.tree.chatLog[len(m.tree.chatLog)-200:]
		}

		// Apply revisions through the normal cascade path
		if len(revisions) > 0 {
			m.pendingRevise = &revisions[0]
			// Queue remaining revisions for subsequent Update cycles
			for i := 1; i < len(revisions); i++ {
				rev := revisions[i]
				// Apply directly since pendingRevise only holds one
				for j := range m.tree.decisions {
					if strings.EqualFold(m.tree.decisions[j].ID, rev.ID) {
						m.tree.decisions[j].SetAnswer(rev.NewAnswer, "user")
						m.tree.decisions[j].Delegated = false
						break
					}
				}
			}
		}

		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case WhyDecisionMsg:
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx
			go func() {
				resp := runSimpleChat(ctx, m.provider,
					"Explain tradeoffs concisely.",
					"Explain tradeoffs of choosing \""+msg.Label+"\" for decision "+msg.ID)
				safeSend(ctx, ch, WhyResponseMsg{Text: resp})
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case AskDecisionMsg:
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx
			go func() {
				resp := runSimpleChat(ctx, m.provider,
					"Answer concisely.",
					"Question about decision "+msg.ID+": "+msg.Question)
				safeSend(ctx, ch, WhyResponseMsg{Text: resp})
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case SaveFeaturesMsg:
		// Persist decision features to store
		if store, _ := decision.LoadStore(m.cwd); store != nil {
			store.Decisions = m.tree.decisions
			_ = decision.SaveStore(m.cwd, store)
		}
		if m.manager != nil {
			m.manager.SyncDecisions(m.tree.decisions)
		}
		return m, nil

	case SuggestDecisionMsg:
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx
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
			go func() {
				resp := runSimpleChat(ctx, m.provider,
					"You output JSON arrays of options. Nothing else.",
					prompt)
				opts := parseSuggestedOptions(resp)
				if len(opts) > 0 {
					safeSend(ctx, ch, SuggestResponseMsg{ID: suggestID, Options: opts})
				} else {
					safeSend(ctx, ch, WhyResponseMsg{Text: resp})
				}
			}()
		}
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)
	}

	// Delegate to active sub-model.
	// Pass ALL message types (not just KeyMsg) so textinput components
	// receive cursor blink commands and other internal messages.
	switch m.view {
	case ViewPriorities:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.priorities, cmd = m.priorities.Update(keyMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewMain:
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) View() string {
	// Render header
	header := ""
	headerHeight := 0
	if m.showMascot {
		header = m.renderHeader(m.width)
		headerHeight = strings.Count(header, "\n") + 2
	} else {
		header = m.renderCompactHeader(m.width)
		headerHeight = strings.Count(header, "\n") + 1
	}

	// Remaining height for the main panel
	panelHeight := m.height - headerHeight
	if panelHeight < 10 {
		panelHeight = 10
	}

	var base string
	switch m.view {
	case ViewPriorities:
		m.priorities.height = panelHeight
		base = m.priorities.View()

	case ViewMain:
		m.tree.height = panelHeight
		m.tree.width = m.width
		base = m.tree.View()

	default:
		base = ""
	}

	// Combine header + main panel
	if header != "" {
		base = header + "\n" + base
	}

	// If a permission overlay is pending, render it on top of the base view
	if m.pendingPermission != nil {
		base = m.overlayPermission(base, m.width, m.height)
	}

	return base
}

// overlayPermission renders the permission request box centered over the base view.
// renderHeader renders the mascot + info section above the main panel.
func (m Model) renderHeader(width int) string {
	// Determine mascot mood from multiple sources
	mood := StatusToMood(m.tree.overallStatus)
	if m.tree.pendingCount > 0 {
		mood = MoodAsking // pending decisions need attention
	} else if mood == MoodIdle && m.tree.chatThinking {
		mood = MoodActive
	}
	sz := m.mascotSize
	if sz == 0 {
		sz = displaySize // default
	}
	mascot := RenderMascotAtSize(mood, m.mascotTick, sz, eyeGap)
	mascotLines := strings.Split(mascot, "\n")

	// Info panel on the right of the mascot
	cwd := m.cwd
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cwd = strings.Replace(cwd, home, "~", 1)
	}

	var info []string
	info = append(info, BoldAccent.Render("defer")+" "+DimStyle.Render("v"+m.version))
	info = append(info, DimStyle.Render("model: ")+m.modelName)
	info = append(info, DimStyle.Render("cwd: ")+DimStyle.Render(cwd))
	if m.task != "" {
		taskDisplay := m.task
		if len(taskDisplay) > 50 {
			taskDisplay = taskDisplay[:47] + "..."
		}
		info = append(info, DimStyle.Render("task: ")+taskDisplay)
	}
	status := m.tree.overallStatus
	if m.tree.chatThinking {
		status = "thinking"
	} else if status == "" {
		status = "ready"
	}
	info = append(info, DimStyle.Render("status: ")+status)

	if m.updateAvailable != "" {
		info = append(info, YellowStyle.Render("Update available: v"+m.updateAvailable)+" "+DimStyle.Render("— run ")+AccentStyle.Render("defer update"))
	}

	// Join mascot and info side by side
	mascotWidth := 0
	for _, ml := range mascotLines {
		w := lipgloss.Width(ml)
		if w > mascotWidth {
			mascotWidth = w
		}
	}

	maxLines := len(mascotLines)
	if len(info) > maxLines {
		maxLines = len(info)
	}

	// Build the combined mascot + info block
	gap := "   "
	var rawLines []string
	contentWidth := 0
	for i := 0; i < maxLines; i++ {
		left := ""
		if i < len(mascotLines) {
			left = mascotLines[i]
		}
		leftPad := mascotWidth - lipgloss.Width(left)
		if leftPad < 0 {
			leftPad = 0
		}
		left += strings.Repeat(" ", leftPad)

		right := ""
		if i < len(info) {
			right = info[i]
		}
		line := left + gap + right
		rawLines = append(rawLines, line)
		if w := lipgloss.Width(line); w > contentWidth {
			contentWidth = w
		}
	}

	// Center horizontally
	var headerLines []string
	for _, line := range rawLines {
		padLeft := (m.width - contentWidth) / 2
		if padLeft < 1 {
			padLeft = 1
		}
		headerLines = append(headerLines, strings.Repeat(" ", padLeft)+line)
	}

	return "\n" + strings.Join(headerLines, "\n")
}

// renderCompactHeader renders a single-line header when mascot is disabled.
func (m Model) renderCompactHeader(width int) string {
	cwd := m.cwd
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cwd = strings.Replace(cwd, home, "~", 1)
	}

	status := m.tree.overallStatus
	if m.tree.chatThinking {
		status = "thinking"
	} else if status == "" {
		status = "ready"
	}

	left := BoldAccent.Render("defer") + " " + DimStyle.Render("v"+m.version)
	mid := DimStyle.Render("model: ") + m.modelName + DimStyle.Render("  cwd: ") + DimStyle.Render(cwd)
	right := DimStyle.Render("status: ") + status

	line := " " + left + "  " + mid + "  " + right
	return line
}

func (m Model) overlayPermission(base string, width, height int) string {
	if m.pendingPermission == nil {
		return base
	}

	// Build the overlay content
	toolLine := BoldAccent.Render(m.pendingPermission.ToolName) + ": " + m.pendingPermission.Description
	// Truncate if too wide
	maxContentWidth := width - 8
	if maxContentWidth < 30 {
		maxContentWidth = 30
	}
	if lipgloss.Width(toolLine) > maxContentWidth {
		// Truncate the description part
		desc := m.pendingPermission.Description
		maxDesc := maxContentWidth - lipgloss.Width(m.pendingPermission.ToolName) - 6 // ": " + "..."
		if maxDesc > 3 {
			desc = desc[:maxDesc] + "..."
		}
		toolLine = BoldAccent.Render(m.pendingPermission.ToolName) + ": " + desc
	}

	prompt := "Allow this tool to run?"
	keys := GreenStyle.Render("y") + " allow    " + RedStyle.Render("n") + " deny"

	content := "\n" + toolLine + "\n\n" + prompt + "\n\n" + keys + "\n"

	boxWidth := lipgloss.Width(toolLine) + 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	box := buildBorderedBox(content, boxWidth, "Permission Required", "")

	// Split base and box into lines for overlay
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")

	// Center the box vertically and horizontally
	boxHeight := len(boxLines)
	startRow := (height - boxHeight) / 2
	if startRow < 0 {
		startRow = 0
	}

	boxVisualWidth := 0
	for _, bl := range boxLines {
		w := lipgloss.Width(bl)
		if w > boxVisualWidth {
			boxVisualWidth = w
		}
	}
	startCol := (width - boxVisualWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Ensure base has enough lines
	for len(baseLines) < height {
		baseLines = append(baseLines, strings.Repeat(" ", width))
	}

	// Overlay box lines onto base
	for i, boxLine := range boxLines {
		row := startRow + i
		if row >= len(baseLines) {
			break
		}

		baseLine := baseLines[row]
		// Pad baseLine if needed
		baseVisWidth := lipgloss.Width(baseLine)
		if baseVisWidth < width {
			baseLine += strings.Repeat(" ", width-baseVisWidth)
		}

		// Build: leading base chars + box line + trailing base chars
		// Since we're working with ANSI strings, we use a simpler approach:
		// pad left + box + pad right (replacing the base)
		left := strings.Repeat(" ", startCol)
		baseLines[row] = left + boxLine
	}

	return strings.Join(baseLines, "\n")
}

// tryParseChangeCommand detects "@ID change to X" or "@ID = X" patterns and updates the decision.
// Returns true if a change was made.
func (m *Model) tryParseChangeCommand(text string) bool {
	// Look for @ID patterns
	words := strings.Fields(text)
	if len(words) < 3 {
		return false
	}

	// Find the @ID — IDs are stored with @ prefix
	var targetID string
	var restIdx int
	for i, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			targetID = strings.TrimPrefix(word, "@") // strip @ — IDs stored without prefix
			restIdx = i + 1
			break
		}
	}
	if targetID == "" || restIdx >= len(words) {
		return false
	}

	rest := strings.Join(words[restIdx:], " ")
	restLower := strings.ToLower(rest)

	// Detect change intent: "change to X", "= X", "switch to X", "use X", "set to X"
	var newAnswer string
	for _, prefix := range []string{"change to ", "switch to ", "set to ", "use ", "= "} {
		if strings.HasPrefix(restLower, prefix) {
			newAnswer = strings.TrimSpace(rest[len(prefix):])
			break
		}
	}
	if newAnswer == "" {
		return false
	}

	// Find the decision and dispatch ReviseDecisionMsg so the full cascade
	// (invalidation, persistence, re-execution) is triggered.
	found := false
	for _, d := range m.tree.decisions {
		if strings.EqualFold(d.ID, targetID) {
			found = true
			// Queue the revision as a message — it will be handled by the
			// ReviseDecisionMsg case which does cascade + persist + re-execute.
			m.pendingRevise = &ReviseDecisionMsg{ID: d.ID, NewAnswer: newAnswer}
			break
		}
	}
	if !found {
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
			Type: "system",
			Text: fmt.Sprintf("Decision %s not found", targetID),
		})
	}
	return true
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

// isTopicTool returns true ONLY for clearly high-level operations.
// Default is subtool (false) — most things nest under the last topic.
func isTopicTool(desc string) bool {
	lower := strings.ToLower(desc)
	// Only these specific patterns are topics (new context, not sub-operations):
	topicExact := []string{
		"planning approach...",
		"plan complete, executing...",
	}
	for _, p := range topicExact {
		if lower == p {
			return true
		}
	}
	// Agent spawns with short imperative descriptions are topics
	topicPrefixes := []string{
		"explore ", "research ", "design ", "implement ",
		"investigate ", "scaffold ",
	}
	for _, p := range topicPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// Everything else is a subtool
	return false
}

// signalExecutorContinue tells all waiting executors to resume.
func (m Model) signalExecutorContinue() {
	if m.manager == nil {
		return
	}
	for _, exec := range m.manager.Executors() {
		if exec.State().Status == agent.DomainWaiting {
			select {
			case exec.ContinueCh <- struct{}{}:
			default: // already signalled
			}
		}
	}
}

// cleanAgentResponse strips internal agent reasoning from the response text.
// Removes: JSON blocks, TASK: prefix, markdown headers (##), empty repo notices,
// decision format hints, and other internal text not meant for the user.
func cleanAgentResponse(text string, hasTaskPrefix bool) string {
	// Strip JSON/decision blocks
	cleaned := stripJSONBlocks(text)

	// Strip TASK: line
	if hasTaskPrefix {
		if parts := strings.SplitN(strings.TrimSpace(cleaned), "\n", 2); len(parts) > 1 {
			cleaned = strings.TrimSpace(parts[1])
		} else {
			cleaned = ""
		}
	}

	// Strip internal reasoning lines
	var result []string
	for _, line := range strings.Split(cleaned, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip markdown headers that look like internal reasoning
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			continue
		}
		// Skip TASK: lines that weren't caught above
		if strings.HasPrefix(trimmed, "TASK:") {
			continue
		}
		// Skip "Empty repo" and similar internal notes
		if strings.Contains(strings.ToLower(trimmed), "empty repo") ||
			strings.Contains(strings.ToLower(trimmed), "all decisions are new") ||
			strings.Contains(strings.ToLower(trimmed), "answer the decisions above") ||
			strings.Contains(strings.ToLower(trimmed), "let me map out") {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func (m Model) countPending() int {
	count := 0
	for _, d := range m.tree.decisions {
		if d.IsPending() {
			count++
		}
	}
	return count
}

func decisionSummaryForAgent(decs []decision.Decision) string {
	var b strings.Builder
	for _, d := range decs {
		answer := "(pending)"
		if d.Answer != nil {
			answer = *d.Answer
		}
		b.WriteString(fmt.Sprintf("%s [%s, impact %d]: %s → %s\n", d.ID, d.Category, d.Impact, d.Question, answer))
	}
	return b.String()
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

// runSimpleChat runs a one-shot completion and returns the text response.
// Blocks until done. Safe to call from goroutines — respects context cancellation.
// runSimpleChat runs a completion and returns the text. Does NOT stream tool events.
// Use runStreamingChat when you need tool activity to show in the UI.
func runSimpleChat(ctx context.Context, provider api.Provider, systemPrompt, userPrompt string) string {
	events := make(chan api.Event, 100)
	go provider.RunCompletion(ctx, systemPrompt, userPrompt, events)
	var text string
	for ev := range events {
		if ev.Type == api.EventTextDelta {
			text += ev.Text
		}
		// Auto-allow permission requests in simple completions (internal agent calls)
		if ev.Type == api.EventPermissionRequest && ev.PermissionReq != nil {
			ev.PermissionReq.ResponseCh <- api.PermissionResponse{Allow: true}
		}
		if ev.Type == api.EventDone || ev.Type == api.EventError {
			break
		}
	}
	if text == "" {
		return "(no response)"
	}
	return text
}

// runStreamingChat runs a completion while forwarding tool events to the UI channel.
// This gives users visibility into what the agent is doing (reading files, running commands, etc).
func runStreamingChat(ctx context.Context, provider api.Provider, systemPrompt, userPrompt string, uiChan chan<- tea.Msg) string {
	events := make(chan api.Event, 100)
	go provider.RunCompletion(ctx, systemPrompt, userPrompt, events)
	var text string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			text += ev.Text
		case api.EventToolCallStart:
			if ev.ToolCall != nil {
				safeSend(ctx, uiChan, ToolActivityMsg{Description: ev.ToolCall.HumanDescription()})
			}
		case api.EventPermissionRequest:
			if ev.PermissionReq != nil {
				safeSend(ctx, uiChan, PermissionRequestMsg{
					ToolName:    ev.PermissionReq.ToolName,
					Description: permissionDescription(ev.PermissionReq),
					Input:       ev.PermissionReq.Input,
					ResponseCh:  ev.PermissionReq.ResponseCh,
				})
			}
		case api.EventDone, api.EventError:
			if text == "" {
				return "(no response)"
			}
			return text
		}
	}
	if text == "" {
		return "(no response)"
	}
	return text
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

