package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

type treeMode int

const (
	tmTree treeMode = iota
	tmDetail
	tmRevise
	tmAsk
	tmChat         // full-screen chat (narrow fallback only)
	tmEditFeatures // editing feature tags on a decision
)

// Focus panel constants for side-by-side layout.
const (
	FocusTree     = 0 // left panel (tree)
	FocusChat     = 1 // right panel (chat log)
	FocusResolver = 2 // right panel (pending decisions)

	minSideBySideWidth = 80
)

// ChatEntry is a line in the chat panel.
type ChatEntry struct {
	Type     string // "topic", "subtool", "tool", "agent", "user", "system", "action"
	Text     string
	Expanded bool       // for agent/topic messages: show full content (ctrl+o toggles)
	Children []ChatEntry // for topics: subprocess calls that ran under this topic
}

// TreeModel is the main decision tree view.
type TreeModel struct {
	decisions      []decision.Decision
	execStates     []agent.ExecState
	overallStatus  string
	cursor         int
	optCursor      int
	mode           treeMode
	textInput      textinput.Model
	whyText        string
	width, height  int
	mascotTick     int
	chatLog        []ChatEntry     // chat panel
	chatInput      textinput.Model // chat input
	chatFocused    bool            // true = keys go to chat, false = keys go to tree
	chatThinking   bool            // true while waiting for agent response
	chatThinkStart time.Time       // when thinking started
	pendingCount   int             // number of pending decisions (shown in chat footer)
	chatScrollUp   int             // lines scrolled up from bottom (0 = at bottom)
	completions    []string        // current @ID autocomplete matches
	completionIdx  int             // selected completion (-1 = none)
	activityLine   string          // last tool activity for status bar
	domainStatuses map[string]string // per-domain execution status (key=domain, value=planning|executing|verifying|done|error)
	mdRenderer     *glamour.TermRenderer
	searchMode     bool            // true when search input is active
	searchQuery    string          // current search filter (persists after exiting search mode)
	searchInput    textinput.Model // input for search filtering
	showDetail     bool            // true when a decision is selected and terminal is wide enough for split pane
	sortMode       int             // 0=category, 1=impact, 2=status, 3=alphabetical
	// Jump search (Ctrl+F) — find and jump without filtering
	jumpSearchMode  bool
	jumpSearchInput textinput.Model
	jumpMatches     []jumpMatch
	jumpCursor      int

	// Side-by-side layout — three focus zones
	focusPanel        int  // FocusTree (0), FocusChat (1), FocusResolver (2)
	resolverIdx       int  // which pending decision is shown (0-based into pending list)
	resolverOptIdx    int  // option cursor in the resolver
	showingPriorities bool // true = resolver shows care level picker instead of decisions
	priorityCategories []string              // categories for priorities picker
	priorityLevels     map[string]agent.CareLevel // current care levels
	priorityCursor     int  // which category is selected
}

// jumpMatch represents a match in the Ctrl+F jump search dropdown.
type jumpMatch struct {
	Type  string // "decision", "category", "feature"
	Label string // display label
	Index int    // decision index to jump to (for categories/features: first decision in group)
}

func NewTreeModel() TreeModel {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // we handle wrapping ourselves
	)

	ci := textinput.New()
	ci.Placeholder = "Ask anything..."
	ci.Prompt = AccentStyle.Render("> ")
	ci.CharLimit = 0

	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Prompt = AccentStyle.Render("> ")
	ti.CharLimit = 0

	si := textinput.New()
	si.Placeholder = "Filter decisions..."
	si.Prompt = AccentStyle.Render("/ ")
	si.CharLimit = 0

	ji := textinput.New()
	ji.Placeholder = "Jump to decision, category, or feature..."
	ji.Prompt = AccentStyle.Render("find: ")
	ji.CharLimit = 0

	return TreeModel{mode: tmTree, mdRenderer: r, chatInput: ci, textInput: ti, searchInput: si, jumpSearchInput: ji, completionIdx: -1, focusPanel: FocusChat}
}

// --- Exported setters for snapshot/testing ---

func (m *TreeModel) SetDecisions(decs []decision.Decision) { m.decisions = decs }
func (m *TreeModel) SetChatLog(log []ChatEntry)             { m.chatLog = log }
func (m *TreeModel) SetSize(w, h int)                       { m.width = w; m.height = h }
func (m *TreeModel) SetFocusPanel(f int)                    { m.focusPanel = f }
func (m *TreeModel) SetOverallStatus(s string)              { m.overallStatus = s }

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		// Handle mouse wheel for scrolling the tree or chat
		if msg.Button == tea.MouseButtonWheelUp {
			if m.mode == tmTree || m.mode == tmDetail {
				if m.cursor > 0 {
					m.cursor--
				}
			}
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.mode == tmTree || m.mode == tmDetail {
				if m.cursor < m.decisionCount()-1 {
					m.cursor++
				}
			}
			return m, nil
		}
		return m, nil
	default:
		// Forward non-key messages to the active textinput for cursor blink etc.
		var cmd tea.Cmd
		switch m.mode {
		case tmChat:
			m.chatInput, cmd = m.chatInput.Update(msg)
		case tmRevise, tmAsk:
			m.textInput, cmd = m.textInput.Update(msg)
		}
		// Also forward to chatInput when chat panel is focused (wide layout)
		if m.focusPanel == FocusChat && m.mode != tmChat {
			m.chatInput, cmd = m.chatInput.Update(msg)
		}
		if m.searchMode {
			m.searchInput, cmd = m.searchInput.Update(msg)
		}
		if m.jumpSearchMode {
			m.jumpSearchInput, cmd = m.jumpSearchInput.Update(msg)
		}
		return m, cmd
	}
}

