package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// WelcomeModel is the initial screen where the user describes their project.
type WelcomeModel struct {
	input         string
	width, height int
	mascotTick    int
}

func NewWelcomeModel() WelcomeModel {
	return WelcomeModel{}
}

func (m WelcomeModel) Update(msg tea.Msg) (WelcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			task := strings.TrimSpace(m.input)
			if task != "" {
				return m, func() tea.Msg { return TaskSubmittedMsg{Task: task} }
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}
	}
	return m, nil
}

func (m WelcomeModel) View() string {
	w := m.width
	if w < 40 {
		w = 80
	}

	mascot := RenderMascot(MoodIdle, m.mascotTick)
	mascotLines := strings.Split(mascot, "\n")

	info := BoldAccent.Render("defer") + "\n"
	info += DimStyle.Render("Zero-autonomy AI. Every decision is yours.") + "\n\n"

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if home != "" {
		cwd = strings.Replace(cwd, home, "~", 1)
	}
	info += DimStyle.Render("cwd "+cwd) + "\n\n"
	info += DimStyle.Render("Describe your project to start.")

	infoLines := strings.Split(info, "\n")

	var b strings.Builder
	mascotW := 36
	maxLines := len(mascotLines)
	if len(infoLines) > maxLines {
		maxLines = len(infoLines)
	}
	for i := 0; i < maxLines; i++ {
		ml := ""
		if i < len(mascotLines) {
			ml = mascotLines[i]
		}
		il := ""
		if i < len(infoLines) {
			il = infoLines[i]
		}
		b.WriteString(fmt.Sprintf("  %-*s  %s\n", mascotW, ml, il))
	}

	// Fill remaining space
	for i := 0; i < m.height-maxLines-3; i++ {
		b.WriteString("\n")
	}

	// Input
	b.WriteString("  " + Separator(w-4) + "\n")
	b.WriteString("  " + AccentStyle.Render("> ") + m.input + AccentStyle.Render("_"))

	return b.String()
}
