package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/config"
	"github.com/defer-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var (
	model     string
	provider  string
	apiKey    string
	debug     bool
	noMascot  bool
)

var rootCmd = &cobra.Command{
	Use:   "defer [task]",
	Short: "Zero-Autonomy AI. Every decision is yours.",
	Long: `Zero-autonomy AI. Every decision is yours.

Describe your project and defer decomposes it into decisions, lets you
set care levels (auto or review), then implements while you watch,
chat, and challenge in real-time.

Quick start:
  defer "build a REST API"       New project
  defer                          Resume last session
  defer setup                    Change AI provider

Keybindings:
  tab / shift+tab                Cycle focus: tree → chat → resolver
  ↑↓                             Navigate decisions / options
  enter                          Inspect / confirm
  /                              Search decisions
  ctrl+q                         Quit

Commands:
  defer init [target]            Scaffold config for AI tools
  defer init --preset <name>     Seed decisions from a template
  defer stats                    Decision analytics
  defer trailers                 Git Decision-Ref trailers
  defer commit -m "msg"          Git commit with trailers
  defer import <path> [@IDs]     Import decisions from another project
  defer review --pr <n>          Post decision diff to a GitHub PR
  defer serve --mcp              Run as MCP server
  defer setup                    Change AI provider
  defer sessions list            List sessions
  defer sessions delete          Delete .defer/
  defer sessions export          Print DECISIONS.md

Configuration:
  ~/.defer/config.json           Global defaults
  .defer/config.json             Project overrides
  ~/.defer/keybindings.json      Custom keybindings
  .defer/skills/*.md             Prompt overrides
  .defer/templates/*.yaml        Decision presets`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// First run: if no global config exists and no flags given, run setup wizard
		if needsOnboarding() && provider == "" && apiKey == "" {
			result, err := runSetupWizard()
			if err != nil || result.Skipped {
				// Ctrl+C or error — exit entirely
				os.Exit(0)
			}
			saveSetupResult(result)
			// Apply wizard choices for this session
			if result.Provider != "" && result.Provider != "claude" {
				provider = result.Provider
			}
			if result.APIKey != "" {
				apiKey = result.APIKey
			}
			// Apply theme for this session
			if result.Theme != "" {
				for _, t := range tui.Themes {
					if t.Name == result.Theme {
						tui.ApplyTheme(t.Accent)
						break
					}
				}
			}
		}

		// Load saved config — CLI flags override saved values
		if cfg, err := config.LoadGlobalConfig(); err == nil && cfg != nil {
			if provider == "" && cfg.Provider != "" {
				provider = cfg.Provider
			}
			if apiKey == "" && cfg.APIKey != "" {
				apiKey = cfg.APIKey
			}
			if model == "sonnet" && cfg.Model != "" {
				model = cfg.Model
			}
		}

		p, err := api.ResolveProvider(provider, apiKey, model)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  export OPENAI_API_KEY=sk-...")
			fmt.Fprintln(os.Stderr, "  export GROQ_API_KEY=gsk_...")
			fmt.Fprintln(os.Stderr, "  npm install -g @anthropic-ai/claude-code && claude login")
			fmt.Fprintln(os.Stderr, "  defer --provider ollama --model llama3.1")
			fmt.Fprintln(os.Stderr, "  defer setup")
			os.Exit(1)
		}

		task := ""
		if len(args) > 0 {
			task = args[0]
		}

		cwd, _ := os.Getwd()

		if debug {
			return runDebug(task, model, p, cwd)
		}

		// Load display preferences from config
		mascotSize := "medium"
		if cfgLoaded, _ := config.LoadGlobalConfig(); cfgLoaded != nil {
			if cfgLoaded.MascotSize != "" {
				mascotSize = cfgLoaded.MascotSize
			}
			if cfgLoaded.Theme != "" {
				for _, t := range tui.Themes {
					if t.Name == cfgLoaded.Theme {
						tui.ApplyTheme(t.Accent)
						break
					}
				}
			}
		}
		if noMascot {
			mascotSize = "none"
		}

		m := tui.NewModel(task, p, cwd, tui.ModelOpts{
			ShowMascot: mascotSize != "none",
			MascotSize: tui.MascotDisplaySize(mascotSize),
			Version:    Version,
			ModelName:  p.GetModel(),
		})
		prog := tea.NewProgram(m, tea.WithAltScreen())
		_, err = prog.Run()
		return err
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true // no shell completion command
	rootCmd.PersistentFlags().StringVar(&model, "model", "sonnet", "Model to use (sonnet, opus, haiku, or provider-specific ID)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "", "AI provider (openai, groq, mistral, together, ollama, or URL)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides environment variable)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Run headless (no TUI), print all output to stdout")
	rootCmd.PersistentFlags().BoolVar(&noMascot, "no-mascot", false, "Hide the mascot header")
	rootCmd.AddCommand(initCmd)
}

func Execute() {
	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