func (m TreeModel) handleKey(msg tea.KeyMsg) (TreeModel, tea.Cmd) {
	key := msg.String()

	isWide := m.width >= minSideBySideWidth

	// Tab toggles focus panel in wide layout, or mode in narrow layout.
	// Exception: chat mode with active completions uses tab for cycling.
	if (key == "tab" || key == "shift+tab") && !((m.mode == tmChat || (isWide && m.focusPanel == FocusChat)) && len(m.completions) > 0) {
		if isWide {
			if key == "tab" {
				// Forward: tree → chat → resolver → tree
				switch m.focusPanel {
				case FocusTree:
					m.focusPanel = FocusChat
					m.chatFocused = true
					m.chatInput.Focus()
				case FocusChat:
					m.focusPanel = FocusResolver
					m.chatFocused = false
					m.chatInput.Blur()
				case FocusResolver:
					m.focusPanel = FocusTree
					m.chatFocused = false
					m.chatInput.Blur()
				}
			} else {
				// Reverse: tree → resolver → chat → tree
				switch m.focusPanel {
				case FocusTree:
					m.focusPanel = FocusResolver
					m.chatFocused = false
					m.chatInput.Blur()
				case FocusChat:
					m.focusPanel = FocusTree
					m.chatFocused = false
					m.chatInput.Blur()
				case FocusResolver:
					m.focusPanel = FocusChat
					m.chatFocused = true
					m.chatInput.Focus()
				}
			}
		} else {
			// Narrow: toggle tmTree <-> tmChat
			if m.mode == tmChat {
				m.mode = tmTree
				m.chatFocused = false
				m.chatInput.Blur()
			} else if m.mode == tmTree {
				m.mode = tmChat
				m.chatFocused = true
				m.chatInput.Focus()
			}
		}
		return m, nil
	}

	// --- Wide layout: chat panel focused ---
	if isWide && m.focusPanel == FocusChat && m.mode != tmDetail && m.mode != tmRevise && m.mode != tmAsk && m.mode != tmEditFeatures {
		return m.handleChatKey(msg)
	}

	// --- Wide layout: resolver panel focused ---
	if isWide && m.focusPanel == FocusResolver {
		return m.handleResolverKey(msg)
	}

	// --- Narrow layout: chat input mode (full screen) ---
	if m.mode == tmChat {
		switch key {
		case "ctrl+o":
			// Toggle expand/collapse on the last topic (only topics are collapsible)
			for i := len(m.chatLog) - 1; i >= 0; i-- {
				if m.chatLog[i].Type == "topic" {
					m.chatLog[i].Expanded = !m.chatLog[i].Expanded
					break
				}
			}
			return m, nil
		case "pgup", "shift+up":
			m.chatScrollUp += 5
			return m, nil
		case "pgdown", "shift+down":
			m.chatScrollUp -= 5
			if m.chatScrollUp < 0 {
				m.chatScrollUp = 0
			}
			return m, nil
		case "esc":
			if m.chatThinking {
				// Stop the agent — emit a cancel signal
				return m, func() tea.Msg { return StopAgentMsg{} }
			}
			m.mode = tmTree
			m.chatFocused = false
			m.chatInput.Reset()
			m.chatInput.Blur()
			m.completions = nil
			m.completionIdx = -1
			return m, nil
		case "tab":
			// Tab cycles through completions if available
			if len(m.completions) > 0 {
				m.completionIdx++
				if m.completionIdx >= len(m.completions) {
					m.completionIdx = 0
				}
				// Replace the @partial with @FULL-ID
				val := m.chatInput.Value()
				lastAt := strings.LastIndex(val, "@")
				if lastAt >= 0 {
					prefix := val[:lastAt]
					newVal := prefix + "@" + m.completions[m.completionIdx]
					m.chatInput.SetValue(newVal)
					m.chatInput.SetCursor(len(newVal))
				}
				return m, nil
			}
			// No completions: toggle back to tree
			m.mode = tmTree
			m.chatFocused = false
			m.chatInput.Blur()
			return m, nil
		case "enter":
			if strings.TrimSpace(m.chatInput.Value()) != "" {
				input := strings.TrimSpace(m.chatInput.Value())
				m.chatLog = append(m.chatLog, ChatEntry{Type: "user", Text: input})
				m.chatScrollUp = 0 // snap to bottom on send
				m.chatInput.Reset()
				m.chatInput.Focus() // keep input focused after sending
				m.chatThinking = true
				m.chatThinkStart = time.Now()
				m.completions = nil
				m.completionIdx = -1

				// Check for @DECISION-ID references
				// Parse: "STA-0001 change to Go" → ReviseDecisionMsg
				// Parse: "STA-0001 why?" → WhyDecisionMsg
				// Otherwise: general chat message
				return m, func() tea.Msg { return ChatMessageMsg{Text: input} }
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.chatInput, cmd = m.chatInput.Update(msg)

			// Update completions based on current input
			m.completions, m.completionIdx = m.updateCompletions()

			return m, cmd
		}
	}

	// --- Text input ---
	if m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		switch key {
		case "esc":
			if m.mode == tmEditFeatures {
				m.mode = tmDetail
				m.textInput.Reset()
				m.textInput.Blur()
				return m, nil
			}
			m.mode = tmDetail
			m.textInput.Reset()
			m.textInput.Blur()
			return m, nil
		case "enter":
			if m.mode == tmEditFeatures {
				sel := m.selected()
				if sel != nil {
					raw := strings.TrimSpace(m.textInput.Value())
					features := parseFeatureTags(raw)
					// Update the decision's features
					for i := range m.decisions {
						if m.decisions[i].ID == sel.ID {
							m.decisions[i].Features = features
							break
						}
					}
				}
				m.mode = tmDetail
				m.textInput.Reset()
				m.textInput.Blur()
				return m, func() tea.Msg { return SaveFeaturesMsg{} }
			}
			if strings.TrimSpace(m.textInput.Value()) != "" {
				sel := m.selected()
				if sel != nil {
					if m.mode == tmRevise {
						m.mode = tmTree
						id := sel.ID
						answer := strings.TrimSpace(m.textInput.Value())
						m.textInput.Reset()
						return m, func() tea.Msg { return ReviseDecisionMsg{ID: id, NewAnswer: answer} }
					} else if m.mode == tmAsk {
						m.mode = tmDetail
						id := sel.ID
						q := strings.TrimSpace(m.textInput.Value())
						m.textInput.Reset()
						m.whyText = "..."
						return m, func() tea.Msg { return AskDecisionMsg{ID: id, Question: q} }
					}
				}
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// --- Detail ---
	if m.mode == tmDetail {
		sel := m.selected()
		switch key {
		case "esc", "q":
			m.mode = tmTree
			m.whyText = ""
			return m, nil
		case "c":
			m.mode = tmRevise
			m.textInput.Reset()
			m.textInput.Placeholder = "Type custom answer..."
			m.textInput.Focus()
			return m, nil
		case "a":
			m.mode = tmAsk
			m.textInput.Reset()
			m.textInput.Placeholder = "Ask a question..."
			m.textInput.Focus()
			m.whyText = ""
			return m, nil
		case "w":
			if sel != nil {
				label := sel.StrAnswer()
				if label == "" && len(sel.Options) > 0 && m.optCursor < len(sel.Options) {
					label = sel.Options[m.optCursor].Label
				}
				if label != "" {
					m.whyText = "..."
					return m, func() tea.Msg { return WhyDecisionMsg{ID: sel.ID, Label: label} }
				}
			}
			return m, nil
		case "s":
			// Shuffle: replace options with AI suggestions
			if sel != nil {
				m.whyText = "Shuffling options..."
				return m, func() tea.Msg { return SuggestDecisionMsg{ID: sel.ID} }
			}
			return m, nil
		case "f":
			// Edit feature tags
			if sel != nil {
				m.mode = tmEditFeatures
				m.textInput.Reset()
				m.textInput.Placeholder = "Comma-separated feature tags..."
				m.textInput.SetValue(strings.Join(sel.Features, ", "))
				m.textInput.Focus()
				m.textInput.SetCursor(len(m.textInput.Value()))
			}
			return m, nil
		}

		// Option navigation -- works for BOTH pending AND answered decisions
		if sel != nil && len(sel.Options) > 0 {
			switch key {
			case "down":
				if m.optCursor < len(sel.Options)-1 {
					m.optCursor++
				}
				return m, nil
			case "up":
				if m.optCursor > 0 {
					m.optCursor--
				}
				return m, nil
			case "enter":
				if m.optCursor < len(sel.Options) {
					id := sel.ID
					answer := sel.Options[m.optCursor].Label
					m.optCursor = 0
					m.mode = tmTree // go back to tree after confirming
					return m, func() tea.Msg { return ReviseDecisionMsg{ID: id, NewAnswer: answer} }
				}
				return m, nil
			}
		}
		return m, nil
	}

	// --- Jump search mode (Ctrl+F) ---
	if m.jumpSearchMode {
		switch key {
		case "esc":
			m.jumpSearchMode = false
			m.jumpSearchInput.Reset()
			m.jumpSearchInput.Blur()
			m.jumpMatches = nil
			m.jumpCursor = 0
			return m, nil
		case "up":
			if m.jumpCursor > 0 {
				m.jumpCursor--
			}
			return m, nil
		case "down":
			if m.jumpCursor < len(m.jumpMatches)-1 {
				m.jumpCursor++
			}
			return m, nil
		case "enter":
			if len(m.jumpMatches) > 0 && m.jumpCursor < len(m.jumpMatches) {
				m.cursor = m.jumpMatches[m.jumpCursor].Index
			}
			m.jumpSearchMode = false
			m.jumpSearchInput.Reset()
			m.jumpSearchInput.Blur()
			m.jumpMatches = nil
			m.jumpCursor = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.jumpSearchInput, cmd = m.jumpSearchInput.Update(msg)
			m.jumpMatches = m.computeJumpMatches(m.jumpSearchInput.Value())
			m.jumpCursor = 0
			return m, cmd
		}
	}

	// --- Tree (search mode) ---
	if m.searchMode {
		switch key {
		case "esc":
			m.searchMode = false
			m.searchQuery = ""
			m.searchInput.Reset()
			m.searchInput.Blur()
			m.cursor = 0
			return m, nil
		case "enter":
			m.searchMode = false
			m.searchInput.Blur()
			// Keep the filter active via m.searchQuery
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.searchQuery = m.searchInput.Value()
			// Clamp cursor to filtered results
			count := m.decisionCount()
			if m.cursor >= count && count > 0 {
				m.cursor = count - 1
			} else if count == 0 {
				m.cursor = 0
			}
			return m, cmd
		}
	}

	// --- Tree ---
	decCount := m.decisionCount()
	switch key {
	case "/":
		m.searchMode = true
		m.searchInput.Focus()
		return m, nil
	case "f", "ctrl+f":
		m.jumpSearchMode = true
		m.jumpSearchInput.Reset()
		m.jumpSearchInput.Focus()
		m.jumpMatches = nil
		m.jumpCursor = 0
		return m, nil
	case "s":
		// Cycle sort: category → impact → status → alphabetical → category
		m.sortMode = (m.sortMode + 1) % 4
		m.cursor = 0
		return m, nil
	case "down":
		if m.cursor < decCount-1 {
			m.cursor++
		}
		return m, nil
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "enter":
		if m.selected() != nil {
			m.mode = tmDetail
			m.whyText = ""
			m.optCursor = 0
		}
		return m, nil
	}

	return m, nil
}

// handleResolverKey handles key events when the resolver panel is focused.
func (m TreeModel) handleResolverKey(msg tea.KeyMsg) (TreeModel, tea.Cmd) {
	key := msg.String()

	// Priorities mode
	if m.showingPriorities {
		switch key {
		case "down":
			if m.priorityCursor < len(m.priorityCategories)-1 {
				m.priorityCursor++
			}
		case "up":
			if m.priorityCursor > 0 {
				m.priorityCursor--
			}
		case "h", "left", "l", "right":
			cat := m.priorityCategories[m.priorityCursor]
			if m.priorityLevels[cat] == agent.CareLevelAuto {
				m.priorityLevels[cat] = agent.CareLevelReview
			} else {
				m.priorityLevels[cat] = agent.CareLevelAuto
			}
		case "enter":
			m.showingPriorities = false
			priorities := make(map[string]agent.CareLevel)
			for k, v := range m.priorityLevels {
				priorities[k] = v
			}
			return m, func() tea.Msg { return PrioritiesConfirmedMsg{Priorities: priorities} }
		}
		return m, nil
	}

	// Pending decisions
	var pendingDecs []decision.Decision
	for _, d := range m.decisions {
		if d.IsPending() {
			pendingDecs = append(pendingDecs, d)
		}
	}
	if len(pendingDecs) == 0 {
		return m, nil
	}

	switch key {
	case "down":
		if m.resolverOptIdx < len(pendingDecs[m.resolverIdx].Options)-1 {
			m.resolverOptIdx++
		}
	case "up":
		if m.resolverOptIdx > 0 {
			m.resolverOptIdx--
		}
	case "right":
		m.resolverIdx++
		if m.resolverIdx >= len(pendingDecs) {
			m.resolverIdx = 0
		}
		m.resolverOptIdx = 0
	case "left":
		m.resolverIdx--
		if m.resolverIdx < 0 {
			m.resolverIdx = len(pendingDecs) - 1
		}
		m.resolverOptIdx = 0
	case "enter":
		idx := m.resolverIdx
		if idx < len(pendingDecs) && m.resolverOptIdx < len(pendingDecs[idx].Options) {
			d := pendingDecs[idx]
			answer := d.Options[m.resolverOptIdx].Label
			m.resolverOptIdx = 0
			// Advance to next pending
			if m.resolverIdx >= len(pendingDecs)-1 {
				m.resolverIdx = 0
			}
			return m, func() tea.Msg { return ReviseDecisionMsg{ID: d.ID, NewAnswer: answer} }
		}
	}
	return m, nil
}

// handleChatKey handles key events when the chat panel is focused (wide layout).
// Only handles chat input + scrolling. Resolver keys are in handleResolverKey.
func (m TreeModel) handleChatKey(msg tea.KeyMsg) (TreeModel, tea.Cmd) {
	key := msg.String()

	// Chat-only handling — resolver keys are in handleResolverKey
	inputEmpty := strings.TrimSpace(m.chatInput.Value()) == ""

	switch key {
	case "ctrl+o":
		// Toggle expand/collapse on the last topic
		for i := len(m.chatLog) - 1; i >= 0; i-- {
			if m.chatLog[i].Type == "topic" {
				m.chatLog[i].Expanded = !m.chatLog[i].Expanded
				break
			}
		}
		return m, nil
	case "pgup", "shift+up":
		m.chatScrollUp += 5
		return m, nil
	case "pgdown", "shift+down":
		m.chatScrollUp -= 5
		if m.chatScrollUp < 0 {
			m.chatScrollUp = 0
		}
		return m, nil
	case "esc":
		if m.chatThinking {
			return m, func() tea.Msg { return StopAgentMsg{} }
		}
		// If input has content, clear it; otherwise switch to tree
		if !inputEmpty {
			m.chatInput.Reset()
			m.completions = nil
			m.completionIdx = -1
		} else {
			m.focusPanel = FocusTree
			m.chatFocused = false
			m.chatInput.Blur()
		}
		return m, nil
	case "tab":
		// Tab cycles through completions if available
		if len(m.completions) > 0 {
			m.completionIdx++
			if m.completionIdx >= len(m.completions) {
				m.completionIdx = 0
			}
			val := m.chatInput.Value()
			lastAt := strings.LastIndex(val, "@")
			if lastAt >= 0 {
				prefix := val[:lastAt]
				newVal := prefix + "@" + m.completions[m.completionIdx]
				m.chatInput.SetValue(newVal)
				m.chatInput.SetCursor(len(newVal))
			}
			return m, nil
		}
		// No completions: toggle to tree
		m.focusPanel = FocusTree
		m.chatFocused = false
		m.chatInput.Blur()
		return m, nil
	case "down":
		if inputEmpty {
			m.chatScrollUp -= 3
			if m.chatScrollUp < 0 {
				m.chatScrollUp = 0
			}
			return m, nil
		}
		// Fall through to text input
	case "up":
		if inputEmpty {
			m.chatScrollUp += 3
			return m, nil
		}
		// Fall through to text input
	case "enter":
		if !inputEmpty {
			// Send chat message
			input := strings.TrimSpace(m.chatInput.Value())
			m.chatLog = append(m.chatLog, ChatEntry{Type: "user", Text: input})
			m.chatScrollUp = 0
			m.chatInput.Reset()
			m.chatInput.Focus()
			m.chatThinking = true
			m.chatThinkStart = time.Now()
			m.completions = nil
			m.completionIdx = -1
			return m, func() tea.Msg { return ChatMessageMsg{Text: input} }
		}
		return m, nil
	}

	// Default: forward to chat input
	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	m.completions, m.completionIdx = m.updateCompletions()
	return m, cmd
}

func (m TreeModel) selected() *decision.Decision {
	decs := m.decisionItems()
	if m.cursor >= 0 && m.cursor < len(decs) {
		return &decs[m.cursor]
	}
	return nil
}

func (m TreeModel) decisionItems() []decision.Decision {
	var items []decision.Decision
	for _, d := range m.decisions {
		if d.Question == "Uncategorized implementation decisions" && d.Source == "auto" {
			continue
		}
		items = append(items, d)
	}
	// Apply search filter
	if m.searchQuery != "" {
		q := strings.ToLower(m.searchQuery)
		var filtered []decision.Decision
		for _, d := range items {
			if strings.Contains(strings.ToLower(d.Question), q) ||
				strings.Contains(strings.ToLower(d.Category), q) ||
				strings.Contains(strings.ToLower(d.ID), q) {
				filtered = append(filtered, d)
			}
		}
		items = filtered
	}
	// Sort based on current mode
	switch m.sortMode {
	case 0: // category
		sortDecisionsByCategory(items)
	case 1: // impact (high first)
		sort.SliceStable(items, func(i, j int) bool { return items[i].Impact > items[j].Impact })
	case 2: // status (pending first)
		sort.SliceStable(items, func(i, j int) bool {
			pi, pj := items[i].IsPending(), items[j].IsPending()
			if pi != pj { return pi }
			return false
		})
	case 3: // alphabetical
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Question) < strings.ToLower(items[j].Question)
		})
	}
	return items
}

