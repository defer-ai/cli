package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingResult holds the user's choices from the setup wizard.
type OnboardingResult struct {
	Provider string
	APIKey   string
	Skipped  bool
}

// ProviderChoice represents a selectable provider.
type ProviderChoice struct {
	Name        string
	Value       string // internal provider value
	Description string
	NeedsKey    bool
	KeyEnvVar   string
}

// ProviderChoices is the list of supported providers for the setup wizard.
var ProviderChoices = []ProviderChoice{
	{Name: "Claude Code", Value: "claude", Description: "Anthropic's CLI (free with subscription)", NeedsKey: false},
	{Name: "OpenAI", Value: "openai", Description: "GPT-4o and variants", NeedsKey: true, KeyEnvVar: "OPENAI_API_KEY"},
	{Name: "Groq", Value: "groq", Description: "Fast inference (Llama, Mixtral)", NeedsKey: true, KeyEnvVar: "GROQ_API_KEY"},
	{Name: "Mistral", Value: "mistral", Description: "Mistral AI models", NeedsKey: true, KeyEnvVar: "MISTRAL_API_KEY"},
	{Name: "Together", Value: "together", Description: "Open-source models", NeedsKey: true, KeyEnvVar: "TOGETHER_API_KEY"},
	{Name: "OpenRouter", Value: "openrouter", Description: "Multi-provider router", NeedsKey: true, KeyEnvVar: "OPENROUTER_API_KEY"},
	{Name: "Ollama", Value: "ollama", Description: "Local models (no API key)", NeedsKey: false},
}

// OnboardingModel is the Bubbletea model for the setup wizard.
type OnboardingModel struct {
	cursor   int
	step     int // 0=provider, 1=api key
	choices  []ProviderChoice
	selected ProviderChoice
	keyInput strings.Builder
	width    int
	height   int
	result   OnboardingResult
}

// NewOnboardingModel creates the setup wizard.
func NewOnboardingModel() OnboardingModel {
	return OnboardingModel{
		choices: ProviderChoices,
		width:   80,
		height:  24,
	}
}

// Result returns the user's choices after the wizard completes.
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

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			m.result.Skipped = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.step > 0 {
				m.step--
				return m, nil
			}
			m.result.Skipped = true
			return m, tea.Quit
		}

		switch m.step {
		case 0:
			return m.updateProviderStep(msg)
		case 1:
			return m.updateKeyStep(msg)
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
			m.keyInput.Reset()
		} else {
			m.result = OnboardingResult{Provider: m.selected.Value}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m OnboardingModel) updateKeyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.result = OnboardingResult{
			Provider: m.selected.Value,
			APIKey:   strings.TrimSpace(m.keyInput.String()),
		}
		return m, tea.Quit
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
			b.WriteString(fmt.Sprintf("%s%-14s %s\n", cursor, nameStyle.Render(c.Name), dim.Render(c.Description)))
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc skip") + "\n")

	case 1:
		b.WriteString(bold.Render(fmt.Sprintf("  API key for %s", m.selected.Name)) + "\n\n")
		if m.selected.KeyEnvVar != "" {
			b.WriteString(dim.Render(fmt.Sprintf("  Alternatively, set %s in your shell.\n\n", m.selected.KeyEnvVar)))
		}
		key := m.keyInput.String()
		display := key
		if len(display) > 40 {
			display = display[:8] + "..." + display[len(display)-4:]
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", accent.Render(">"), display))
		b.WriteString("\n")
		b.WriteString(dim.Render("  enter confirm  esc back  (leave empty to use env var)") + "\n")
	}

	return b.String()
}
