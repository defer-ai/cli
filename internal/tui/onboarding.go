package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingResult holds the user's choices from the setup wizard.
type OnboardingResult struct {
	Provider   string
	APIKey     string
	MascotSize string // "none", "small", "medium", "large"
	Theme      string // accent color name
	Skipped    bool
}

// ProviderChoice represents a selectable provider.
type ProviderChoice struct {
	Name        string
	Value       string
	Description string
	NeedsKey    bool
	KeyEnvVar   string
}

// ProviderChoices is the list of supported providers.
var ProviderChoices = []ProviderChoice{
	{Name: "Claude Code", Value: "claude", Description: "Anthropic's CLI (free with subscription)", NeedsKey: false},
	{Name: "OpenAI", Value: "openai", Description: "GPT-4o and variants", NeedsKey: true, KeyEnvVar: "OPENAI_API_KEY"},
	{Name: "Groq", Value: "groq", Description: "Fast inference (Llama, Mixtral)", NeedsKey: true, KeyEnvVar: "GROQ_API_KEY"},
	{Name: "Mistral", Value: "mistral", Description: "Mistral AI models", NeedsKey: true, KeyEnvVar: "MISTRAL_API_KEY"},
	{Name: "Together", Value: "together", Description: "Open-source models", NeedsKey: true, KeyEnvVar: "TOGETHER_API_KEY"},
	{Name: "OpenRouter", Value: "openrouter", Description: "Multi-provider router", NeedsKey: true, KeyEnvVar: "OPENROUTER_API_KEY"},
	{Name: "Ollama", Value: "ollama", Description: "Local models (no API key)", NeedsKey: false},
}

type mascotSizeOption struct {
	Name  string
	Value string
	Size  int // display pixels (0 = none)
}

var mascotSizes = []mascotSizeOption{
	{Name: "None", Value: "none", Size: 0},
	{Name: "Small", Value: "small", Size: 15},
	{Name: "Large", Value: "large", Size: 30},
}

// MascotDisplaySize returns the pixel size for a config mascotSize value.
func MascotDisplaySize(size string) int {
	switch size {
	case "large":
		return 30
	case "none":
		return 0
	default: // "small" or empty
		return 15
	}
}

// OnboardingModel is the Bubbletea model for the setup wizard.
type OnboardingModel struct {
	cursor     int
	step       int // 0=provider, 1=api key, 2=mascot size, 3=theme
	choices    []ProviderChoice
	selected   ProviderChoice
	keyInput   strings.Builder
	width      int
	height     int
	result     OnboardingResult
	mascotTick int
	mascotMood MascotMood
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewOnboardingModel creates the setup wizard.
func NewOnboardingModel() OnboardingModel {
	return OnboardingModel{
		choices: ProviderChoices,
		width:   80,
		height:  24,
		result:  OnboardingResult{MascotSize: "small"},
	}
}

// Result returns the user's choices.
func (m OnboardingModel) Result() OnboardingResult {
	return m.result
}

func (m OnboardingModel) Init() tea.Cmd { return nil }

func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.mascotTick++
		// Cycle mood every 30 ticks (3s)
		moods := []MascotMood{MoodIdle, MoodActive, MoodAsking, MoodDone, MoodError}
		m.mascotMood = moods[(m.mascotTick/30)%len(moods)]
		return m, tickCmd()

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.result.Skipped = true
			return m, tea.Quit
		}
		// Esc goes back one step, but never skips the wizard
		if msg.Type == tea.KeyEsc {
			if m.step == 3 {
				m.step = 2
				m.cursor = 1 // default to "small"
				return m, nil
			}
			if m.step == 2 {
				m.step = 0
				m.cursor = 0
				return m, nil
			}
			if m.step == 1 {
				m.step = 0
				m.cursor = 0
				return m, nil
			}
			return m, nil
		}

		switch m.step {
		case 0:
			return m.updateProviderStep(msg)
		case 1:
			return m.updateKeyStep(msg)
		case 2:
			return m.updateMascotStep(msg)
		case 3:
			return m.updateThemeStep(msg)
		}
	}
	return m, nil
}

func (m OnboardingModel) updateProviderStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		m.selected = m.choices[m.cursor]
		if m.selected.NeedsKey {
			m.step = 1
			m.cursor = 0
			m.keyInput.Reset()
		} else {
			m.result.Provider = m.selected.Value
			m.step = 2
			m.cursor = 1 // default to "small"
		}
		return m, tickCmd() // start ticking for mascot animation
	}
	return m, nil
}

