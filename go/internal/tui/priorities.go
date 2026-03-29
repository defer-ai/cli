package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

var careLevels = []struct {
	Key   agent.CareLevel
	Label string
	Color lipgloss.Color
	Desc  string
	Bar   string
}{
	{agent.CareLevelSkip, "skip", DimGray, "delegate everything", "░░░░░"},
	{agent.CareLevelLow, "low", lipgloss.Color("4"), "only key question", "█░░░░"},
	{agent.CareLevelMedium, "medium", lipgloss.Color("3"), "important decisions", "██░░░"},
	{agent.CareLevelHigh, "high", lipgloss.Color("2"), "ask me everything", "████░"},
	{agent.CareLevelParanoid, "paranoid", lipgloss.Color("1"), "deep dive", "█████"},
}

// PrioritiesModel lets the user set care levels per domain.
type PrioritiesModel struct {
	categories    []string
	priorities    map[string]agent.CareLevel
	counts        map[string]int
	cursor        int
	width, height int
}

func NewPrioritiesModel(decisions []decision.Decision) PrioritiesModel {
	cats := []string{}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, d := range decisions {
		if !seen[d.Category] {
			cats = append(cats, d.Category)
			seen[d.Category] = true
		}
		counts[d.Category]++
	}
	prios := map[string]agent.CareLevel{}
	for _, c := range cats {
		prios[c] = agent.CareLevelMedium
	}
	return PrioritiesModel{
		categories: cats,
		priorities: prios,
		counts:     counts,
	}
}

func (m PrioritiesModel) Update(msg tea.Msg) (PrioritiesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.categories)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "h", "left":
			m.adjustLevel(-1)
		case "l", "right":
			m.adjustLevel(1)
		case "enter", "esc":
			return m, func() tea.Msg {
				return PrioritiesConfirmedMsg{Priorities: m.priorities}
			}
		}
	}
	return m, nil
}

func (m *PrioritiesModel) adjustLevel(dir int) {
	cat := m.categories[m.cursor]
	current := m.priorities[cat]
	idx := 0
	for i, l := range careLevels {
		if l.Key == current {
			idx = i
			break
		}
	}
	next := idx + dir
	if next >= 0 && next < len(careLevels) {
		m.priorities[cat] = careLevels[next].Key
	}
}

func (m PrioritiesModel) View() string {
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

	// Instruction
	lines = append(lines, "")
	lines = append(lines, "  "+BoldWhite.Render("How much do you care about each area?"))
	lines = append(lines, "")

	// Category rows
	for i, cat := range m.categories {
		isCur := i == m.cursor
		level := m.priorities[cat]
		var info struct {
			Color lipgloss.Color
			Bar   string
			Label string
		}
		for _, l := range careLevels {
			if l.Key == level {
				info.Color = l.Color
				info.Bar = l.Bar
				info.Label = l.Label
				break
			}
		}

		cursor := "  "
		if isCur {
			cursor = AccentStyle.Render("> ")
		}

		catStyle := DimStyle
		if isCur {
			catStyle = BoldWhite
		}

		count := m.counts[cat]
		row := fmt.Sprintf("  %s%s  %s %s  %s",
			cursor,
			catStyle.Render(pad(cat, 18)),
			lipgloss.NewStyle().Foreground(info.Color).Render(info.Bar),
			lipgloss.NewStyle().Foreground(info.Color).Render(pad(info.Label, 10)),
			DimStyle.Render(fmt.Sprintf("%d decisions", count)),
		)
		lines = append(lines, row)
	}

	// Fill vertical space
	usedLines := len(lines) + 5 // divider + detail + divider + footer + borders
	remaining := h - usedLines - 4
	for i := 0; i < remaining; i++ {
		lines = append(lines, "")
	}

	// Middle divider + selected category detail
	lines = append(lines, buildMiddleBorder(innerWidth))
	if m.cursor < len(m.categories) {
		cat := m.categories[m.cursor]
		level := m.priorities[cat]
		var desc string
		for _, l := range careLevels {
			if l.Key == level {
				desc = l.Desc
				break
			}
		}
		lines = append(lines, "  "+BoldAccent.Render(cat)+": "+DimStyle.Render(desc))
	}

	// Middle divider + footer
	lines = append(lines, buildMiddleBorder(innerWidth))
	footer := "  " + AccentStyle.Render("←→") + DimStyle.Render(" adjust  ") +
		AccentStyle.Render("↑↓") + DimStyle.Render(" navigate  ") +
		AccentStyle.Render("enter") + DimStyle.Render(" confirm")
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, "Domain Priorities", "")
}