// sortLabel returns the display name of the current sort mode.
func sortLabel(mode int) string {
	switch mode {
	case 1: return "impact"
	case 2: return "status"
	case 3: return "a-z"
	default: return "domain"
	}
}

// sortDecisionsByCategory sorts decisions so same-category items are grouped,
// preserving the order of first appearance for categories.
func sortDecisionsByCategory(decs []decision.Decision) {
	if len(decs) <= 1 {
		return
	}
	// Determine category order by first appearance
	catOrder := map[string]int{}
	idx := 0
	for _, d := range decs {
		key := strings.ToLower(strings.TrimSpace(d.Category))
		if _, ok := catOrder[key]; !ok {
			catOrder[key] = idx
			idx++
		}
	}
	// Stable sort by category order
	sort.SliceStable(decs, func(i, j int) bool {
		ki := strings.ToLower(strings.TrimSpace(decs[i].Category))
		kj := strings.ToLower(strings.TrimSpace(decs[j].Category))
		return catOrder[ki] < catOrder[kj]
	})
}

func (m TreeModel) decisionCount() int {
	return len(m.decisionItems())
}

// highlightDecisionRefs highlights @ID patterns (e.g. @STA-0001) in text using AccentStyle.
func highlightDecisionRefs(text string) string {
	return highlightRefs(text)
}

// highlightRefs highlights @DECISION-ID and #FEATURE references in text using AccentStyle.
func highlightRefs(text string) string {
	result := text
	words := strings.Fields(text)
	for _, word := range words {
		if (strings.HasPrefix(word, "@") || strings.HasPrefix(word, "#")) && len(word) > 1 {
			ref := word
			highlighted := AccentStyle.Render(ref)
			result = strings.Replace(result, ref, highlighted, 1)
		}
	}
	return result
}

// getCompletions returns decision IDs that start with the given partial string (case-insensitive).
// Returns at most 5 results.
func getCompletions(decisions []decision.Decision, partial string) []string {
	if partial == "" {
		return nil
	}
	lower := strings.ToLower(partial)
	var matches []string
	for _, d := range decisions {
		if strings.HasPrefix(strings.ToLower(d.ID), lower) {
			matches = append(matches, d.ID)
			if len(matches) >= 5 {
				break
			}
		}
	}
	return matches
}

// updateCompletions checks the current chat input for an @partial prefix and
// returns matching completions and a reset completion index.
func (m TreeModel) updateCompletions() ([]string, int) {
	val := m.chatInput.Value()
	lastAt := strings.LastIndex(val, "@")
	if lastAt < 0 {
		return nil, -1
	}
	// The @-word must be the last word (no space after @partial)
	after := val[lastAt+1:]
	if strings.Contains(after, " ") {
		return nil, -1
	}
	matches := getCompletions(m.decisions, after)
	if len(matches) == 0 {
		return nil, -1
	}
	return matches, -1
}

// toolIcon returns a contextual icon for a tool activity line.
func toolIcon(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "running:") || strings.HasPrefix(lower, "run:"):
		return "$"
	case strings.HasPrefix(lower, "reading"):
		return "→"
	case strings.HasPrefix(lower, "creating"):
		return "+"
	case strings.HasPrefix(lower, "editing"):
		return "~"
	case strings.HasPrefix(lower, "searching"), strings.HasPrefix(lower, "finding"):
		return "?"
	case strings.HasPrefix(lower, "fetching"):
		return "↓"
	case strings.HasPrefix(lower, "planning"):
		return "◇"
	case strings.HasPrefix(lower, "plan complete"):
		return "◆"
	case strings.HasPrefix(lower, "looking up"):
		return "…"
	case strings.HasPrefix(lower, "waiting"):
		return "⏳"
	default:
		return "↳"
	}
}

// toolCallLabel converts a tool description like "Running: npm install" into
// a Claude Code-style label like "Bash(npm install)".
func toolCallLabel(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "running:"):
		arg := strings.TrimSpace(text[len("running:"):])
		return "Bash(" + arg + ")"
	case strings.HasPrefix(lower, "run:"):
		arg := strings.TrimSpace(text[len("run:"):])
		return "Bash(" + arg + ")"
	case strings.HasPrefix(lower, "reading "):
		arg := strings.TrimSpace(text[len("reading "):])
		return "Read(" + arg + ")"
	case strings.HasPrefix(lower, "creating "):
		arg := strings.TrimSpace(text[len("creating "):])
		return "Write(" + arg + ")"
	case strings.HasPrefix(lower, "editing "):
		arg := strings.TrimSpace(text[len("editing "):])
		return "Edit(" + arg + ")"
	case strings.HasPrefix(lower, "searching"):
		arg := strings.TrimSpace(text[len("searching"):])
		if arg != "" && arg[0] == ':' {
			arg = strings.TrimSpace(arg[1:])
		}
		return "Search(" + arg + ")"
	case strings.HasPrefix(lower, "finding"):
		arg := strings.TrimSpace(text[len("finding"):])
		return "Glob(" + arg + ")"
	case strings.HasPrefix(lower, "fetching"):
		arg := strings.TrimSpace(text[len("fetching"):])
		return "Fetch(" + arg + ")"
	default:
		return text
	}
}

// resolveDepIDs converts dependency question strings to decision IDs where possible.
func resolveDepIDs(deps []string, decisions []decision.Decision) []string {
	var result []string
	for _, dep := range deps {
		// If it's already an ID (starts with @), keep it
		if strings.HasPrefix(dep, "@") {
			result = append(result, dep)
			continue
		}
		// Try to find a matching decision by question
		found := false
		depLower := strings.ToLower(strings.TrimSpace(dep))
		for _, d := range decisions {
			if strings.ToLower(strings.TrimSpace(d.Question)) == depLower {
				result = append(result, d.ID)
				found = true
				break
			}
		}
		if !found {
			result = append(result, dep) // fallback to raw text
		}
	}
	return result
}

// thinkingSpinner returns an animated spinner character based on tick.
func thinkingSpinner(tick int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[tick%len(frames)]
}

// thinkingPhrase returns a rotating phrase that changes every ~8 seconds.
func thinkingPhrase(tick int, elapsed time.Duration) string {
	phrases := []string{
		"Thinking...",
		"Working on it...",
		"Processing...",
		"Reasoning...",
		"Analyzing...",
		"Figuring it out...",
		"Almost there...",
		"Crunching...",
		"Considering options...",
		"Connecting the dots...",
	}
	// Change phrase every ~80 ticks (8 seconds at 100ms tick rate)
	idx := (tick / 80) % len(phrases)
	// After 2 minutes, cycle through the longer-wait phrases
	if elapsed > 2*time.Minute {
		latePhrases := []string{
			"Still working...",
			"This is a big one...",
			"Hang tight...",
			"Deep in thought...",
			"Bear with me...",
		}
		idx = (tick / 80) % len(latePhrases)
		return latePhrases[idx]
	}
	return phrases[idx]
}

// renderTag renders a #label with a semantic background color.
func renderTag(label string) string {
	lower := strings.ToLower(strings.TrimSpace(label))
	// Pick background color based on label content
	var bg, fg string
	switch {
	case strings.Contains(lower, "stack") || strings.Contains(lower, "lang"):
		bg, fg = "#1e2840", "#64a0ff" // blue
	case strings.Contains(lower, "data") || strings.Contains(lower, "db") || strings.Contains(lower, "storage"):
		bg, fg = "#2d1e37", "#a078dc" // purple
	case strings.Contains(lower, "api") || strings.Contains(lower, "backend"):
		bg, fg = "#192d1e", "#50c878" // green
	case strings.Contains(lower, "auth") || strings.Contains(lower, "security"):
		bg, fg = "#371e1e", "#f06464" // red
	case strings.Contains(lower, "ui") || strings.Contains(lower, "frontend") || strings.Contains(lower, "style"):
		bg, fg = "#37291e", "#f9a050" // orange
	case strings.Contains(lower, "deploy") || strings.Contains(lower, "infra") || strings.Contains(lower, "ci"):
		bg, fg = "#142d2d", "#50c8c8" // teal
	case strings.Contains(lower, "test"):
		bg, fg = "#2d2d14", "#c8c850" // yellow
	case strings.Contains(lower, "scope") || strings.Contains(lower, "config"):
		bg, fg = "#2d1e28", "#c878a0" // pink
	default:
		bg, fg = "#232323", "#8c8c8c" // neutral gray
	}
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg))
	return style.Render(" #" + lower + " ")
}

func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	// Use visible width (handles unicode, ANSI codes)
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 3 {
		runes := []rune(s)
		if len(runes) > n {
			return string(runes[:n])
		}
		return s
	}
	// Truncate by runes to avoid cutting multi-byte chars
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+3 > n {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// footerAction represents a key-label pair for footer rendering.
type footerAction struct {
	key   string
	label string
}

// renderFooter builds a footer string from a list of actions, fitting as many
// as possible within the given width. Actions are shown in order (most
// important first) and truncated gracefully when space runs out.
func renderFooter(actions []footerAction, width int) string {
	const sep = "  " // two-space separator between actions
	sepLen := len(sep)
	prefix := " " // left padding inside the border
	available := width - len(prefix)
	if available < 0 {
		available = 0
	}

	var parts []string
	used := 0
	for _, a := range actions {
		// Visible width: key + space + label
		visLen := len(a.key) + 1 + len(a.label)
		needed := visLen
		if len(parts) > 0 {
			needed += sepLen
		}
		if used+needed > available {
			break
		}
		parts = append(parts, AccentStyle.Render(a.key)+DimStyle.Render(" "+a.label))
		used += needed
	}
	if len(parts) == 0 {
		return prefix
	}
	return prefix + strings.Join(parts, DimStyle.Render(sep))
}

func (m TreeModel) View() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	// Narrow fallback: same as old tab-switching behavior
	if w < minSideBySideWidth {
		if m.mode == tmChat || m.focusPanel == FocusChat {
			return m.viewChat()
		}
		if m.mode == tmDetail || m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
			return m.viewDetail()
		}
		return m.viewTree()
	}

	h-- // reserve 1 line for global status bar

	// Side-by-side layout — focused panel gets more space
	var treeW, chatW int
	if m.focusPanel == FocusTree {
		treeW = w * 60 / 100
	} else {
		treeW = w * 35 / 100
	}
	chatW = w - treeW

	leftPanel := m.renderLeftPanel(treeW, h)

	// Right side: chat panel on top, resolver panel on bottom
	resolverLines := m.renderResolver(chatW - 4)
	resolverH := len(resolverLines) + 4 // +4 for borders, footer, padding
	if len(resolverLines) == 0 {
		resolverH = 0
	}

	chatH := h - resolverH
	if chatH < 8 {
		chatH = 8
	}

	chatPanel := m.renderChatPanel(chatW, chatH)
	var rightPanel string
	if resolverH > 0 {
		resolverPanel := m.renderResolverPanel(chatW, resolverH, resolverLines)
		rightPanel = chatPanel + "\n" + resolverPanel
	} else {
		// No resolver — chat takes full height
		chatPanel = m.renderChatPanel(chatW, h)
		rightPanel = chatPanel
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Global status bar below all panels
	globalActions := []footerAction{
		{"tab", "next panel"},
		{"shift+tab", "prev panel"},
		{"esc", "stop agent"},
		{"ctrl+q", "quit"},
	}
	globalBar := renderFooter(globalActions, w)

	return panels + "\n" + globalBar
}

