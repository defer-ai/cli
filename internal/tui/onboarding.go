package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/defer-ai/cli/internal/api"
)

// OnboardingResult holds the user's choices from the setup wizard.
type OnboardingResult struct {
	Provider   string
	Model      string
	APIKey     string
	Effort     string // Claude Code only
	MascotSize string // "none", "small", "large"
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

// effortLevels for the Claude Code effort step.
var effortLevels = []struct {
	Name  string
	Value string
	Desc  string
}{
	{"Low", "low", "fastest, least exploration"},
	{"Medium", "medium", "balanced (default)"},
	{"High", "high", "deeper reasoning, slower"},
	{"Max", "max", "maximum effort, slowest"},
}

// Wizard step constants — using named steps for clarity.
const (
	stepProvider = iota
	stepModel
	stepAPIKey
	stepEffort
	stepMascot
	stepTheme
)

// OnboardingModel is the Bubbletea model for the setup wizard.
type OnboardingModel struct {
	cursor       int
	step         int
	choices      []ProviderChoice
	selected     ProviderChoice
	models       []api.ModelChoice // populated when entering stepModel
	customMode   bool              // true when on the "Custom..." text input
	customInput  strings.Builder
	keyInput     strings.Builder
	width        int
	height       int
	result       OnboardingResult
	mascotTick   int
	mascotMood   MascotMood
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
		result:  OnboardingResult{MascotSize: "small", Effort: "medium"},
	}
}

// Result returns the user's choices.
func (m OnboardingModel) Result() OnboardingResult {
	return m.result
}

func (m OnboardingModel) Init() tea.Cmd { return nil }

// nextStepAfter returns the next logical step from the current one,
// skipping API key for providers that don't need one and skipping
// effort for providers that aren't Claude Code.
func (m OnboardingModel) nextStepAfter(current int) int {
	switch current {
	case stepProvider:
		return stepModel
	case stepModel:
		if m.selected.NeedsKey {
			return stepAPIKey
		}
		if m.selected.Value == "claude" {
			return stepEffort
		}
		return stepMascot
	case stepAPIKey:
		if m.selected.Value == "claude" {
			return stepEffort
		}
		return stepMascot
	case stepEffort:
		return stepMascot
	case stepMascot:
		return stepTheme
	default:
		return stepTheme
	}
}

// prevStepBefore returns the previous step, again skipping steps that
// don't apply to the current provider.
func (m OnboardingModel) prevStepBefore(current int) int {
	switch current {
	case stepModel:
		return stepProvider
	case stepAPIKey:
		return stepModel
	case stepEffort:
		if m.selected.NeedsKey {
			return stepAPIKey
		}
		return stepModel
	case stepMascot:
		if m.selected.Value == "claude" {
			return stepEffort
		}
		if m.selected.NeedsKey {
			return stepAPIKey
		}
		return stepModel
	case stepTheme:
		return stepMascot
	default:
		return stepProvider
	}
}

func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.mascotTick++
		moods := []MascotMood{MoodIdle, MoodActive, MoodAsking, MoodDone, MoodError}
		m.mascotMood = moods[(m.mascotTick/30)%len(moods)]
		return m, tickCmd()

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.result.Skipped = true
			return m, tea.Quit
		}
		if msg.Type == tea.KeyEsc {
			if m.step == stepProvider {
				return m, nil // unskippable
			}
			// On the model step in custom-mode, esc cancels the custom input first
			if m.step == stepModel && m.customMode {
				m.customMode = false
				m.customInput.Reset()
				return m, nil
			}
			m.step = m.prevStepBefore(m.step)
			m.cursor = 0
			return m, nil
		}

		switch m.step {
		case stepProvider:
			return m.updateProviderStep(msg)
		case stepModel:
			return m.updateModelStep(msg)
		case stepAPIKey:
			return m.updateKeyStep(msg)
		case stepEffort:
			return m.updateEffortStep(msg)
		case stepMascot:
			return m.updateMascotStep(msg)
		case stepTheme:
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
		m.result.Provider = m.selected.Value
		// Load model list for the selected provider
		m.models = api.ModelsForProvider(m.selected.Value)
		m.step = stepModel
		m.cursor = 0
		return m, tickCmd()
	}
	return m, nil
}