func (m OnboardingModel) updateKeyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.result.Provider = m.selected.Value
		m.result.APIKey = strings.TrimSpace(m.keyInput.String())
		m.step = 2
		m.cursor = 1 // default to "small"
		return m, tickCmd()
	case tea.KeyBackspace:
		s := m.keyInput.String()
		if len(s) > 0 {
			m.keyInput.Reset()
			m.keyInput.WriteString(s[:len(s)-1])
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.keyInput.WriteString(string(msg.Runes))
		}
	}
	return m, nil
}

func (m OnboardingModel) updateMascotStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(mascotSizes)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		m.result.MascotSize = mascotSizes[m.cursor].Value
		m.step = 3
		m.cursor = 0 // default to first theme (Orange)
	}
	return m, nil
}

func (m OnboardingModel) updateThemeStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(Themes)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		m.result.Theme = Themes[m.cursor].Name
		return m, tea.Quit
	}
	// Live-apply theme for preview
	ApplyTheme(Themes[m.cursor].Accent)
	return m, nil
}

func (m OnboardingModel) View() string {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#f97316"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold := lipgloss.NewStyle().Bold(true)
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f97316")).Bold(true)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(bold.Render("  defer") + dim.Render(" — setup") + "\n")
	b.WriteString(dim.Render("  Zero-autonomy AI. Every decision is yours.") + "\n\n")

	switch m.step {
	case 0:
		b.WriteString(bold.Render("  Which AI provider will you use?") + "\n\n")
		for i, c := range m.choices {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = accent.Render("> ")
				nameStyle = selStyle
			}
			name := nameStyle.Render(c.Name)
			visW := lipgloss.Width(name)
			pad := ""
			if visW < 14 {
				pad = strings.Repeat(" ", 14-visW)
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", cursor, name, pad, dim.Render(c.Description)))
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc skip") + "\n")

	case 1:
		b.WriteString(bold.Render(fmt.Sprintf("  API key for %s", m.selected.Name)) + "\n\n")

		key := m.keyInput.String()
		if key == "" {
			b.WriteString("  " + dim.Render("Paste your API key here...") + "\n")
		} else {
			display := key
			if len(display) > 12 {
				display = display[:4] + strings.Repeat("*", len(display)-8) + display[len(display)-4:]
			}
			b.WriteString("  " + display + "\n")
		}

		b.WriteString("\n")
		if m.selected.KeyEnvVar != "" {
			b.WriteString(dim.Render(fmt.Sprintf("  Or set %s in your shell instead.", m.selected.KeyEnvVar)) + "\n")
		}
		b.WriteString(dim.Render("  enter confirm  esc back") + "\n")

	case 2:
		b.WriteString(bold.Render("  Mascot size?") + "\n\n")

		// Render the mascot preview for the currently highlighted size
		sz := mascotSizes[m.cursor]
		if sz.Size > 0 {
			mascot := RenderMascotAtSize(m.mascotMood, m.mascotTick, sz.Size, 3)
			for _, line := range strings.Split(mascot, "\n") {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
		}

		// Size options
		for i, s := range mascotSizes {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = accent.Render("> ")
				nameStyle = selStyle
			}
			label := nameStyle.Render(s.Name)
			desc := ""
			switch s.Value {
			case "none":
				desc = "no mascot"
			case "small":
				desc = "15px"
			case "large":
				desc = "30px"
			}
			visW := lipgloss.Width(label)
			pad := ""
			if visW < 10 {
				pad = strings.Repeat(" ", 10-visW)
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", cursor, label, pad, dim.Render(desc)))
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc back") + "\n")

	case 3:
		b.WriteString(bold.Render("  Theme?") + "\n\n")

		// Mascot preview with current theme
		if m.result.MascotSize != "none" {
			sz := MascotDisplaySize(m.result.MascotSize)
			mascot := RenderMascotAtSize(m.mascotMood, m.mascotTick, sz, 3)
			for _, line := range strings.Split(mascot, "\n") {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
		}

		const nameCol = 12
		for i, t := range Themes {
			cursor := "  "
			if i == m.cursor {
				cursor = lipgloss.NewStyle().Foreground(t.Accent).Render("> ")
			}
			name := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(t.Name)
			namePad := nameCol - lipgloss.Width(name)
			if namePad < 1 { namePad = 1 }

			swatch := lipgloss.NewStyle().Foreground(t.Accent).Render("████")
			b.WriteString(cursor + name + strings.Repeat(" ", namePad) + swatch + "\n")
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc back") + "\n")
	}

	return b.String()
}