// ========== SIDE-BY-SIDE PANEL RENDERERS ==========

// renderLeftPanel renders the left panel (tree or detail) inside a bordered box.
func (m TreeModel) renderLeftPanel(w, h int) string {
	if w < 20 {
		w = 20
	}
	innerWidth := w - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	active := m.focusPanel == FocusTree

	// Use detail view if in detail/revise/ask/editFeatures mode
	if m.mode == tmDetail || m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		return m.renderLeftDetailPanel(innerWidth, h, active)
	}

	// Tree view
	return m.renderLeftTreePanel(innerWidth, h, active)
}

// renderLeftTreePanel renders the decision list for the left panel.
func (m TreeModel) renderLeftTreePanel(innerWidth, h int, active bool) string {
	visibleDecs := m.decisionItems()
	total := len(visibleDecs)
	answered := 0
	pending := 0
	for _, d := range visibleDecs {
		if d.Answer != nil {
			answered++
		} else {
			pending++
		}
	}

	// Status summary
	var statusParts []string
	statusParts = append(statusParts, fmt.Sprintf("%d/%d", answered, total))
	if pending > 0 {
		statusParts = append(statusParts, fmt.Sprintf("○ %d", pending))
	}
	rightStatus := strings.Join(statusParts, " ")

	// Title line with sort indicator
	title := TitleStyle.Render("Decisions")
	sortInfo := DimStyle.Render("by " + sortLabel(m.sortMode))
	left := " " + title + " " + sortInfo
	leftW := lipgloss.Width(left)
	gap := innerWidth - leftW - lipgloss.Width(rightStatus) - 1
	if gap < 1 { gap = 1 }
	titleLine := left + strings.Repeat(" ", gap) + DimStyle.Render(rightStatus) + " "

	var lines []string
	lines = append(lines, titleLine)
	dividerLen := innerWidth - 4
	if dividerLen < 4 { dividerLen = 4 }
	lines = append(lines, "  "+DimStyle.Render(strings.Repeat("─", dividerLen)))

	// Card dimensions: full width, no centering margin
	cardInner := innerWidth - 4
	if cardInner < 10 {
		cardInner = 10
	}

	// Compute card heights to map cursor → scroll position
	type cardInfo struct {
		dec      *decision.Decision
		decIdx   int
		height   int
		startLine int
	}
	var cards []cardInfo
	totalLines := 0
	for i, d := range visibleDecs {
		ch := 4 // top + question + answer + bottom
		hasTags := d.Category != "" || len(d.Features) > 0
		if hasTags {
			ch = 6 // +blank line +tags
		}
		cards = append(cards, cardInfo{dec: &visibleDecs[i], decIdx: i, height: ch, startLine: totalLines})
		totalLines += ch
	}

	// Available height for cards
	fixedLines := 4 // title + divider + footer divider + footer
	if m.searchMode || m.searchQuery != "" {
		fixedLines += 2
	}
	if m.jumpSearchMode {
		dd := len(m.jumpMatches)
		if dd > 8 { dd = 8 }
		fixedLines += 2 + dd
	}
	treeH := h - fixedLines
	if treeH < 5 {
		treeH = 5
	}

	// Scroll: ensure the selected card is visible
	scrollLine := 0
	if m.cursor >= 0 && m.cursor < len(cards) {
		curCard := cards[m.cursor]
		// If card starts before the scroll window, scroll up
		if curCard.startLine < scrollLine {
			scrollLine = curCard.startLine
		}
		// If card ends after the scroll window, scroll down
		if curCard.startLine+curCard.height > scrollLine+treeH {
			scrollLine = curCard.startLine + curCard.height - treeH
		}
	}
	if scrollLine < 0 {
		scrollLine = 0
	}

	// Render cards into a virtual buffer, then slice
	var cardLines []string
	for _, ci := range cards {
		d := ci.dec
		isCur := ci.decIdx == m.cursor

		borderCol := BorderColor
		if isCur && active {
			borderCol = ActiveBorderColor
		}
		bStyle := lipgloss.NewStyle().Foreground(borderCol)

		padLine := func(content string) string {
			w := lipgloss.Width(content)
			right := cardInner - w
			if right < 0 { right = 0 }
			return " " + bStyle.Render("│") + " " + content + strings.Repeat(" ", right) + " " + bStyle.Render("│")
		}

		// Top border with ID
		idLabel := " @" + d.ID + " "
		fillLen := cardInner + 2 - lipgloss.Width(idLabel)
		if fillLen < 0 { fillLen = 0 }
		topLine := " " + bStyle.Render("╭") + bStyle.Render(idLabel) + bStyle.Render(strings.Repeat("─", fillLen)) + bStyle.Render("╮")

		// Question
		qStr := trunc(d.Question, cardInner-1)
		if isCur {
			qStr = BoldWhite.Render(qStr)
		}

		// Answer
		var answerStr string
		if d.Answer != nil {
			ans := trunc(*d.Answer, cardInner-4)
			if isCur {
				answerStr = "  " + GreenStyle.Render(ans)
			} else {
				answerStr = "  " + DimStyle.Render(ans)
			}
		} else {
			answerStr = "  " + YellowStyle.Render("pending")
		}

		// Tags (deduplicated, truncated to fit)
		var tagStr string
		if d.Category != "" || len(d.Features) > 0 {
			var tags []string
			seen := map[string]bool{}

			// Add category first
			if d.Category != "" {
				catKey := strings.ToLower(strings.TrimSpace(d.Category))
				seen[catKey] = true
				tags = append(tags, renderTag(d.Category))
			}

			// Add features, skipping any that match the category
			for _, f := range d.Features {
				fKey := strings.ToLower(strings.TrimSpace(f))
				if seen[fKey] {
					continue
				}
				seen[fKey] = true
				t := renderTag(f)
				testLine := strings.Join(append(tags, t), " ")
				if lipgloss.Width(testLine) > cardInner-1 {
					break
				}
				tags = append(tags, t)
			}
			tagStr = strings.Join(tags, " ")
		}

		bottomLine := " " + bStyle.Render("╰") + bStyle.Render(strings.Repeat("─", cardInner+2)) + bStyle.Render("╯")

		cardLines = append(cardLines, topLine)
		cardLines = append(cardLines, padLine(qStr))
		cardLines = append(cardLines, padLine(answerStr))
		if tagStr != "" {
			cardLines = append(cardLines, padLine("")) // blank line before tags
			cardLines = append(cardLines, padLine(tagStr))
		}
		cardLines = append(cardLines, bottomLine)
	}

	// Slice the visible portion
	endLine := scrollLine + treeH
	if endLine > len(cardLines) {
		endLine = len(cardLines)
	}
	if scrollLine < len(cardLines) {
		lines = append(lines, cardLines[scrollLine:endLine]...)
	}

	// Fill remaining height
	rendered := endLine - scrollLine
	for rendered < treeH {
		lines = append(lines, "")
		rendered++
	}

	// Jump search overlay
	if m.jumpSearchMode {
		lines = append(lines, DimStyle.Render(" "+strings.Repeat("─", innerWidth)))
		lines = append(lines, " "+m.jumpSearchInput.View())
		maxDropdown := 8
		if len(m.jumpMatches) < maxDropdown {
			maxDropdown = len(m.jumpMatches)
		}
		for i := 0; i < maxDropdown; i++ {
			jm := m.jumpMatches[i]
			prefix := "   "
			if i == m.jumpCursor {
				prefix = " " + AccentStyle.Render("> ")
			}
			typeTag := DimStyle.Render("[" + jm.Type + "]")
			label := jm.Label
			if i == m.jumpCursor {
				label = BoldWhite.Render(label)
			}
			lines = append(lines, prefix+typeTag+" "+label)
		}
	}

	// Search bar
	if m.searchMode {
		lines = append(lines, DimStyle.Render(" "+strings.Repeat("─", innerWidth)))
		lines = append(lines, " "+m.searchInput.View())
	} else if m.searchQuery != "" {
		lines = append(lines, DimStyle.Render(" "+strings.Repeat("─", innerWidth)))
		lines = append(lines, " "+DimStyle.Render(fmt.Sprintf("Filtered: %d results", total)))
	}

	// Footer
	lines = append(lines, "  "+DimStyle.Render(strings.Repeat("─", dividerLen)))
	var footerActions []footerAction
	if m.jumpSearchMode {
		footerActions = []footerAction{{"type", "find"}, {"↑↓", "select"}, {"enter", "jump"}, {"esc", "close"}}
	} else if m.searchMode {
		footerActions = []footerAction{{"type", "filter"}, {"enter", "confirm"}, {"esc", "clear"}}
	} else {
		footerActions = []footerAction{{"↑↓", "navigate"}, {"enter", "inspect"}, {"/", "filter"}, {"s", "sort"}}
	}
	lines = append(lines, renderFooter(footerActions, innerWidth))

	return strings.Join(lines, "\n")
}

