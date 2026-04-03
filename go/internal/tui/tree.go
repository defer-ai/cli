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
	tmChat         // full-screen chat
	tmEditFeatures // editing feature tags on a decision
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
	groupByFeature bool            // true = group tree by feature tag, false = group by category (default)
	// Jump search (Ctrl+F) — find and jump without filtering
	jumpSearchMode  bool
	jumpSearchInput textinput.Model
	jumpMatches     []jumpMatch
	jumpCursor      int
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

	return TreeModel{mode: tmTree, mdRenderer: r, chatInput: ci, textInput: ti, searchInput: si, jumpSearchInput: ji, completionIdx: -1}
}

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

	// Tab toggles between tree and chat,
	// unless we're in chat with active completions (tab-complete).
	if key == "tab" && !(m.mode == tmChat && len(m.completions) > 0) {
		if m.mode == tmChat {
			m.mode = tmTree
			m.chatFocused = false
			m.chatInput.Blur()
		} else if m.mode == tmTree {
			m.mode = tmChat
			m.chatFocused = true
			m.chatInput.Focus()
		}
		return m, nil
	}

	// --- Chat input mode (full screen chat via tmChat) ---
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
			case "j", "down":
				if m.optCursor < len(sel.Options)-1 {
					m.optCursor++
				}
				return m, nil
			case "k", "up":
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
	case "g":
		m.groupByFeature = !m.groupByFeature
		m.cursor = 0
		return m, nil
	case "j", "down":
		if m.cursor < decCount-1 {
			m.cursor++
		}
		return m, nil
	case "k", "up":
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
	sortDecisionsByCategory(items)
	return items
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

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
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
	if m.mode == tmChat {
		return m.viewChat()
	}

	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	// Detail, revise, ask, edit-features — all full-screen, replacing the tree
	if m.mode == tmDetail || m.mode == tmRevise || m.mode == tmAsk || m.mode == tmEditFeatures {
		return m.viewDetail()
	}

	return m.viewTree()
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
			// Claude Code style: ● BoldWhite(topic text)
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, " "+AccentStyle.Render("●")+" "+BoldWhite.Render(label))
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
			// Standalone tool: ● AccentStyle(icon + text)
			label := toolCallLabel(entry.Text)
			chatLines = append(chatLines, " "+AccentStyle.Render("● "+label))

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
	if m.groupByFeature {
		statusParts = append(statusParts, "by feature")
	} else {
		statusParts = append(statusParts, "by domain")
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

	if m.groupByFeature {
		// Group by feature tags
		featureMap := map[string][]int{} // feature name → decision indices (into visibleDecs)
		var featureOrder []string
		featureSeen := map[string]bool{}
		for i, d := range visibleDecs {
			if len(d.Features) == 0 {
				if !featureSeen["untagged"] {
					featureOrder = append(featureOrder, "untagged")
					featureSeen["untagged"] = true
				}
				featureMap["untagged"] = append(featureMap["untagged"], i)
			}
			for _, f := range d.Features {
				fl := strings.ToLower(strings.TrimSpace(f))
				if !featureSeen[fl] {
					featureOrder = append(featureOrder, fl)
					featureSeen[fl] = true
				}
				featureMap[fl] = append(featureMap[fl], i)
			}
		}
		sort.Strings(featureOrder)
		// Build flat items grouped by feature
		decIdx := 0
		// We need a global index mapping: visibleDecs index → decIdx
		// Since feature grouping can show same decision under multiple features,
		// we use the original visibleDecs order for cursor mapping.
		// Build a mapping from visibleDecs index to decIdx (they are 1:1)
		for _, feat := range featureOrder {
			indices := featureMap[feat]
			label := "#" + feat
			if feat == "untagged" {
				label = "(untagged)"
			}
			if len(flat) > 0 {
				flat = append(flat, flatItem{isCat: true, cat: ""})
			}
			flat = append(flat, flatItem{isCat: true, cat: label})
			for _, idx := range indices {
				flat = append(flat, flatItem{dec: &visibleDecs[idx], decIdx: idx})
			}
		}
		_ = decIdx
	} else {
		// Default: group by category
		sortDecisionsByCategory(visibleDecs)
		lastCat := ""
		decIdx := 0
		for i := range visibleDecs {
			d := &visibleDecs[i]
			catKey := strings.ToLower(strings.TrimSpace(d.Category))
			lastKey := strings.ToLower(strings.TrimSpace(lastCat))
			if catKey != lastKey {
				if lastCat != "" {
					flat = append(flat, flatItem{isCat: true, cat: ""})
				}
				flat = append(flat, flatItem{isCat: true, cat: d.Category})
				lastCat = d.Category
			}
			flat = append(flat, flatItem{dec: d, decIdx: decIdx})
			decIdx++
		}
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

	idW := 12
	ansW := (innerWidth - idW - 14) / 2 // split remaining space between question and answer
	qW := innerWidth - idW - ansW - 10
	if qW < 10 {
		qW = 10
	}
	if ansW < 10 {
		ansW = 10
	}

	rendered := 0
	for i := scrollStart; i < len(flat) && rendered < treeH; i++ {
		item := flat[i]
		if item.isCat {
			if item.cat == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, "  "+CategoryStyle.Render(item.cat))
			}
			rendered++
			continue
		}

		d := item.dec
		isCur := item.decIdx == m.cursor
		icon := "○"
		iconStyle := YellowStyle
		if d.Answer != nil {
			if d.Source == "user" {
				icon = "✓"
				iconStyle = GreenStyle
			} else {
				icon = "▪"
				iconStyle = DimStyle
			}
		}

		cursor := "  "
		if isCur {
			cursor = AccentStyle.Render("> ")
		}

		answer := ""
		if d.Answer != nil {
			answer = "→ " + trunc(*d.Answer, ansW)
		} else {
			answer = DimStyle.Render("(pending)")
		}

		// Color the ID based on impact level
		idStr := pad(d.ID, idW)
		qStr := trunc(d.Question, qW)

		var idStyle lipgloss.Style
		if d.Impact >= 7 {
			idStyle = RedStyle
		} else if d.Impact >= 4 {
			idStyle = YellowStyle
		} else if d.Impact >= 1 {
			idStyle = DimStyle
		} else {
			idStyle = lipgloss.NewStyle()
		}

		var row string
		if isCur {
			curIDStyle := idStyle.Bold(true)
			row = fmt.Sprintf(" %s%s %s %s  %s",
				cursor,
				iconStyle.Render(icon),
				curIDStyle.Render(idStr),
				BoldWhite.Render(qStr),
				DimStyle.Render(answer),
			)
		} else {
			row = fmt.Sprintf(" %s%s %s %s  %s",
				cursor,
				iconStyle.Render(icon),
				idStyle.Render(idStr),
				qStr,
				DimStyle.Render(answer),
			)
		}
		lines = append(lines, row)
		rendered++
	}

	// Fill remaining tree space
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
			{"j/k", "select"},
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
			{"j/k", "navigate"},
			{"enter", "inspect"},
			{"/", "filter"},
			{"f", "find"},
			{"g", "group"},
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
		{"j/k", "pick"},
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
			{"j/k", "pick"},
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
