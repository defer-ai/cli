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
	categories []string
	priorities map[string]agent.CareLevel
	counts     map[string]int
	cursor     int
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

	var b strings.Builder
	b.WriteString("\n  " + BoldAccent.Render("How much do you care about each area?") + "\n")
	b.WriteString("  " + DimStyle.Render("←→ adjust, ↑↓ navigate, enter confirm") + "\n\n")

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
		b.WriteString(fmt.Sprintf("  %s%s  %s %s  %s\n",
			cursor,
			catStyle.Render(pad(cat, 18)),
			lipgloss.NewStyle().Foreground(info.Color).Render(info.Bar),
			lipgloss.NewStyle().Foreground(info.Color).Render(pad(info.Label, 10)),
			DimStyle.Render(fmt.Sprintf("%d decisions", count)),
		))
	}

	// Selected category detail
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
		b.WriteString("\n  " + Separator(w-4) + "\n")
		b.WriteString("  " + BoldAccent.Render(cat) + "  " + DimStyle.Render(desc) + "\n")
	}

	// Footer
	b.WriteString("\n  " + Separator(w-4) + "\n")
	b.WriteString("  " + AccentStyle.Render("←→") + DimStyle.Render(" adjust  "))
	b.WriteString(AccentStyle.Render("↑↓") + DimStyle.Render(" navigate  "))
	b.WriteString(AccentStyle.Render("enter") + DimStyle.Render(" confirm"))

	return b.String()
}