// renderLeftDetailPanel renders the detail view for the left panel.
func (m TreeModel) renderLeftDetailPanel(innerWidth, h int, active bool) string {
	sel := m.selected()
	if sel == nil {
		// Pad to full height
		var lines []string
		lines = append(lines, "")
		lines = append(lines, " "+DimStyle.Render("No decision selected."))
		remaining := h - 2 - 4 // borders + footer
		for i := 0; i < remaining; i++ {
			lines = append(lines, "")
		}
		lines = append(lines, buildMiddleBorderActive(innerWidth, active))
		lines = append(lines, renderFooter([]footerAction{{"esc", "back"}}, innerWidth))
		content := strings.Join(lines, "\n")
		return buildBorderedBoxActive(content, innerWidth, "", "", active)
	}

	var lines []string
	lines = append(lines, "")

	// Category + impact
	header := " " + DimStyle.Render(sel.Category)
	if sel.Impact >= 7 {
		header += " " + RedStyle.Render(fmt.Sprintf("(%d)", sel.Impact))
	} else if sel.Impact >= 4 {
		header += " " + YellowStyle.Render(fmt.Sprintf("(%d)", sel.Impact))
	}
	if sel.Delegated {
		header += " " + MagentaStyle.Render("auto")
	}
	lines = append(lines, header)

	// Dependencies
	if len(sel.DependsOn) > 0 {
		depIDs := resolveDepIDs(sel.DependsOn, m.decisions)
		lines = append(lines, " "+DimStyle.Render("deps: "+strings.Join(depIDs, ", ")))
	}
	lines = append(lines, "")

	// Question
	qLines := wrapText(sel.Question, innerWidth-2)
	for _, ql := range qLines {
		lines = append(lines, " "+DetailQuestionStyle.Render(ql))
	}
	if sel.Context != "" {
		ctxLines := wrapText(sel.Context, innerWidth-2)
		for _, cl := range ctxLines {
			lines = append(lines, " "+DetailContextStyle.Render(cl))
		}
	}
	lines = append(lines, "")

	// Current answer
	if sel.Answer != nil {
		style := GreenStyle
		if sel.Delegated {
			style = MagentaStyle
		}
		ansLines := wrapText(*sel.Answer, innerWidth-4)
		for _, al := range ansLines {
			lines = append(lines, " "+style.Render(al))
		}
	} else {
		lines = append(lines, " "+YellowStyle.Render("pending"))
	}
	lines = append(lines, "")

	// Options
	if len(sel.Options) > 0 {
		for i, opt := range sel.Options {
			isSel := i == m.optCursor
			isChosen := sel.Answer != nil && opt.Label == *sel.Answer
			cur := "   "
			if isSel {
				cur = " " + AccentStyle.Render("> ")
			}
			style := lipgloss.NewStyle().Foreground(DimGray)
			if isChosen {
				style = GreenStyle
			} else if isSel {
				style = BoldWhite
			}
			optText := trunc(fmt.Sprintf("%s) %s", opt.Key, opt.Label), innerWidth-4)
			lines = append(lines, cur+style.Render(optText))
		}
		lines = append(lines, "")
	}

	// Why text — rendered as markdown
	if m.whyText != "" && m.whyText != "..." && m.whyText != "Shuffling options..." {
		if m.mdRenderer != nil {
			if md, err := m.mdRenderer.Render(m.whyText); err == nil {
				for _, ml := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
					visWidth := lipgloss.Width(ml)
					if visWidth > innerWidth-2 {
						for _, wl := range wrapText(ml, innerWidth-2) {
							lines = append(lines, " "+wl)
						}
					} else {
						lines = append(lines, " "+ml)
					}
				}
				lines = append(lines, "")
			} else {
				for _, wl := range wrapText(m.whyText, innerWidth-2) {
					lines = append(lines, " "+DimStyle.Render(wl))
				}
				lines = append(lines, "")
			}
		}
	} else if m.whyText == "..." || m.whyText == "Shuffling options..." {
		lines = append(lines, " "+AccentStyle.Render(m.whyText))
		lines = append(lines, "")
	}

	// Text input (revise/ask/features)
	if m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		label := "Override:"
		if m.mode == tmAsk {
			label = "Ask:"
		} else if m.mode == tmEditFeatures {
			label = "Features:"
		}
		lines = append(lines, " "+AccentStyle.Render(label))
		lines = append(lines, " "+m.textInput.View())
		lines = append(lines, "")
	}

	// Fill remaining vertical space: h - borders(2) - footer divider(1) - footer(1) = h-4
	remaining := h - len(lines) - 4
	for i := 0; i < remaining; i++ {
		lines = append(lines, "")
	}

	// Footer
	lines = append(lines, buildMiddleBorderActive(innerWidth, active))
	var detailFooterActions []footerAction
	if m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		detailFooterActions = []footerAction{
			{"enter", "submit"},
			{"esc", "cancel"},
		}
	} else {
		detailFooterActions = []footerAction{
			{"↑↓", "pick"},
			{"enter", "confirm"},
			{"c", "custom"},
			{"w", "why"},
			{"q", "back"},
		}
	}
	lines = append(lines, renderFooter(detailFooterActions, innerWidth))

	content := strings.Join(lines, "\n")
	return buildBorderedBoxActive(content, innerWidth, sel.ID, sel.Category, active)
}

// renderChatPanel renders the chat panel as its own bordered box.
func (m TreeModel) renderChatPanel(w, h int) string {
	if w < 20 {
		w = 20
	}
	innerWidth := w - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	active := m.focusPanel == FocusChat

	// Chat content height: h - borders(2) - gap(1) - inputDivider(1) - input(1) - footerDivider(1) - footer(1)
	fixedH := 2 + 1 + 1 + 1 + 1 + 1
	if len(m.completions) > 0 {
		fixedH++
	}
	chatContentH := h - fixedH
	if chatContentH < 3 {
		chatContentH = 3
	}

	var lines []string
	lines = append(lines, "")

	// Render chat entries
	maxTextWidth := innerWidth - 2
	if maxTextWidth < 20 {
		maxTextWidth = 20
	}

	const maxChildLines = 5

	var chatLines []string
	for i, entry := range m.chatLog {
		prevType := ""
		if i > 0 {
			prevType = m.chatLog[i-1].Type
		}

		if prevType != "" {
			switch {
			case prevType == "topic" && entry.Type == "topic":
			case prevType == "tool" && entry.Type == "tool":
			case prevType == "subtool" && entry.Type == "subtool":
			default:
				chatLines = append(chatLines, "")
			}
		}

		switch entry.Type {
		case "topic":
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, AccentStyle.Render("●")+" "+lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(label))
			childCount := len(entry.Children)
			showCount := childCount
			if !entry.Expanded && childCount > maxChildLines {
				showCount = maxChildLines
			}
			for ci := 0; ci < showCount; ci++ {
				child := entry.Children[ci]
				childLabel := toolCallLabel(child.Text)
				for li, wl := range wrapText(childLabel, maxTextWidth-4) {
					if li == 0 {
						chatLines = append(chatLines, DimStyle.Render(" └ "+wl))
					} else {
						chatLines = append(chatLines, DimStyle.Render("   "+wl))
					}
				}
			}
			if !entry.Expanded && childCount > maxChildLines {
				remaining := childCount - maxChildLines
				chatLines = append(chatLines, DimStyle.Render(fmt.Sprintf(" └ ... %d more (ctrl+o to expand)", remaining)))
			}

		case "tool":
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, AccentStyle.Render("●")+" "+lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(label))

		case "agent":
			var renderedLines []string
			if m.mdRenderer != nil {
				if md, err := m.mdRenderer.Render(entry.Text); err == nil {
					for _, ml := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
						visWidth := lipgloss.Width(ml)
						if visWidth > maxTextWidth {
							renderedLines = append(renderedLines, wrapText(ml, maxTextWidth)...)
						} else {
							renderedLines = append(renderedLines, ml)
						}
					}
				}
			}
			if len(renderedLines) == 0 {
				renderedLines = wrapText(entry.Text, maxTextWidth-1)
			}
			for _, rl := range renderedLines {
				chatLines = append(chatLines, strings.TrimRight(rl, " "))
			}

		case "user":
			chatLines = append(chatLines, "")
			wrapped := wrapText(entry.Text, maxTextWidth-5)
			for j, wl := range wrapped {
				styledLine := UserMsgStyle.Render(" " + wl + " ")
				if j == 0 {
					chatLines = append(chatLines, UserMsgStyle.Render(" > ")+styledLine)
				} else {
					chatLines = append(chatLines, UserMsgStyle.Render("   ")+styledLine)
				}
			}
			chatLines = append(chatLines, "")

		case "action":
			for _, wl := range wrapText(entry.Text, maxTextWidth) {
				chatLines = append(chatLines, AccentStyle.Render(wl))
			}

		default:
			for _, wl := range wrapText(entry.Text, maxTextWidth) {
				chatLines = append(chatLines, DimStyle.Render(wl))
			}
		}
	}

	// Thinking indicator
	if m.chatThinking {
		elapsed := time.Since(m.chatThinkStart)
		var timeStr string
		if elapsed < time.Minute {
			timeStr = fmt.Sprintf("%.0fs", elapsed.Seconds())
		} else {
			timeStr = fmt.Sprintf("%dm%ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		}
		spinner := thinkingSpinner(m.mascotTick)
		phrase := thinkingPhrase(m.mascotTick, elapsed)
		chatLines = append(chatLines, "")
		chatLines = append(chatLines, AccentStyle.Render(spinner+" "+phrase+" ")+DimStyle.Render("("+timeStr+")"))
	}

	// Scrolling
	scrollUp := m.chatScrollUp
	maxScroll := len(chatLines) - chatContentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollUp > maxScroll {
		scrollUp = maxScroll
	}

	start := 0
	if len(chatLines) > chatContentH {
		start = len(chatLines) - chatContentH - scrollUp
		if start < 0 {
			start = 0
		}
	}
	end := start + chatContentH
	if end > len(chatLines) {
		end = len(chatLines)
	}
	visible := chatLines[start:end]

	for _, cl := range visible {
		lines = append(lines, cl)
	}

	// Fill remaining chat space
	emptyBelow := chatContentH - len(visible)
	for i := 0; i < emptyBelow; i++ {
		if len(m.chatLog) == 0 && i == 0 {
			lines = append(lines, DimStyle.Render("Describe your project to get started."))
		} else {
			lines = append(lines, "")
		}
	}

	// Input divider + completions + input
	lines = append(lines, buildMiddleBorderActive(innerWidth, active))
	if len(m.completions) > 0 {
		var parts []string
		for i, c := range m.completions {
			label := "@" + c
			if i == m.completionIdx {
				parts = append(parts, AccentStyle.Render(label))
			} else {
				parts = append(parts, DimStyle.Render(label))
			}
		}
		lines = append(lines, " "+strings.Join(parts, "  "))
	}
	promptW := lipgloss.Width(m.chatInput.Prompt)
	m.chatInput.Width = innerWidth - 2 - promptW
	lines = append(lines, m.chatInput.View())

	// Footer
	lines = append(lines, buildMiddleBorderActive(innerWidth, active))
	chatFooterActions := []footerAction{
		{"enter", "send"},
		{"↑↓", "scroll"},
	}
	lines = append(lines, renderFooter(chatFooterActions, innerWidth))

	content := strings.Join(lines, "\n")
	rightStatus := ""
	if m.overallStatus != "" {
		rightStatus = m.overallStatus
	}
	return buildBorderedBoxActive(content, innerWidth, "Chat", rightStatus, active)
}

// renderResolver renders the pending decision resolver section.
// Returns nil if no pending decisions and not showing priorities.
// renderResolverPanel wraps the resolver content in its own bordered box.
func (m TreeModel) renderResolverPanel(w, h int, resolverLines []string) string {
	if w < 20 {
		w = 20
	}
	innerWidth := w - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	active := m.focusPanel == FocusResolver

	var lines []string
	lines = append(lines, resolverLines...)

	// Footer
	lines = append(lines, buildMiddleBorderActive(innerWidth, active))
	footerActions := []footerAction{
		{"↑↓", "pick"},
		{"enter", "confirm"},
		{"←→", "cycle"},
	}
	lines = append(lines, renderFooter(footerActions, innerWidth))

	// Pad to height
	contentLines := len(lines) + 2 // +2 for top/bottom borders
	for contentLines < h {
		lines = append(lines, "")
		contentLines++
	}

	// Title — blinks when there are pending decisions
	title := "Resolver"
	if m.showingPriorities {
		title = "Care Levels"
	} else if m.pendingCount > 0 && !active {
		if (m.mascotTick/5)%2 == 0 {
			title = YellowStyle.Render(fmt.Sprintf("● %d to resolve", m.pendingCount))
		} else {
			title = DimStyle.Render(fmt.Sprintf("● %d to resolve", m.pendingCount))
		}
	}

	content := strings.Join(lines, "\n")
	return buildBorderedBoxActive(content, innerWidth, title, "", active)
}

func (m TreeModel) renderResolver(innerWidth int) []string {
	// Priorities picker mode — show care level toggles per domain
	if m.showingPriorities {
		var lines []string
		lines = append(lines, " "+BoldWhite.Render("Set care level per domain"))
		lines = append(lines, "")
		for i, cat := range m.priorityCategories {
			level := m.priorityLevels[cat]
			cursor := "  "
			if i == m.priorityCursor {
				cursor = AccentStyle.Render("> ")
			}
			// Count decisions in this category
			count := 0
			for _, d := range m.decisions {
				if d.Category == cat {
					count++
				}
			}
			// Button-style toggle: selected has background
			autoStyle := lipgloss.NewStyle().Foreground(DimGray)
			reviewStyle := lipgloss.NewStyle().Foreground(DimGray)
			if level == agent.CareLevelAuto {
				autoStyle = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("15")).Bold(true)
			} else {
				reviewStyle = lipgloss.NewStyle().Background(lipgloss.Color("#f97316")).Foreground(lipgloss.Color("0")).Bold(true)
			}
			catStr := pad(cat, 14)
			countStr := DimStyle.Render(fmt.Sprintf("(%d)", count))
			line := fmt.Sprintf(" %s%s %s %s %s",
				cursor, catStr, countStr,
				autoStyle.Render(" auto "),
				reviewStyle.Render(" review "),
			)
			lines = append(lines, line)
		}
		lines = append(lines, "")
		lines = append(lines, " "+AccentStyle.Render("↑↓")+" navigate  "+AccentStyle.Render("←→")+" toggle  "+AccentStyle.Render("enter")+" confirm")
		return lines
	}

	var pending []decision.Decision
	for _, d := range m.decisions {
		if d.IsPending() {
			pending = append(pending, d)
		}
	}
	if len(pending) == 0 {
		return []string{
			" " + DimStyle.Render("No pending decisions."),
		}
	}

	idx := m.resolverIdx
	if idx >= len(pending) {
		idx = len(pending) - 1
	}
	if idx < 0 {
		idx = 0
	}

	current := pending[idx]

	var lines []string
	// Header
	lines = append(lines, " "+DimStyle.Render(fmt.Sprintf("Pending %d/%d", idx+1, len(pending))))
	lines = append(lines, "")
	// Question
	lines = append(lines, " "+YellowStyle.Render("○")+" "+BoldWhite.Render(trunc(current.Question, innerWidth-4)))
	// Options
	for i, opt := range current.Options {
		cursor := "   "
		style := lipgloss.NewStyle().Foreground(DimGray)
		if i == m.resolverOptIdx {
			cursor = " " + AccentStyle.Render("> ")
			style = AccentStyle
		}
		lines = append(lines, cursor+style.Render(fmt.Sprintf("%s) %s", opt.Key, trunc(opt.Label, innerWidth-8))))
	}

	return lines
}

