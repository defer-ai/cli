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
)

// View represents which screen is active.
type View int

const (
	ViewChat View = iota // conversation is the primary view
	ViewPriorities               // care level picker (overlay, returns to conversation)
	ViewTree                     // decision tree (tab toggles)
)

// Model is the root Bubbletea model.
// ModelOpts configures the TUI model.
type ModelOpts struct {
	ShowMascot bool
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
	quitting        bool // set on quit, prevents new goroutine spawns

	// Permission overlay
	pendingPermission *PermissionRequestMsg

	// Priorities (persisted)
	domainPriorities map[string]agent.CareLevel
}

// NewModel creates the root model.
func NewModel(task string, provider api.Provider, cwd string, opts ...ModelOpts) Model {
	ctx, cancel := context.WithCancel(context.Background())
	tree := NewTreeModel()
	tree.domainStatuses = make(map[string]string)
	// Conversation is the default mode — chat input focused
	tree.mode = tmChat
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
		version:          o.Version,
		modelName:        o.ModelName,
		tree:             tree,
		eventChan:        make(chan tea.Msg, 100),
		ctx:              ctx,
		cancel:           cancel,
		domainPriorities: make(map[string]agent.CareLevel),
		notifications:    NewNotificationManager(),
		view:             ViewChat,
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
			// Resume existing session
			m.tree.decisions = store.Decisions
			m.domainPriorities = priorities
			m.task = store.Task

			// Check if any decisions already have answers
			hasAnswered := false
			for _, d := range store.Decisions {
				if d.Answer != nil {
					hasAnswered = true
					break
				}
			}

			if hasAnswered {
				// Session has progress — resume directly in conversation
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: fmt.Sprintf("Resumed session: %s", store.Task)})
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: fmt.Sprintf("%d decisions loaded. Tab to view decision tree.", len(store.Decisions))})

				hasPending := false
				for _, d := range m.tree.decisionItems() {
					if d.IsPending() {
						hasPending = true
						break
					}
				}
				if !hasPending {
					m.tree.overallStatus = "done"
				} else {
					m.tree.overallStatus = "thinking"
				}
			} else {
				// All decisions pending, no answers yet — show priorities picker
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: fmt.Sprintf("Resumed session: %s (%d pending decisions)", store.Task, len(store.Decisions))})
				m.view = ViewPriorities
				m.priorities = NewPrioritiesModel(store.Decisions)
			}
		}
		// else: fresh start, conversation is empty, user types task
	} else {
		// Task given via CLI arg — add as first message and start decomposition
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "user", Text: task})
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Analyzing project and identifying decisions..."})
	}

	return m
}

