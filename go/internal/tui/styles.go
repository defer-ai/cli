package tui

import "github.com/charmbracelet/lipgloss"

var (
	Accent       = lipgloss.Color("#f97316")
	DimGray      = lipgloss.Color("240")
	AccentStyle  = lipgloss.NewStyle().Foreground(Accent)
	BoldAccent   = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(DimGray)
	BoldWhite    = lipgloss.NewStyle().Bold(true)
	GreenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	YellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	MagentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	RedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func Separator(width int) string {
	if width > 120 {
		width = 120
	}
	s := ""
	for i := 0; i < width; i++ {
		s += "─"
	}
	return DimStyle.Render(s)
}