// ========== FULL-SCREEN CHAT VIEW ==========
func (m TreeModel) viewChat() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	// No bordered box — use full width with small padding
	contentWidth := w - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	var lines []string
	lines = append(lines, "")

	// Chat content area: total height minus fixed UI elements
	// Fixed: status line(1) + gap before input(1) + divider(1) + input(1) + divider(1) + footer(1) = 6
	chatContentH := h - 6
	if len(m.completions) > 0 {
		chatContentH-- // completions overlay takes one line
	}
	if chatContentH < 3 {
		chatContentH = 3
	}

	// Render chat entries with markdown and word-wrap
	maxTextWidth := contentWidth - 2
	if maxTextWidth < 20 {
		maxTextWidth = 20
	}

	const maxChildLines = 5 // collapse topic children after this many

	var chatLines []string
	for i, entry := range m.chatLog {
		prevType := ""
		if i > 0 {
			prevType = m.chatLog[i-1].Type
		}

		// Blank line between different entry groups
		// Tool calls within a group (topic-topic, tool-tool): no blank line
		// Everything else: blank line separator
		if prevType != "" {
			switch {
			case prevType == "topic" && entry.Type == "topic":
				// no blank line between consecutive topics
			case prevType == "tool" && entry.Type == "tool":
				// no blank line between consecutive standalone tools
			case prevType == "subtool" && entry.Type == "subtool":
				// no blank line between consecutive subtools
			default:
				chatLines = append(chatLines, "")
			}
		}

		switch entry.Type {
		case "topic":
			// Orange dot, white text
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, " "+AccentStyle.Render("●")+" "+lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(label))
			// Render children indented with └ connector
			childCount := len(entry.Children)
			showCount := childCount
			if !entry.Expanded && childCount > maxChildLines {
				showCount = maxChildLines
			}
			for ci := 0; ci < showCount; ci++ {
				child := entry.Children[ci]
				childLabel := toolCallLabel(child.Text)
				for li, wl := range wrapText(childLabel, maxTextWidth-5) {
					if li == 0 {
						chatLines = append(chatLines, "  "+DimStyle.Render(" └ "+wl))
					} else {
						chatLines = append(chatLines, "  "+DimStyle.Render("   "+wl))
					}
				}
			}
			if !entry.Expanded && childCount > maxChildLines {
				remaining := childCount - maxChildLines
				chatLines = append(chatLines, "  "+DimStyle.Render(fmt.Sprintf(" └ ... %d more (ctrl+o to expand)", remaining)))
			}

		case "tool":
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, " "+AccentStyle.Render("●")+" "+lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(label))

		case "agent":
			// Render agent response with markdown
			var renderedLines []string
			if m.mdRenderer != nil {
				if md, err := m.mdRenderer.Render(entry.Text); err == nil {
					for _, ml := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
						visWidth := lipgloss.Width(ml)
						if visWidth > maxTextWidth {
							renderedLines = append(renderedLines, wrapText(ml, maxTextWidth)...)
						} else {
							renderedLines = append(renderedLines, ml)
						}
					}
				}
			}
			if len(renderedLines) == 0 {
				// Fallback: plain text
				renderedLines = wrapText(entry.Text, maxTextWidth-1)
			}

			// All lines rendered directly — blank lines between paragraphs preserved
			for _, rl := range renderedLines {
				chatLines = append(chatLines, " "+strings.TrimRight(rl, " "))
			}

		case "user":
			chatLines = append(chatLines, "")
			wrapped := wrapText(entry.Text, maxTextWidth-6)
			for j, wl := range wrapped {
				styledLine := UserMsgStyle.Render(" " + wl + " ")
				if j == 0 {
					chatLines = append(chatLines, " "+UserMsgStyle.Render(" > ")+styledLine)
				} else {
					chatLines = append(chatLines, " "+UserMsgStyle.Render("   ")+styledLine)
				}
			}
			chatLines = append(chatLines, "")

		case "action":
			// Action confirmations rendered in accent (orange)
			for _, wl := range wrapText(entry.Text, maxTextWidth) {
				chatLines = append(chatLines, " "+AccentStyle.Render(wl))
			}

		default:
			// System messages rendered dim
			for _, wl := range wrapText(entry.Text, maxTextWidth) {
				chatLines = append(chatLines, " "+DimStyle.Render(wl))
			}
		}
	}

	// Thinking indicator with animated spinner and rotating phrases
	if m.chatThinking {
		elapsed := time.Since(m.chatThinkStart)
		var timeStr string
		if elapsed < time.Minute {
			timeStr = fmt.Sprintf("%.0fs", elapsed.Seconds())
		} else {
			timeStr = fmt.Sprintf("%dm%ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		}
		spinner := thinkingSpinner(m.mascotTick)
		phrase := thinkingPhrase(m.mascotTick, elapsed)
		chatLines = append(chatLines, "")
		chatLines = append(chatLines, " "+AccentStyle.Render(spinner+" "+phrase+" ")+DimStyle.Render("("+timeStr+")"))
	}

	// Top-to-bottom: content starts at top, scrolls down as it grows
	// When content exceeds viewport, show the latest (auto-scroll to bottom)
	// pgup/pgdown adjusts chatScrollUp to scroll back
	scrollUp := m.chatScrollUp
	maxScroll := len(chatLines) - chatContentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollUp > maxScroll {
		scrollUp = maxScroll
	}

	start := 0
	if len(chatLines) > chatContentH {
		start = len(chatLines) - chatContentH - scrollUp
		if start < 0 {
			start = 0
		}
	}
	end := start + chatContentH
	if end > len(chatLines) {
		end = len(chatLines)
	}
	visible := chatLines[start:end]

	// Render visible content (top-to-bottom)
	for _, cl := range visible {
		lines = append(lines, cl)
	}

	// Fill remaining space below content
	emptyBelow := chatContentH - len(visible)
	for i := 0; i < emptyBelow; i++ {
		if len(m.chatLog) == 0 && i == 0 {
			lines = append(lines, " "+DimStyle.Render("Describe your project to get started, or ask anything."))
		} else {
			lines = append(lines, "")
		}
	}

	// Ensure a gap between content and input
	if len(visible) > 0 {
		lines = append(lines, "")
	}

	// Input divider + completions overlay + input
	lines = append(lines, buildChatDivider(contentWidth))
	if len(m.completions) > 0 {
		var parts []string
		for i, c := range m.completions {
			label := "@" + c
			if i == m.completionIdx {
				parts = append(parts, AccentStyle.Render(label))
			} else {
				parts = append(parts, DimStyle.Render(label))
			}
		}
		lines = append(lines, "  "+strings.Join(parts, "  "))
	}
	// Width = content width - left padding(1) - prompt width - cursor(1)
	promptW := lipgloss.Width(m.chatInput.Prompt)
	m.chatInput.Width = contentWidth - 2 - promptW
	lines = append(lines, " "+m.chatInput.View())

	// Footer divider + actions
	lines = append(lines, buildChatDivider(contentWidth))
	var chatFooterActions []footerAction
	if m.pendingCount > 0 {
		chatFooterActions = append(chatFooterActions, footerAction{"○", fmt.Sprintf("%d pending", m.pendingCount)})
	}
	chatFooterActions = append(chatFooterActions,
		footerAction{"enter", "send"},
		footerAction{"tab", "tree"},
		footerAction{"esc", "stop"},
		footerAction{"ctrl+q", "quit"},
	)
	lines = append(lines, renderFooter(chatFooterActions, contentWidth))

	return strings.Join(lines, "\n")
}

// ========== TREE VIEW ==========
func (m TreeModel) viewTree() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	return m.viewTreePane(w, h)
}