// NewScanModel creates a model that scans an existing project.
func NewScanModel(provider api.Provider, cwd string) Model {
	ctx, cancel := context.WithCancel(context.Background())
	tree := NewTreeModel()
	tree.domainStatuses = make(map[string]string)
	tree.mode = tmChat
	tree.chatFocused = true
	tree.chatInput.Focus()
	tree.chatLog = append(tree.chatLog, ChatEntry{Type: "system", Text: "Scanning project..."})

	m := Model{
		task:             "(scanning project)",
		provider:         provider,
		cwd:              cwd,
		tree:             tree,
		eventChan:        make(chan tea.Msg, 100),
		ctx:              ctx,
		cancel:           cancel,
		domainPriorities: make(map[string]agent.CareLevel),
		notifications:    NewNotificationManager(),
		view:             ViewChat,
	}
	m.manager = agent.NewManager(provider, cwd)

	// Start scan in background
	ch := m.eventChan
	scanCtx := ctx
	go func() {
		events := make(chan api.Event, 100)
		scanUserPrompt := fmt.Sprintf("Scan the project at %s. Start by using Glob to find all files, then Read the key config files (go.mod, package.json, tsconfig.json, Dockerfile, etc.), then Read source files to understand the architecture. Output ALL discovered decisions.", cwd)
		go provider.RunCompletion(scanCtx, agent.ScanPrompt, scanUserPrompt, events)

		var fullText string
		for ev := range events {
			switch ev.Type {
			case api.EventTextDelta:
				fullText += ev.Text
				safeSend(scanCtx, ch, AgentStateChangedMsg{})
			case api.EventPermissionRequest:
				if ev.PermissionReq != nil {
					safeSend(scanCtx, ch, PermissionRequestMsg{
						ToolName:    ev.PermissionReq.ToolName,
						Description: permissionDescription(ev.PermissionReq),
						Input:       ev.PermissionReq.Input,
						ResponseCh:  ev.PermissionReq.ResponseCh,
					})
				}
			case api.EventDone:
				decs := agent.ParseScanDecisions(fullText)
				// Mark all as discovered
				today := time.Now().Format("2006-01-02")
				for i := range decs {
					if decs[i].Answer == nil && len(decs[i].Options) > 0 {
						answer := decs[i].Options[0].Label
						decs[i].Answer = &answer
					}
					decs[i].Source = "discovered"
					decs[i].Date = today
				}
				// Save immediately
				store, _ := decision.LoadStore(cwd)
				if store == nil {
					store, _ = decision.CreateStore(cwd, "(scanned project)")
				}
				if store != nil {
					store.Decisions = decs
					_ = decision.SaveStore(cwd, store)
				}
				safeSend(scanCtx, ch, AgentDecisionsReadyMsg{Decisions: decs})
				return
			case api.EventError:
				safeSend(scanCtx, ch, AgentDecisionsReadyMsg{Decisions: nil})
				return
			}
		}
	}()

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

	case tea.KeyMsg:
		// Ctrl+Q to quit
		if msg.String() == "ctrl+q" {
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		}
		// Ctrl+C shows a warning
		if msg.String() == "ctrl+c" {
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
		m.view = ViewChat
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
		summary.WriteString("\nSet your care level per domain (how much you want to control).")
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: summary.String()})
		// Switch to priorities picker
		m.view = ViewPriorities
		m.priorities = NewPrioritiesModel(msg.Decisions)
		m.priorities.width = m.width
		m.priorities.height = m.height
		cmds = append(cmds, ListenForEvents(m.eventChan))
		return m, tea.Batch(cmds...)

	case PrioritiesConfirmedMsg:
		m.domainPriorities = msg.Priorities
		savePriorities(m.cwd, msg.Priorities, m.task)
		m.view = ViewChat
		m.tree.mode = tmChat
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
				if level != agent.CareLevelParanoid && level != agent.CareLevelHigh {
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
					d.Answer = &answer
					d.Delegated = true
					d.Source = "auto"
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
			m.tree.overallStatus = "thinking"
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

		// Skip AskUserQuestion-related activity — it can't be handled interactively
		if strings.Contains(msg.Description, "Waiting for input") ||
			strings.Contains(msg.Description, "AskUserQuestion") {
			cmds = append(cmds, ListenForEvents(m.eventChan))
			return m, tea.Batch(cmds...)
		}

		// Classify: topics (Agent, planning, high-level) vs subtools (Bash, Read, etc.)
		isTopic := isTopicTool(msg.Description)

		if isTopic {
			// New topic — add as a top-level entry
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "topic", Text: msg.Description})
		} else {
			// Subtool — attach to the last topic as a child
			attached := false
			for i := len(m.tree.chatLog) - 1; i >= 0; i-- {
				if m.tree.chatLog[i].Type == "topic" {
					m.tree.chatLog[i].Children = append(m.tree.chatLog[i].Children, ChatEntry{Type: "subtool", Text: msg.Description})
					attached = true
					break
				}
			}
			if !attached {
				// No parent topic — show as standalone tool
				m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "tool", Text: msg.Description})
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
		m.tree.chatThinking = false
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "All domains complete. Tab to view decision tree."})
		m.notifications.Push("All domains complete.", NotifyHigh, 0)
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
				answer := msg.NewAnswer
				m.tree.decisions[i].Answer = &answer
				m.tree.decisions[i].Delegated = false
				m.tree.decisions[i].Source = "user"
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

		// If executors already ran, re-execute with updated decisions
		if m.executorsLaunched && changedDecision != nil {
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
				ctx := m.ctx
				m.manager.LaunchExecutors(ctx, m.task, m.tree.decisions, m.domainPriorities, func(ev agent.Event) {
					safeSend(ctx, ch, BridgeAgentEvent(ev))
				})
				m.executorsLaunched = true
				m.tree.overallStatus = "executing"
				m.tree.chatThinking = true
				m.tree.chatThinkStart = time.Now()
				m.notifications.Push("All decisions answered. Launching executors...", NotifyMedium, 5*time.Second)
				cmds = append(cmds, ListenForEvents(m.eventChan))
				return m, tea.Batch(cmds...)
			}
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
		return m, nil

	case ChatMessageMsg:
		text := msg.Text

		// Check for direct change commands: "@STA-0001 change to Go with Gin"
		if changed := m.tryParseChangeCommand(text); changed {
			return m, nil
		}

		// Send to agent — system prompt adapts based on whether we have a task
		if m.provider != nil && !m.quitting {
			ch := m.eventChan
			ctx := m.ctx

			var sysPrompt string
			if m.task == "" {
				// No task yet — conversational mode with task detection
				sysPrompt = `You are defer, a zero-autonomy AI assistant. Have a natural conversation with the user.

When the user describes a project, feature, or task they want built, respond with a brief acknowledgment (1-2 sentences max) of what you'll build. Do NOT ask questions — every uncertainty becomes a structured decision that the user will see in a decision tree.

If the user is just chatting, greeting, or asking questions — respond naturally and concisely.`
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

				sysPrompt = fmt.Sprintf(`You are the defer assistant managing project decisions.

Status: %d answered, %d pending, %d invalidated. Execution: %s.
Task: %s

IMPORTANT:
- Only reference decisions by their CURRENT state shown below
- Decisions marked PENDING need user input before implementation can proceed
- Decisions marked INVALIDATED had their answers cleared due to a dependency change
- Do NOT ask conversational questions — if something is unclear, say what needs to be decided
- Be concise

Current decisions:
%s`, answeredCount, pendingCount, invalidatedCount, execStatus, m.task, decContext.String())
			}

			go func() {
				resp := runStreamingChat(ctx, m.provider, sysPrompt, text, ch)
				safeSend(ctx, ch, ChatResponseMsg{Text: resp})
			}()
			cmds = append(cmds, ListenForEvents(m.eventChan))
		}
		return m, tea.Batch(cmds...)

	case ChatResponseMsg:
		m.tree.chatThinking = false

		if m.task == "" {
			// Check if response contains a defer-decisions block
			decs := agent.ParseScanDecisions(msg.Text)
			if len(decs) > 0 {
				// Agent output decisions directly — use them
				for i := len(m.tree.chatLog) - 1; i >= 0; i-- {
					if m.tree.chatLog[i].Type == "user" {
						m.task = m.tree.chatLog[i].Text
						break
					}
				}
				if m.task == "" {
					m.task = "(from conversation)"
				}
				cleaned := stripJSONBlocks(msg.Text)
				if strings.TrimSpace(cleaned) != "" {
					m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: cleaned})
				}
				return m, func() tea.Msg {
					return AgentDecisionsReadyMsg{Decisions: decs}
				}
			}

			// No decisions block — check if the agent is planning something
			// (substantive response to a substantive request = task detected)
			if looksLikeTaskResponse(msg.Text) {
				// Extract task from the conversation
				var taskDesc string
				for i := len(m.tree.chatLog) - 1; i >= 0; i-- {
					if m.tree.chatLog[i].Type == "user" {
						taskDesc = m.tree.chatLog[i].Text
						break
					}
				}
				if taskDesc != "" && len(taskDesc) > 10 {
					m.task = taskDesc
					m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: msg.Text})
					m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "system", Text: "Identifying decisions..."})

					// Run full decomposition with the proper prompt
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
		}

		// Normal chat response
		m.tree.chatLog = append(m.tree.chatLog, ChatEntry{Type: "agent", Text: msg.Text})
		if len(m.tree.chatLog) > 200 {
			m.tree.chatLog = m.tree.chatLog[len(m.tree.chatLog)-200:]
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
	case ViewChat, ViewTree:
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
	// Render header (mascot + info) if enabled
	header := ""
	headerHeight := 0
	if m.showMascot {
		header = m.renderHeader(m.width)
		headerHeight = strings.Count(header, "\n") + 2 // +1 for the line itself, +1 for gap
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

	case ViewChat, ViewTree:
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
	mascot := RenderMascot(StatusToMood(m.tree.overallStatus), m.mascotTick)
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
	if status == "" {
		status = "ready"
	}
	info = append(info, DimStyle.Render("status: ")+status)

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

	var headerLines []string
	gap := "   "
	for i := 0; i < maxLines; i++ {
		left := ""
		if i < len(mascotLines) {
			left = mascotLines[i]
		}
		// Pad left to mascot width
		leftPad := mascotWidth - lipgloss.Width(left)
		if leftPad < 0 {
			leftPad = 0
		}
		left += strings.Repeat(" ", leftPad)

		right := ""
		if i < len(info) {
			right = info[i]
		}
		headerLines = append(headerLines, " "+left+gap+right)
	}

	return strings.Join(headerLines, "\n")
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
			targetID = word // keep the @ prefix, IDs are stored as @STA-0001
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

	// Find and update the decision
	for i := range m.tree.decisions {
		if strings.EqualFold(m.tree.decisions[i].ID, targetID) {
			m.tree.decisions[i].Answer = &newAnswer
			m.tree.decisions[i].Delegated = false
			m.tree.decisions[i].Source = "user"

			// Add confirmation to chat
			m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
				Type: "action",
				Text: fmt.Sprintf("Updated %s → %s", targetID, newAnswer),
			})

			// Persist
			if store, _ := decision.LoadStore(m.cwd); store != nil {
				store.Decisions = m.tree.decisions
				_ = decision.SaveStore(m.cwd, store)
			}
			if m.manager != nil {
				m.manager.SyncDecisions(m.tree.decisions)
			}
			return true
		}
	}

	m.tree.chatLog = append(m.tree.chatLog, ChatEntry{
		Type: "system",
		Text: fmt.Sprintf("Decision %s not found", targetID),
	})
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