func (m OnboardingModel) updateModelStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.customMode {
		// Text input mode for "Custom..." entry
		switch msg.Type {
		case tea.KeyEnter:
			val := strings.TrimSpace(m.customInput.String())
			if val == "" {
				return m, nil
			}
			m.result.Model = val
			m.customMode = false
			m.customInput.Reset()
			m.step = m.nextStepAfter(stepModel)
			m.cursor = 0
			return m, nil
		case tea.KeyBackspace:
			s := m.customInput.String()
			if len(s) > 0 {
				m.customInput.Reset()
				m.customInput.WriteString(s[:len(s)-1])
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.customInput.WriteString(string(msg.Runes))
			}
		}
		return m, nil
	}

	// List nav mode
	total := len(m.models) + 1 // +1 for "Custom..."
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < total-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if m.cursor == len(m.models) {
			// Custom...
			m.customMode = true
			m.customInput.Reset()
			return m, nil
		}
		m.result.Model = m.models[m.cursor].Value
		m.step = m.nextStepAfter(stepModel)
		m.cursor = 0
	}
	return m, nil
}

func (m OnboardingModel) updateKeyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.result.APIKey = strings.TrimSpace(m.keyInput.String())
		m.step = m.nextStepAfter(stepAPIKey)
		m.cursor = 0
		return m, nil
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

func (m OnboardingModel) updateEffortStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(effortLevels)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		m.result.Effort = effortLevels[m.cursor].Value
		m.step = m.nextStepAfter(stepEffort)
		m.cursor = 1 // default mascot to "small"
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
		m.step = m.nextStepAfter(stepMascot)
		m.cursor = 0 // default to first theme
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
	case stepProvider:
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
		b.WriteString(dim.Render("  ↑↓ navigate  enter select") + "\n")

	case stepModel:
		b.WriteString(bold.Render(fmt.Sprintf("  Which %s model?", m.selected.Name)) + "\n\n")

		if m.customMode {
			b.WriteString(dim.Render("  Type a model name (e.g. claude-sonnet-4-6, gpt-4o, ...):") + "\n\n")
			input := m.customInput.String()
			if input == "" {
				b.WriteString("  " + accent.Render(">") + " " + dim.Render("(type here)") + "\n")
			} else {
				b.WriteString("  " + accent.Render(">") + " " + input + "\n")
			}
			b.WriteString("\n")
			b.WriteString(dim.Render("  enter confirm  esc cancel") + "\n")
			break
		}

		const nameCol = 22
		for i, choice := range m.models {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = accent.Render("> ")
				nameStyle = selStyle
			}
			name := nameStyle.Render(choice.Name)
			pad := nameCol - lipgloss.Width(name)
			if pad < 1 {
				pad = 1
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", cursor, name, strings.Repeat(" ", pad), dim.Render(choice.Description)))
		}
		// Custom... entry
		customCursor := "  "
		customStyle := dim
		if m.cursor == len(m.models) {
			customCursor = accent.Render("> ")
			customStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f97316"))
		}
		b.WriteString(fmt.Sprintf("%s%s\n", customCursor, customStyle.Render("Custom...")))

		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc back") + "\n")

	case stepAPIKey:
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

	case stepEffort:
		b.WriteString(bold.Render("  Effort level?") + "\n\n")
		b.WriteString(dim.Render("  Controls how much Claude Code thinks before acting.") + "\n\n")
		const nameCol = 10
		for i, e := range effortLevels {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = accent.Render("> ")
				nameStyle = selStyle
			}
			name := nameStyle.Render(e.Name)
			pad := nameCol - lipgloss.Width(name)
			if pad < 1 {
				pad = 1
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", cursor, name, strings.Repeat(" ", pad), dim.Render(e.Desc)))
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc back") + "\n")

	case stepMascot:
		b.WriteString(bold.Render("  Mascot size?") + "\n\n")

		sz := mascotSizes[m.cursor]
		if sz.Size > 0 {
			mascot := RenderMascotAtSize(m.mascotMood, m.mascotTick, sz.Size, 3)
			for _, line := range strings.Split(mascot, "\n") {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
		}

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

	case stepTheme:
		b.WriteString(bold.Render("  Theme?") + "\n\n")

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
			if namePad < 1 {
				namePad = 1
			}
			swatch := lipgloss.NewStyle().Foreground(t.Accent).Render("████")
			b.WriteString(cursor + name + strings.Repeat(" ", namePad) + swatch + "\n")
		}
		b.WriteString("\n")
		b.WriteString(dim.Render("  ↑↓ navigate  enter select  esc back") + "\n")
	}

	return b.String()
}