func (m TreeModel) viewTreePane(w, h int) string {
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}

	innerWidth := w - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	visibleDecs := m.decisionItems()
	total := len(visibleDecs)
	answered := 0
	pending := 0
	for _, d := range visibleDecs {
		if d.Answer != nil {
			answered++
		} else {
			pending++
		}
	}

	// Build header status for right side of title bar
	var statusParts []string
	statusParts = append(statusParts, fmt.Sprintf("%d/%d decisions", answered, total))
	if pending > 0 {
		statusParts = append(statusParts, fmt.Sprintf("○ %d pending", pending))
	}
	if m.overallStatus != "" {
		statusParts = append(statusParts, m.overallStatus)
	}
	rightStatus := strings.Join(statusParts, " ── ")

	var lines []string
	lines = append(lines, "")

	// Build flat item list for the tree
	type flatItem struct {
		isCat  bool
		cat    string
		dec    *decision.Decision
		decIdx int
	}

	// Build ID set for dependency lookups
	idSet := make(map[string]bool)
	for _, d := range visibleDecs {
		idSet[d.ID] = true
	}

	var flat []flatItem

	// Flat list — no category grouping (domain shown as tag in each card)
	for i := range visibleDecs {
		flat = append(flat, flatItem{dec: &visibleDecs[i], decIdx: i})
	}

	// Find cursor position in flat list
	cursorFlat := 0
	di := 0
	for i, item := range flat {
		if !item.isCat {
			if di == m.cursor {
				cursorFlat = i
				break
			}
			di++
		}
	}

	// Calculate available tree height:
	// total = h - borders(2) - empty(1) - footer divider(1) - footer(1)
	fixedLines := 2 + 1 + 1 + 1
	// If search bar is visible, it takes a divider + content line
	if m.searchMode || m.searchQuery != "" {
		fixedLines += 2
	}
	// If jump search is active, it takes divider + input + dropdown lines
	if m.jumpSearchMode {
		dropdownLines := len(m.jumpMatches)
		if dropdownLines > 8 {
			dropdownLines = 8
		}
		fixedLines += 2 + dropdownLines // divider + input + matches
	}
	treeH := h - fixedLines
	if treeH < 3 {
		treeH = 3
	}

	// Scrolling
	scrollStart := cursorFlat - treeH/2
	if scrollStart < 0 {
		scrollStart = 0
	}
	if scrollStart+treeH > len(flat) {
		scrollStart = len(flat) - treeH
		if scrollStart < 0 {
			scrollStart = 0
		}
	}

	// Card rendering (same as wide-terminal version)
	cardInner := innerWidth - 4
	if cardInner < 10 {
		cardInner = 10
	}

	// Compute card heights for scroll
	type cardInfo struct {
		dec       *decision.Decision
		decIdx    int
		height    int
		startLine int
	}
	var cards []cardInfo
	totalLines := 0
	for _, item := range flat {
		if item.isCat {
			continue
		}
		ch := 4
		hasTags := item.dec.Category != "" || len(item.dec.Features) > 0
		if hasTags {
			ch = 6
		}
		cards = append(cards, cardInfo{dec: item.dec, decIdx: item.decIdx, height: ch, startLine: totalLines})
		totalLines += ch
	}

	// Scroll to keep cursor visible
	scrollLine := 0
	if m.cursor >= 0 && m.cursor < len(cards) {
		curCard := cards[m.cursor]
		if curCard.startLine+curCard.height > scrollLine+treeH {
			scrollLine = curCard.startLine + curCard.height - treeH
		}
		if curCard.startLine < scrollLine {
			scrollLine = curCard.startLine
		}
	}
	if scrollLine < 0 {
		scrollLine = 0
	}

	// Render cards
	var cardLines []string
	for _, ci := range cards {
		d := ci.dec
		isCur := ci.decIdx == m.cursor
		borderCol := BorderColor
		if isCur {
			borderCol = ActiveBorderColor
		}
		bStyle := lipgloss.NewStyle().Foreground(borderCol)

		padLine := func(content string) string {
			cw := lipgloss.Width(content)
			right := cardInner - cw
			if right < 0 { right = 0 }
			return " " + bStyle.Render("│") + " " + content + strings.Repeat(" ", right) + " " + bStyle.Render("│")
		}

		idLabel := " @" + d.ID + " "
		fillLen := cardInner + 2 - lipgloss.Width(idLabel)
		if fillLen < 0 { fillLen = 0 }
		topLine := " " + bStyle.Render("╭") + bStyle.Render(idLabel) + bStyle.Render(strings.Repeat("─", fillLen)) + bStyle.Render("╮")

		qStr := trunc(d.Question, cardInner-1)
		if isCur { qStr = BoldWhite.Render(qStr) }

		var ansStr string
		if d.Answer != nil {
			ans := trunc(*d.Answer, cardInner-4)
			if isCur {
				ansStr = "  " + GreenStyle.Render(ans)
			} else {
				ansStr = "  " + DimStyle.Render(ans)
			}
		} else {
			ansStr = "  " + YellowStyle.Render("pending")
		}

		var tagStr string
		if d.Category != "" || len(d.Features) > 0 {
			var tags []string
			seen := map[string]bool{}
			if d.Category != "" {
				catKey := strings.ToLower(strings.TrimSpace(d.Category))
				seen[catKey] = true
				tags = append(tags, renderTag(d.Category))
			}
			for _, f := range d.Features {
				fKey := strings.ToLower(strings.TrimSpace(f))
				if seen[fKey] { continue }
				seen[fKey] = true
				t := renderTag(f)
				testLine := strings.Join(append(tags, t), " ")
				if lipgloss.Width(testLine) > cardInner-1 { break }
				tags = append(tags, t)
			}
			tagStr = strings.Join(tags, " ")
		}

		bottomLine := " " + bStyle.Render("╰") + bStyle.Render(strings.Repeat("─", cardInner+2)) + bStyle.Render("╯")

		cardLines = append(cardLines, topLine)
		cardLines = append(cardLines, padLine(qStr))
		cardLines = append(cardLines, padLine(ansStr))
		if tagStr != "" {
			cardLines = append(cardLines, padLine(""))
			cardLines = append(cardLines, padLine(tagStr))
		}
		cardLines = append(cardLines, bottomLine)
	}

	// Slice visible portion
	endLine := scrollLine + treeH
	if endLine > len(cardLines) { endLine = len(cardLines) }
	if scrollLine < len(cardLines) {
		lines = append(lines, cardLines[scrollLine:endLine]...)
	}
	rendered := endLine - scrollLine
	for rendered < treeH {
		lines = append(lines, "")
		rendered++
	}

	// Jump search overlay (Ctrl+F) — shown above search bar
	if m.jumpSearchMode {
		lines = append(lines, buildMiddleBorder(innerWidth))
		lines = append(lines, "  "+m.jumpSearchInput.View())
		// Show matches dropdown (max 8)
		maxDropdown := 8
		if len(m.jumpMatches) < maxDropdown {
			maxDropdown = len(m.jumpMatches)
		}
		for i := 0; i < maxDropdown; i++ {
			jm := m.jumpMatches[i]
			prefix := "    "
			if i == m.jumpCursor {
				prefix = "  " + AccentStyle.Render("> ")
			}
			typeTag := DimStyle.Render("[" + jm.Type + "]")
			label := jm.Label
			if i == m.jumpCursor {
				label = BoldWhite.Render(label)
			}
			lines = append(lines, prefix+typeTag+" "+label)
		}
	}

	// Search bar (shown when search mode is active)
	if m.searchMode {
		lines = append(lines, buildMiddleBorder(innerWidth))
		lines = append(lines, "  "+m.searchInput.View())
	} else if m.searchQuery != "" {
		lines = append(lines, buildMiddleBorder(innerWidth))
		lines = append(lines, "  "+DimStyle.Render(fmt.Sprintf("Filtered: %d results", total)))
	}


	// Footer
	lines = append(lines, buildMiddleBorder(innerWidth))
	var footerActions []footerAction
	if m.jumpSearchMode {
		footerActions = []footerAction{
			{"type", "to find"},
			{"↑↓", "select"},
			{"enter", "jump"},
			{"esc", "close"},
		}
	} else if m.searchMode {
		footerActions = []footerAction{
			{"type", "to filter"},
			{"enter", "confirm"},
			{"esc", "clear"},
		}
	} else {
		footerActions = []footerAction{
			{"↑↓", "navigate"},
			{"enter", "inspect"},
			{"/", "filter"},
			{"s", "sort"},
			{"tab", "chat"},
			{"ctrl+q", "quit"},
		}
	}
	lines = append(lines, renderFooter(footerActions, innerWidth))

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, "", rightStatus)
}

// ========== DETAIL PANE (right side of split view) ==========
func (m TreeModel) viewDetailPane(w, h int) string {
	sel := m.selected()
	if sel == nil {
		return DimStyle.Render("No decision selected.")
	}

	if w < 30 {
		w = 30
	}
	if h < 10 {
		h = 10
	}

	innerWidth := w - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	var lines []string
	lines = append(lines, "")

	// Category + impact
	header := "  " + DimStyle.Render(sel.Category)
	if sel.Impact >= 7 {
		header += "  " + RedStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	} else if sel.Impact >= 4 {
		header += "  " + YellowStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	} else if sel.Impact >= 1 {
		header += "  " + DimStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	}
	if sel.Delegated {
		header += "  " + MagentaStyle.Render("auto")
	}
	lines = append(lines, header)

	// Dependencies
	if len(sel.DependsOn) > 0 {
		depIDs := resolveDepIDs(sel.DependsOn, m.decisions)
		lines = append(lines, "  "+DimStyle.Render("depends on: "+strings.Join(depIDs, ", ")))
	}

	// Reverse dependencies
	revDeps := decision.FindDependents(sel.ID, m.decisions)
	if len(revDeps) > 0 {
		var revIDs []string
		for _, rd := range revDeps {
			revIDs = append(revIDs, rd.ID)
		}
		lines = append(lines, "  "+DimStyle.Render("depended on by: "+strings.Join(revIDs, ", ")))
	}
	lines = append(lines, "")

	// Question (word-wrapped to fit pane)
	qLines := wrapText(sel.Question, innerWidth-4)
	for _, ql := range qLines {
		lines = append(lines, "  "+DetailQuestionStyle.Render(ql))
	}
	if sel.Context != "" {
		ctxLines := wrapText(sel.Context, innerWidth-4)
		for _, cl := range ctxLines {
			lines = append(lines, "  "+DetailContextStyle.Render(cl))
		}
	}
	lines = append(lines, "")

	// Current answer
	if sel.Answer != nil {
		style := GreenStyle
		prefix := "  "
		if sel.Delegated {
			style = MagentaStyle
		}
		ansLines := wrapText(*sel.Answer, innerWidth-6)
		for i, al := range ansLines {
			if i == 0 {
				lines = append(lines, prefix+style.Render(al))
			} else {
				lines = append(lines, "    "+style.Render(al))
			}
		}
	} else {
		lines = append(lines, "  "+YellowStyle.Render("pending"))
	}

	if sel.Reasoning != "" {
		lines = append(lines, "  "+DimStyle.Render(trunc(sel.Reasoning, innerWidth-4)))
	}
	lines = append(lines, "")

	// Options (navigable)
	if len(sel.Options) > 0 {
		for i, opt := range sel.Options {
			isSel := i == m.optCursor
			isChosen := sel.Answer != nil && opt.Label == *sel.Answer
			cur := "    "
			if isSel {
				cur = "  " + AccentStyle.Render("> ")
			}
			style := lipgloss.NewStyle().Foreground(DimGray)
			if isChosen {
				style = GreenStyle
			} else if isSel {
				style = BoldWhite
			}
			optText := trunc(fmt.Sprintf("%s) %s", opt.Key, opt.Label), innerWidth-6)
			lines = append(lines, cur+style.Render(optText))
		}
		lines = append(lines, "")
	}

	// Why text — render as markdown
	if m.whyText != "" && m.whyText != "..." && m.whyText != "Shuffling options..." && m.whyText != "Generating..." {
		if m.mdRenderer != nil {
			if md, err := m.mdRenderer.Render(m.whyText); err == nil {
				for _, ml := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
					visWidth := lipgloss.Width(ml)
					if visWidth > innerWidth-2 {
						for _, wl := range wrapText(ml, innerWidth-2) {
							lines = append(lines, " "+wl)
						}
					} else {
						lines = append(lines, " "+ml)
					}
				}
				lines = append(lines, "")
			} else {
				// Fallback: plain text wrapped
				for _, wl := range wrapText(m.whyText, innerWidth-2) {
					lines = append(lines, " "+DimStyle.Render(wl))
				}
				lines = append(lines, "")
			}
		}
	} else if m.whyText == "..." || m.whyText == "Shuffling options..." || m.whyText == "Generating..." {
		lines = append(lines, " "+AccentStyle.Render(m.whyText))
		lines = append(lines, "")
	}

	// Fill remaining vertical space
	// Account for: borders(2) + footer divider(1) + footer(1)
	fixedLines := 4
	remaining := h - len(lines) - fixedLines
	for i := 0; i < remaining; i++ {
		lines = append(lines, "")
	}

	// Features
	if len(sel.Features) > 0 {
		featureTags := make([]string, len(sel.Features))
		for i, f := range sel.Features {
			featureTags[i] = "#" + f
		}
		lines = append(lines, "  "+AccentStyle.Render(strings.Join(featureTags, " ")))
		lines = append(lines, "")
	}

	// Footer
	lines = append(lines, buildMiddleBorder(innerWidth))
	detailFooterActions := []footerAction{
		{"↑↓", "pick"},
		{"enter", "confirm"},
		{"c", "custom"},
		{"s", "shuffle"},
		{"f", "features"},
		{"w", "why"},
		{"a", "ask"},
		{"q", "back"},
	}
	lines = append(lines, renderFooter(detailFooterActions, innerWidth))

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, sel.ID, sel.Category)
}