// looksLikeTaskResponse returns true if the agent's response indicates it's
// planning or preparing to build something (as opposed to casual conversation).
func looksLikeTaskResponse(text string) bool {
	if len(text) < 100 {
		return false // short responses are probably casual chat
	}
	lower := strings.ToLower(text)
	// Planning/building signals
	signals := []string{
		"let me plan", "here's the plan", "here is the plan",
		"i'll break this down", "let me break this",
		"key decisions", "decisions we need", "decisions to make",
		"architecture", "tech stack", "framework",
		"here's what i", "here is what i",
		"let me start", "i'll start by",
		"first, we need", "first step",
		"components", "features", "requirements",
		"let's build", "we can build", "i'll build",
		"implementation", "project structure",
	}
	for _, sig := range signals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// isTopicTool returns true if the tool description represents a high-level topic
// (Agent spawn, planning) vs a subprocess call (Bash, Read, Write, etc.).
func isTopicTool(desc string) bool {
	lower := strings.ToLower(desc)
	// Topics: agent spawns, planning, looking up tools
	topicPrefixes := []string{
		"explore", "research", "design", "build", "implement",
		"plan", "investigate", "analyze", "check", "review",
		"looking up", "planning approach",
	}
	for _, p := range topicPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// Subtools: running commands, reading/writing files, searching, fetching
	subtoolPrefixes := []string{
		"running:", "reading ", "creating ", "editing ", "searching ",
		"finding ", "fetching ", "waiting", "plan complete",
	}
	for _, p := range subtoolPrefixes {
		if strings.HasPrefix(lower, p) {
			return false
		}
	}
	// Default: if it looks like a sentence/description (Agent tool), it's a topic
	if len(desc) > 20 && !strings.Contains(lower, "/") {
		return true
	}
	return false
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

