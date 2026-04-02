package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Accent  = lipgloss.Color("#f97316")
	DimGray = lipgloss.Color("240")

	// Base text styles
	AccentStyle  = lipgloss.NewStyle().Foreground(Accent)
	BoldAccent   = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(DimGray)
	BoldWhite    = lipgloss.NewStyle().Bold(true)
	GreenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	YellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	MagentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	RedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	BlueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	// Border colors
	BorderColor       = lipgloss.Color("240")
	ActiveBorderColor = Accent

	// Title style for the top border label
	TitleStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	// Detail view styles
	DetailTitleStyle    = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	DetailQuestionStyle = lipgloss.NewStyle().Bold(true)
	DetailContextStyle  = lipgloss.NewStyle().Foreground(DimGray).Italic(true)

	// Category header in tree
	CategoryStyle = lipgloss.NewStyle().Foreground(Accent).Bold(true)
)

// Separator returns a horizontal rule of the given width.
func Separator(width int) string {
	if width > 200 {
		width = 200
	}
	if width < 0 {
		width = 0
	}
	return DimStyle.Render(strings.Repeat("─", width))
}

// buildBorderedBox renders content inside a rounded border with an optional
// title in the top-left and an optional status in the top-right.
// innerWidth is the desired content width (excluding border chars).
func buildBorderedBox(content string, innerWidth int, title, rightStatus string) string {
	if innerWidth < 10 {
		innerWidth = 10
	}

	border := lipgloss.RoundedBorder()
	topLeft := border.TopLeft
	topRight := border.TopRight
	bottomLeft := border.BottomLeft
	bottomRight := border.BottomRight
	horizontal := border.Top
	vertical := border.Left

	bStyle := lipgloss.NewStyle().Foreground(BorderColor)

	// Build top border line with title and status
	titleStr := ""
	if title != "" {
		titleStr = " " + TitleStyle.Render(title) + " "
	}
	rightStr := ""
	if rightStatus != "" {
		rightStr = " " + DimStyle.Render(rightStatus) + " "
	}

	// We need to calculate visible lengths for the styled strings
	titleVisLen := lipgloss.Width(titleStr)
	rightVisLen := lipgloss.Width(rightStr)

	fillLen := innerWidth - titleVisLen - rightVisLen
	if fillLen < 0 {
		fillLen = 0
	}

	topLine := bStyle.Render(topLeft) + bStyle.Render(horizontal) +
		titleStr +
		bStyle.Render(strings.Repeat(horizontal, fillLen)) +
		rightStr +
		bStyle.Render(horizontal) + bStyle.Render(topRight)

	// Build bottom border
	bottomFill := innerWidth
	if bottomFill < 0 {
		bottomFill = 0
	}
	bottomLine := bStyle.Render(bottomLeft) +
		bStyle.Render(strings.Repeat(horizontal, bottomFill+2)) +
		bStyle.Render(bottomRight)

	// Build content lines with side borders
	lines := strings.Split(content, "\n")
	var sb strings.Builder
	sb.WriteString(topLine)
	sb.WriteString("\n")

	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		padRight := innerWidth - lineWidth
		if padRight < 0 {
			padRight = 0
		}
		sb.WriteString(bStyle.Render(vertical) + " " + line + strings.Repeat(" ", padRight) + " " + bStyle.Render(vertical))
		sb.WriteString("\n")
	}

	sb.WriteString(bottomLine)
	return sb.String()
}

// buildMiddleBorder creates a horizontal divider that connects to the side borders.
func buildMiddleBorder(innerWidth int) string {
	border := lipgloss.RoundedBorder()
	bStyle := lipgloss.NewStyle().Foreground(BorderColor)

	// Use tee characters for connecting to side borders
	fill := innerWidth
	if fill < 0 {
		fill = 0
	}
	return bStyle.Render("├") +
		bStyle.Render(strings.Repeat(border.Top, fill+2)) +
		bStyle.Render("┤")
}