// wrapText breaks s into lines of at most width characters, splitting on spaces.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	// Use visible width (ignores ANSI escape codes)
	if lipgloss.Width(s) <= width {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if lipgloss.Width(cur)+1+lipgloss.Width(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	lines = append(lines, cur)
	return lines
}

// renderDomainPills builds a compact status line like "Stack: executing  Data: planning  UI: done"
// with color coding per status.
func (m TreeModel) renderDomainPills(maxWidth int) string {
	if len(m.domainStatuses) == 0 {
		return ""
	}

	// Collect and sort domain names for stable ordering
	var domains []string
	for d := range m.domainStatuses {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	var pills []string
	for _, domain := range domains {
		status := m.domainStatuses[domain]
		var styled string
		switch status {
		case "planning":
			styled = YellowStyle.Render(domain + ": " + status)
		case "executing":
			styled = AccentStyle.Render(domain + ": " + status)
		case "verifying":
			styled = BlueStyle.Render(domain + ": " + status)
		case "done":
			styled = GreenStyle.Render(domain + ": " + status)
		case "error":
			styled = RedStyle.Render(domain + ": " + status)
		default:
			styled = DimStyle.Render(domain + ": " + status)
		}
		pills = append(pills, styled)
	}

	result := strings.Join(pills, "  ")
	// Truncate if too wide (use visible width)
	if lipgloss.Width(result) > maxWidth && maxWidth > 0 {
		// Rebuild with truncated domain names
		var shortPills []string
		for _, domain := range domains {
			status := m.domainStatuses[domain]
			short := trunc(domain, 8)
			var styled string
			switch status {
			case "planning":
				styled = YellowStyle.Render(short + ": " + status)
			case "executing":
				styled = AccentStyle.Render(short + ": " + status)
			case "verifying":
				styled = BlueStyle.Render(short + ": " + status)
			case "done":
				styled = GreenStyle.Render(short + ": " + status)
			case "error":
				styled = RedStyle.Render(short + ": " + status)
			default:
				styled = DimStyle.Render(short + ": " + status)
			}
			shortPills = append(shortPills, styled)
		}
		result = strings.Join(shortPills, "  ")
	}
	return result
}

// ========== DETAIL VIEW (full-screen) ==========
func (m TreeModel) viewDetail() string {
	sel := m.selected()
	if sel == nil {
		return DimStyle.Render("No decision selected.")
	}

	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	innerWidth := w - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	var lines []string
	lines = append(lines, "")

	// Header: ID + category + tags + impact
	header := "  " + DetailTitleStyle.Render(sel.ID) + "  " + DimStyle.Render(sel.Category)
	if sel.Impact >= 7 {
		header += "  " + RedStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	} else if sel.Impact >= 4 {
		header += "  " + YellowStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	} else if sel.Impact >= 1 {
		header += "  " + DimStyle.Render(fmt.Sprintf("impact %d/10", sel.Impact))
	}
	if sel.Delegated {
		header += "  " + MagentaStyle.Render("auto-decided")
	} else if sel.Implicit {
		header += "  " + DimStyle.Render("implicit")
	}
	lines = append(lines, header)

	// Dependencies
	if len(sel.DependsOn) > 0 {
		depIDs := resolveDepIDs(sel.DependsOn, m.decisions)
		deps := "  " + DimStyle.Render("depends on: "+strings.Join(depIDs, ", "))
		lines = append(lines, deps)
	}

	// Reverse dependencies (what depends on THIS decision)
	revDeps := decision.FindDependents(sel.ID, m.decisions)
	if len(revDeps) > 0 {
		var revIDs []string
		for _, rd := range revDeps {
			revIDs = append(revIDs, rd.ID)
		}
		lines = append(lines, "  "+DimStyle.Render("depended on by: "+strings.Join(revIDs, ", ")))
	}
	lines = append(lines, "")

	// Question + context
	lines = append(lines, "  "+DetailQuestionStyle.Render(sel.Question))
	if sel.Context != "" {
		lines = append(lines, "  "+DetailContextStyle.Render(sel.Context))
	}
	lines = append(lines, "")

	// Current answer
	if sel.Answer != nil {
		style := GreenStyle
		prefix := "✓ "
		if sel.Delegated {
			style = MagentaStyle
			prefix = "◆ "
		} else if sel.Implicit {
			style = DimStyle
			prefix = "▪ "
		}
		lines = append(lines, "  "+style.Render(prefix+*sel.Answer))
	} else {
		lines = append(lines, "  "+YellowStyle.Render("○ pending"))
	}

	if sel.Reasoning != "" {
		lines = append(lines, "  "+DimStyle.Render(sel.Reasoning))
	}
	lines = append(lines, "")

	// Options (navigable for ALL decisions, not just pending)
	if len(sel.Options) > 0 {
		label := "Pick / change:"
		if sel.Answer != nil && !sel.IsPending() {
			label = "Change to:"
		}
		lines = append(lines, "  "+DimStyle.Render(label))
		for i, opt := range sel.Options {
			isSel := i == m.optCursor
			isChosen := sel.Answer != nil && opt.Label == *sel.Answer
			cur := "    "
			if isSel {
				cur = "  " + AccentStyle.Render("> ")
			}
			style := lipgloss.NewStyle().Foreground(DimGray)
			if isChosen {
				style = GreenStyle
			} else if isSel {
				style = BoldWhite
			}
			lines = append(lines, cur+style.Render(fmt.Sprintf("%s) %s", opt.Key, opt.Label)))
		}
		lines = append(lines, "")
	}

	// Why text — render as markdown
	if m.whyText != "" && m.whyText != "..." && m.whyText != "Shuffling options..." && m.whyText != "Generating..." {
		if m.mdRenderer != nil {
			if md, err := m.mdRenderer.Render(m.whyText); err == nil {
				for _, ml := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
					visWidth := lipgloss.Width(ml)
					if visWidth > innerWidth-2 {
						for _, wl := range wrapText(ml, innerWidth-2) {
							lines = append(lines, " "+wl)
						}
					} else {
						lines = append(lines, " "+ml)
					}
				}
			} else {
				for _, wl := range wrapText(m.whyText, innerWidth-2) {
					lines = append(lines, " "+DimStyle.Render(wl))
				}
			}
		}
		lines = append(lines, "")
	} else if m.whyText == "..." || m.whyText == "Shuffling options..." || m.whyText == "Generating..." {
		lines = append(lines, " "+AccentStyle.Render(m.whyText))
		lines = append(lines, "")
	}

	// Features
	if len(sel.Features) > 0 {
		featureTags := make([]string, len(sel.Features))
		for i, f := range sel.Features {
			featureTags[i] = "#" + f
		}
		lines = append(lines, "  "+AccentStyle.Render(strings.Join(featureTags, " ")))
		lines = append(lines, "")
	}

	// Text input field
	if m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		label := "Override:"
		if m.mode == tmAsk {
			label = "Ask:"
		} else if m.mode == tmEditFeatures {
			label = "Features:"
		}
		lines = append(lines, "  "+AccentStyle.Render(label))
		lines = append(lines, "  "+m.textInput.View())
		lines = append(lines, "")
	}

	// Fill remaining vertical space
	usedLines := len(lines) + 4 // divider + footer + borders
	remaining := h - usedLines - 4
	for i := 0; i < remaining; i++ {
		lines = append(lines, "")
	}

	// Footer divider + keybindings
	lines = append(lines, buildMiddleBorder(innerWidth))
	var detailFooterActions []footerAction
	if m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		detailFooterActions = []footerAction{
			{"enter", "submit"},
			{"esc", "cancel"},
		}
	} else {
		detailFooterActions = []footerAction{
			{"↑↓", "pick"},
			{"enter", "confirm"},
			{"c", "custom"},
			{"s", "shuffle"},
			{"f", "features"},
			{"w", "why"},
			{"a", "ask"},
			{"q", "back"},
		}
	}
	lines = append(lines, renderFooter(detailFooterActions, innerWidth))

	// Build title for detail border
	detailTitle := sel.ID

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, detailTitle, sel.Category)
}

// parseFeatureTags parses a comma-separated string into deduplicated, lowercase feature tags.
func parseFeatureTags(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := map[string]bool{}
	var result []string
	for _, p := range parts {
		tag := strings.ToLower(strings.TrimSpace(p))
		// Strip leading # if present
		tag = strings.TrimPrefix(tag, "#")
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	return result
}

// computeJumpMatches returns up to 8 matches for the jump search query.
// Matches decisions (by ID, question, category) and features.
func (m TreeModel) computeJumpMatches(query string) []jumpMatch {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var matches []jumpMatch
	decs := m.decisionItems()

	// Match decisions
	for i, d := range decs {
		if strings.Contains(strings.ToLower(d.ID), q) ||
			strings.Contains(strings.ToLower(d.Question), q) ||
			strings.Contains(strings.ToLower(d.Category), q) {
			matches = append(matches, jumpMatch{
				Type:  "decision",
				Label: d.ID + " " + trunc(d.Question, 50),
				Index: i,
			})
		}
		if len(matches) >= 8 {
			return matches
		}
	}

	// Match categories — jump to first decision in category
	catSeen := map[string]bool{}
	for i, d := range decs {
		catLower := strings.ToLower(d.Category)
		if !catSeen[catLower] && strings.Contains(catLower, q) {
			catSeen[catLower] = true
			matches = append(matches, jumpMatch{
				Type:  "category",
				Label: d.Category,
				Index: i,
			})
		}
		if len(matches) >= 8 {
			return matches
		}
	}

	// Match features — jump to first decision with that feature
	featSeen := map[string]bool{}
	for i, d := range decs {
		for _, f := range d.Features {
			fl := strings.ToLower(f)
			if !featSeen[fl] && strings.Contains(fl, q) {
				featSeen[fl] = true
				matches = append(matches, jumpMatch{
					Type:  "feature",
					Label: "#" + f,
					Index: i,
				})
			}
			if len(matches) >= 8 {
				return matches
			}
		}
	}

	return matches
}
