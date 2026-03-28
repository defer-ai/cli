package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	feedLines      []string // live agent output
}

func NewTreeModel() TreeModel {
	return TreeModel{mode: tmTree}
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

	// Global: tab switches to feed, shift+tab toggles permissions
	if key == "tab" {
		if m.mode == tmFeed {
			m.mode = tmTree
		} else {
			m.mode = tmFeed
		}
		return m, nil
	}
	// shift+tab reserved for future use

	// --- Feed mode ---
	if m.mode == tmFeed {
		// Only tab (handled above) and ctrl+c exit feed
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
	h := m.height

	var b strings.Builder
	b.WriteString("\n  " + BoldAccent.Render("Live Agent Feed") + "\n")
	b.WriteString("  " + Separator(w-4) + "\n")

	feedH := h - 5
	start := 0
	if len(m.feedLines) > feedH {
		start = len(m.feedLines) - feedH
	}
	visible := m.feedLines[start:]

	for _, line := range visible {
		b.WriteString("  " + DimStyle.Render(trunc(line, w-4)) + "\n")
	}
	for i := len(visible); i < feedH; i++ {
		b.WriteString("\n")
	}

	b.WriteString("  " + Separator(w-4) + "\n")
	b.WriteString("  " + AccentStyle.Render("tab") + DimStyle.Render(" back to tree"))

	return b.String()
}

// ========== TREE VIEW ==========
func (m TreeModel) viewTree() string {
	w := m.width
	h := m.height

	var b strings.Builder

	mascot := RenderMascot(StatusToMood(m.overallStatus), m.mascotTick)
	mascotLines := strings.Split(mascot, "\n")

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

	stats := BoldAccent.Render("defer") + "\n"
	stats += DimStyle.Render(fmt.Sprintf("%d/%d decisions", answered, total))
	if pending > 0 {
		stats += YellowStyle.Render(fmt.Sprintf("  ○ %d pending", pending))
	}
	statsLines := strings.Split(stats, "\n")

	mascotW := 36
	maxLines := len(mascotLines)
	if len(statsLines) > maxLines {
		maxLines = len(statsLines)
	}
	for i := 0; i < maxLines; i++ {
		ml := ""
		if i < len(mascotLines) {
			ml = mascotLines[i]
		}
		sl := ""
		if i < len(statsLines) {
			sl = statsLines[i]
		}
		b.WriteString(fmt.Sprintf("  %-*s  %s\n", mascotW, ml, sl))
	}

	b.WriteString("  " + Separator(w-4) + "\n")

	headerLines := maxLines + 1
	footerLines := 2
	treeH := h - headerLines - footerLines
	if treeH < 3 {
		treeH = 3
	}

	type flatItem struct {
		isCat  bool
		cat    string
		dec    *decision.Decision
		decIdx int
	}
	// Sort decisions by category (preserving original category order)
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
	ansW := 30
	qW := w - idW - ansW - 12
	if qW < 10 {
		qW = 10
	}

	rendered := 0
	for i := scrollStart; i < len(flat) && rendered < treeH; i++ {
		item := flat[i]
		if item.isCat {
			if item.cat == "" {
				b.WriteString("\n")
			} else {
				b.WriteString("  " + BoldAccent.Render(item.cat) + "\n")
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
				// auto-decided, delegated, implicit, agent -- all gray
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
		}

		idStr := pad(d.ID, idW)
		qStr := trunc(d.Question, qW)

		if isCur {
			b.WriteString(fmt.Sprintf("  %s%s %s %s  %s\n",
				cursor,
				iconStyle.Render(icon),
				BoldWhite.Render(idStr),
				BoldWhite.Render(qStr),
				DimStyle.Render(answer),
			))
		} else {
			b.WriteString(fmt.Sprintf("  %s%s %s %s  %s\n",
				cursor,
				iconStyle.Render(icon),
				idStr,
				qStr,
				DimStyle.Render(answer),
			))
		}
		rendered++
	}

	for rendered < treeH {
		b.WriteString("\n")
		rendered++
	}

	b.WriteString("  " + Separator(w-4) + "\n")
	b.WriteString("  " + AccentStyle.Render("↑↓") + DimStyle.Render(" navigate  "))
	b.WriteString(AccentStyle.Render("enter") + DimStyle.Render(" inspect  "))
	b.WriteString(AccentStyle.Render("tab") + DimStyle.Render(" feed  "))
	b.WriteString(DimStyle.Render("ctrl+c×2 quit"))

	return b.String()
}

// ========== DETAIL VIEW ==========
func (m TreeModel) viewDetail() string {
	sel := m.selected()
	if sel == nil {
		return DimStyle.Render("No decision selected.")
	}

	w := m.width
	var b strings.Builder

	b.WriteString("  " + BoldAccent.Render(sel.ID) + "  " + DimStyle.Render(sel.Category))
	if sel.Delegated {
		b.WriteString(MagentaStyle.Render("  auto-decided"))
	} else if sel.Implicit {
		b.WriteString(DimStyle.Render("  implicit"))
	}
	b.WriteString("\n\n")

	b.WriteString("  " + BoldWhite.Render(sel.Question) + "\n")
	if sel.Context != "" {
		b.WriteString("  " + DimStyle.Render(sel.Context) + "\n")
	}
	b.WriteString("\n")

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
		b.WriteString("  " + style.Render(prefix+*sel.Answer) + "\n")
	} else {
		b.WriteString("  " + YellowStyle.Render("○ pending") + "\n")
	}

	if sel.Reasoning != "" {
		b.WriteString("  " + DimStyle.Render(sel.Reasoning) + "\n")
	}
	b.WriteString("\n")

	// Options (navigable for ALL decisions, not just pending)
	if len(sel.Options) > 0 {
		label := "Pick / change:"
		if sel.Answer != nil && !sel.IsPending() {
			label = "Change to:"
		}
		b.WriteString("  " + DimStyle.Render(label) + "\n")
		for i, opt := range sel.Options {
			isSel := i == m.optCursor
			isChosen := sel.Answer != nil && opt.Label == *sel.Answer
			cur := "    "
			if isSel {
				cur = "  " + AccentStyle.Render("> ")
			}
			style := DimStyle
			if isChosen {
				style = GreenStyle
			} else if isSel {
				style = BoldWhite
			}
			b.WriteString(cur + style.Render(fmt.Sprintf("%s) %s", opt.Key, opt.Label)) + "\n")
		}
		b.WriteString("\n")
	}

	if m.whyText != "" && m.whyText != "..." && m.whyText != "Shuffling options..." && m.whyText != "Generating..." {
		b.WriteString("  " + DimStyle.Render(trunc(m.whyText, 500)) + "\n\n")
	} else if m.whyText == "..." || m.whyText == "Shuffling options..." || m.whyText == "Generating..." {
		b.WriteString("  " + AccentStyle.Render(m.whyText) + "\n\n")
	}

	if m.mode == tmRevise || m.mode == tmAsk {
		label := "Override:"
		if m.mode == tmAsk {
			label = "Ask:"
		}
		b.WriteString("  " + AccentStyle.Render(label) + "\n")
		b.WriteString("  " + AccentStyle.Render("> ") + m.textBuf + AccentStyle.Render("_") + "\n\n")
	}

	b.WriteString("  " + Separator(w-4) + "\n")
	if m.mode == tmRevise || m.mode == tmAsk {
		b.WriteString("  " + AccentStyle.Render("enter") + DimStyle.Render(" submit  "))
		b.WriteString(AccentStyle.Render("esc") + DimStyle.Render(" cancel"))
	} else {
		b.WriteString("  " + AccentStyle.Render("↑↓") + DimStyle.Render(" pick  "))
		b.WriteString(AccentStyle.Render("enter") + DimStyle.Render(" confirm  "))
		b.WriteString(AccentStyle.Render("c") + DimStyle.Render(" custom  "))
		b.WriteString(AccentStyle.Render("s") + DimStyle.Render(" shuffle  "))
		b.WriteString(AccentStyle.Render("w") + DimStyle.Render(" why  "))
		b.WriteString(AccentStyle.Render("a") + DimStyle.Render(" ask  "))
		b.WriteString(AccentStyle.Render("q") + DimStyle.Render(" back"))
	}

	return b.String()
}
