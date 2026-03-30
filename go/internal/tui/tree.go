package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

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
	tmFeed // live agent feed
)

// ChatEntry is a line in the conversation panel.
type ChatEntry struct {
	Type string // "tool", "agent", "user", "system"
	Text string
}

// TreeModel is the main decision tree view.
type TreeModel struct {
	decisions      []decision.Decision
	execStates     []agent.ExecState
	overallStatus  string
	cursor         int
	optCursor      int
	mode           treeMode
	textBuf        string
	whyText        string
	width, height  int
	mascotTick     int
	feedLines      []string    // legacy feed (for tab view)
	chatLog        []ChatEntry // conversation panel
	chatInput      string      // current chat input
	chatFocused    bool        // true = keys go to chat, false = keys go to tree
	chatThinking   bool        // true while waiting for agent response
	chatThinkStart time.Time   // when thinking started
	mdRenderer     *glamour.TermRenderer
}

func NewTreeModel() TreeModel {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // we handle wrapping ourselves
	)
	return TreeModel{mode: tmTree, mdRenderer: r}
}

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m TreeModel) handleKey(msg tea.KeyMsg) (TreeModel, tea.Cmd) {
	key := msg.String()

	// Global: tab toggles chat focus (in tree mode)
	if key == "tab" && m.mode == tmTree {
		m.chatFocused = !m.chatFocused
		return m, nil
	}

	// --- Chat input mode ---
	if m.chatFocused && m.mode == tmTree {
		switch key {
		case "esc":
			m.chatFocused = false
			m.chatInput = ""
			return m, nil
		case "enter":
			if strings.TrimSpace(m.chatInput) != "" {
				input := strings.TrimSpace(m.chatInput)
				m.chatLog = append(m.chatLog, ChatEntry{Type: "user", Text: input})
				m.chatInput = ""
				m.chatThinking = true
				m.chatThinkStart = time.Now()

				// Check for @DECISION-ID references
				// Parse: "@STACK-001 change to Go" → ReviseDecisionMsg
				// Parse: "@STACK-001 why?" → WhyDecisionMsg
				// Otherwise: general chat message
				return m, func() tea.Msg { return ChatMessageMsg{Text: input} }
			}
			return m, nil
		case "backspace":
			if len(m.chatInput) > 0 {
				m.chatInput = m.chatInput[:len(m.chatInput)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.chatInput += key
			}
			return m, nil
		}
	}

	// --- Feed mode (legacy, still accessible) ---
	if m.mode == tmFeed {
		if key == "tab" {
			m.mode = tmTree
			return m, nil
		}
		return m, nil
	}

	// --- Text input ---
	if m.mode == tmRevise || m.mode == tmAsk {
		switch key {
		case "esc":
			m.mode = tmDetail
			m.textBuf = ""
			return m, nil
		case "enter":
			if strings.TrimSpace(m.textBuf) != "" {
				sel := m.selected()
				if sel != nil {
					if m.mode == tmRevise {
						m.mode = tmTree
						id := sel.ID
						answer := strings.TrimSpace(m.textBuf)
						m.textBuf = ""
						return m, func() tea.Msg { return ReviseDecisionMsg{ID: id, NewAnswer: answer} }
					} else if m.mode == tmAsk {
						m.mode = tmDetail
						id := sel.ID
						q := strings.TrimSpace(m.textBuf)
						m.textBuf = ""
						m.whyText = "..."
						return m, func() tea.Msg { return AskDecisionMsg{ID: id, Question: q} }
					}
				}
			}
			return m, nil
		case "backspace":
			if len(m.textBuf) > 0 {
				m.textBuf = m.textBuf[:len(m.textBuf)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.textBuf += key
			}
			return m, nil
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
			m.textBuf = ""
			return m, nil
		case "a":
			m.mode = tmAsk
			m.textBuf = ""
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

	// --- Tree ---
	decCount := m.decisionCount()
	switch key {
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

// highlightDecisionRefs highlights @ID patterns (e.g. @STACK-001) in text using AccentStyle.
func highlightDecisionRefs(text string) string {
	result := text
	words := strings.Fields(text)
	for _, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			ref := word
			highlighted := AccentStyle.Render(ref)
			result = strings.Replace(result, ref, highlighted, 1)
		}
	}
	return result
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

func (m TreeModel) View() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	if m.mode == tmFeed {
		return m.viewFeed()
	}
	if m.mode == tmDetail || m.mode == tmRevise || m.mode == tmAsk {
		return m.viewDetail()
	}
	return m.viewTree()
}

// ========== FEED VIEW ==========
func (m TreeModel) viewFeed() string {
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

	// Feed content area
	feedH := h - 8 // top border + empty line + divider + footer + bottom border + buffer
	if feedH < 3 {
		feedH = 3
	}

	start := 0
	if len(m.feedLines) > feedH {
		start = len(m.feedLines) - feedH
	}
	visible := m.feedLines[start:]

	for _, line := range visible {
		lines = append(lines, "  "+DimStyle.Render(trunc(line, innerWidth-4)))
	}
	for i := len(visible); i < feedH; i++ {
		lines = append(lines, "")
	}

	// Divider + footer
	lines = append(lines, buildMiddleBorder(innerWidth))
	footer := "  " + AccentStyle.Render("tab") + DimStyle.Render(" back to tree")
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, "Live Agent Feed", "")
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
	sortDecisionsByCategory(visibleDecs)

	var flat []flatItem
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

	// Conversation panel height: scales with terminal height
	chatPanelH := h / 4 // 25% of terminal for conversation
	if chatPanelH < 5 {
		chatPanelH = 5
	}
	if chatPanelH > 12 {
		chatPanelH = 12
	}
	activityLines := chatPanelH // used by conversation panel renderer

	// Calculate available tree height:
	// total = h - borders(2) - empty(1) - chat divider(1) - chat(chatPanelH) - footer divider(1) - footer(1) - padding(1)
	fixedLines := 2 + 1 + 1 + chatPanelH + 1 + 1 + 1
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

		// Impact indicator: ▰▰▰ for high impact decisions
		impactStr := ""
		if d.Impact >= 7 {
			impactStr = RedStyle.Render("▰▰▰") + " "
		} else if d.Impact >= 4 {
			impactStr = YellowStyle.Render("▰▰") + " "
		} else if d.Impact >= 1 {
			impactStr = DimStyle.Render("▰") + " "
		}

		idStr := pad(d.ID, idW)
		qStr := trunc(d.Question, qW)

		var row string
		if isCur {
			row = fmt.Sprintf("  %s%s %s%s %s  %s",
				cursor,
				iconStyle.Render(icon),
				impactStr,
				BoldWhite.Render(idStr),
				BoldWhite.Render(qStr),
				DimStyle.Render(answer),
			)
		} else {
			row = fmt.Sprintf("  %s%s %s%s %s  %s",
				cursor,
				iconStyle.Render(icon),
				impactStr,
				idStr,
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

	// Conversation panel divider
	lines = append(lines, buildMiddleBorder(innerWidth))

	// Conversation: show last N chat entries + input
	chatH := activityLines + 2 // activity lines + input + padding
	if chatH < 4 {
		chatH = 4
	}

	// Render chat entries
	chatRendered := 0
	if len(m.chatLog) > 0 {
		start := len(m.chatLog) - (chatH - 1)
		if start < 0 {
			start = 0
		}
		for _, entry := range m.chatLog[start:] {
			if chatRendered >= chatH-1 {
				break
			}
			switch entry.Type {
			case "tool":
				lines = append(lines, "  "+DimStyle.Render("  "+entry.Text))
				chatRendered++
			case "agent":
				// Render markdown for agent responses
				rendered := entry.Text
				if m.mdRenderer != nil {
					if md, err := m.mdRenderer.Render(entry.Text); err == nil {
						// Glamour adds newlines, take first few lines that fit
						mdLines := strings.Split(strings.TrimRight(md, "\n"), "\n")
						for _, ml := range mdLines {
							if chatRendered >= chatH-1 {
								break
							}
							lines = append(lines, "  "+ml)
							chatRendered++
						}
						continue
					}
				}
				lines = append(lines, "  "+AccentStyle.Render("● ")+rendered)
				chatRendered++
			case "user":
				lines = append(lines, "  "+BoldWhite.Render("> ")+highlightDecisionRefs(entry.Text))
				chatRendered++
			default:
				lines = append(lines, "  "+DimStyle.Render(entry.Text))
				chatRendered++
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
		lines = append(lines, "  "+AccentStyle.Render("● Thinking... ")+DimStyle.Render("("+timeStr+")"))
		chatRendered++
	}

	// Fill remaining chat space
	for chatRendered < chatH-1 {
		if chatRendered == 0 {
			lines = append(lines, "  "+DimStyle.Render("Press tab to chat, @ID to reference a decision"))
		} else {
			lines = append(lines, "")
		}
		chatRendered++
	}

	// Chat input line
	if m.chatFocused {
		inputLine := "  " + AccentStyle.Render("> ") + m.chatInput + AccentStyle.Render("▎")
		lines = append(lines, inputLine)
	} else {
		lines = append(lines, "  "+DimStyle.Render("tab to chat..."))
	}

	// Footer divider + keybindings
	lines = append(lines, buildMiddleBorder(innerWidth))
	var footer string
	if m.chatFocused {
		footer = "  " + AccentStyle.Render("enter") + DimStyle.Render(" send  ") +
			AccentStyle.Render("@ID") + DimStyle.Render(" reference  ") +
			AccentStyle.Render("esc") + DimStyle.Render(" back to tree  ") +
			DimStyle.Render("ctrl+c×2 quit")
	} else {
		footer = "  " + AccentStyle.Render("↑↓") + DimStyle.Render(" navigate  ") +
			AccentStyle.Render("enter") + DimStyle.Render(" inspect  ") +
			AccentStyle.Render("tab") + DimStyle.Render(" chat  ") +
			DimStyle.Render("ctrl+c×2 quit")
	}
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, "defer", rightStatus)
}

// ========== DETAIL VIEW ==========
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
		deps := "  " + DimStyle.Render("depends on: "+strings.Join(sel.DependsOn, ", "))
		lines = append(lines, deps)
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

	// Why text / loading indicator
	if m.whyText != "" && m.whyText != "..." && m.whyText != "Shuffling options..." && m.whyText != "Generating..." {
		lines = append(lines, "  "+DimStyle.Render(trunc(m.whyText, 500)))
		lines = append(lines, "")
	} else if m.whyText == "..." || m.whyText == "Shuffling options..." || m.whyText == "Generating..." {
		lines = append(lines, "  "+AccentStyle.Render(m.whyText))
		lines = append(lines, "")
	}

	// Text input field
	if m.mode == tmRevise || m.mode == tmAsk {
		label := "Override:"
		if m.mode == tmAsk {
			label = "Ask:"
		}
		lines = append(lines, "  "+AccentStyle.Render(label))
		lines = append(lines, "  "+AccentStyle.Render("> ")+m.textBuf+AccentStyle.Render("_"))
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
	if m.mode == tmRevise || m.mode == tmAsk {
		footer := "  " + AccentStyle.Render("enter") + DimStyle.Render(" submit  ") +
			AccentStyle.Render("esc") + DimStyle.Render(" cancel")
		lines = append(lines, footer)
	} else {
		footer := "  " + AccentStyle.Render("↑↓") + DimStyle.Render(" pick  ") +
			AccentStyle.Render("enter") + DimStyle.Render(" confirm  ") +
			AccentStyle.Render("c") + DimStyle.Render(" custom  ") +
			AccentStyle.Render("s") + DimStyle.Render(" shuffle  ") +
			AccentStyle.Render("w") + DimStyle.Render(" why  ") +
			AccentStyle.Render("a") + DimStyle.Render(" ask  ") +
			AccentStyle.Render("q") + DimStyle.Render(" back")
		lines = append(lines, footer)
	}

	// Build title for detail border
	detailTitle := sel.ID

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, detailTitle, sel.Category)
}
