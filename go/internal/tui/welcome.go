package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WelcomeModel is the initial screen where the user describes their project.
type WelcomeModel struct {
	input         textinput.Model
	width, height int
	mascotTick    int
}

func NewWelcomeModel() WelcomeModel {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.PromptStyle = AccentStyle
	ti.TextStyle = lipgloss.NewStyle()
	ti.PlaceholderStyle = DimStyle
	ti.Cursor.Style = AccentStyle
	ti.Focus()
	return WelcomeModel{input: ti}
}

func (m WelcomeModel) Update(msg tea.Msg) (WelcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			task := strings.TrimSpace(m.input.Value())
			if task != "" {
				return m, func() tea.Msg { return TaskSubmittedMsg{Task: task} }
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m WelcomeModel) View() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	innerWidth := w - 4 // 2 for border chars + 2 for padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	mascot := RenderMascot(MoodIdle, m.mascotTick)

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if home != "" {
		cwd = strings.Replace(cwd, home, "~", 1)
	}

	// Build content lines
	var lines []string

	// Empty line for spacing
	lines = append(lines, "")

	// Center the mascot
	mascotLines := strings.Split(mascot, "\n")
	for _, ml := range mascotLines {
		lines = append(lines, "     "+ml)
	}
	lines = append(lines, "")

	// Title and tagline
	lines = append(lines, "     "+BoldAccent.Render("defer"))
	lines = append(lines, "     "+DimStyle.Render("Zero-autonomy AI. Every decision is yours."))
	lines = append(lines, "")
	lines = append(lines, "     "+DimStyle.Render("cwd "+cwd))
	lines = append(lines, "")

	// Input via textinput component
	m.input.Width = innerWidth - 8
	if m.input.Width < 10 {
		m.input.Width = 10
	}

	inputView := m.input.View()

	border := lipgloss.RoundedBorder()
	bStyle := lipgloss.NewStyle().Foreground(BorderColor)
	inputBoxWidth := innerWidth - 6
	if inputBoxWidth < 10 {
		inputBoxWidth = 10
	}

	inputVisWidth := lipgloss.Width(inputView)
	inputPad := inputBoxWidth - inputVisWidth
	if inputPad < 0 {
		inputPad = 0
	}

	inputTop := "  " + bStyle.Render(border.TopLeft) + bStyle.Render(strings.Repeat(border.Top, inputBoxWidth+2)) + bStyle.Render(border.TopRight)
	inputMid := "  " + bStyle.Render(border.Left) + " " + inputView + strings.Repeat(" ", inputPad) + " " + bStyle.Render(border.Right)
	inputBot := "  " + bStyle.Render(border.BottomLeft) + bStyle.Render(strings.Repeat(border.Bottom, inputBoxWidth+2)) + bStyle.Render(border.BottomRight)

	lines = append(lines, inputTop)
	lines = append(lines, inputMid)
	lines = append(lines, inputBot)

	// Fill remaining vertical space
	usedLines := len(lines) + 2 // +2 for footer + border overhead
	remaining := h - usedLines - 4
	for i := 0; i < remaining; i++ {
		lines = append(lines, "")
	}

	// Footer
	footer := AccentStyle.Render("/help") + "  " + DimStyle.Render("ctrl+c quit")
	lines = append(lines, footer)

	content := strings.Join(lines, "\n")
	return buildBorderedBox(content, innerWidth, "", "")
}
