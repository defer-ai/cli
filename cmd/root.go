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
	Long: `defer decomposes your project into decisions, lets you set how much
you care about each domain, and implements everything while you watch
and challenge in real-time.

Workflow:
  1. Describe your project        defer "build a REST API"
  2. Set care levels per domain   skip / low / medium / high / paranoid
  3. Inspect & challenge           navigate tree, chat with @ID references
  4. Watch implementation          executor plans, implements, verifies
  5. Everything tracked            DECISIONS.md + .defer/decisions.json

Providers (auto-detected from environment):
  Claude Code subprocess  (default, free with subscription)
  OpenAI                  OPENAI_API_KEY
  Groq                    GROQ_API_KEY
  Mistral                 MISTRAL_API_KEY
  Together                TOGETHER_API_KEY
  Ollama                  --provider ollama (local, no key)
  Any OpenAI-compatible   --provider <url> --api-key <key>

TUI Keybindings (in decision tree):
  j/k or up/down   Navigate decisions
  enter             Inspect decision (split-pane on wide terminals)
  /                 Search/filter decisions
  tab               Open conversation panel
  q or esc          Back

TUI Keybindings (in decision detail):
  j/k               Navigate options
  enter              Confirm option
  c                  Custom answer
  s                  Shuffle (generate new options)
  w                  Why? (explain tradeoffs)
  a                  Ask a question
  q                  Back to tree

TUI Keybindings (in conversation):
  enter              Send message
  @ID                Reference a decision
  tab                Auto-complete @ID / back to tree
  esc                Back to tree

Configuration:
  ~/.defer/config.json           Global defaults (care levels, model, provider)
  .defer/config.json             Project overrides
  ~/.defer/keybindings.json      Custom keybindings
  .defer/skills/*.md             Custom skill/prompt overrides

`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// First run: if no global config exists and no flags given, run setup wizard
		if needsOnboarding() && provider == "" && apiKey == "" {
			result, err := runSetupWizard()
			if err == nil && !result.Skipped {
				saveSetupResult(result)
				// Apply wizard choices for this session
				if result.Provider != "" && result.Provider != "claude" {
					provider = result.Provider
				}
				if result.APIKey != "" {
					apiKey = result.APIKey
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

		m := tui.NewModel(task, p, cwd, tui.ModelOpts{
			ShowMascot: !noMascot,
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
