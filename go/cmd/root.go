package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	model    string
	provider string
	apiKey   string
	debug    bool
	mcpFlag  bool
)

var rootCmd = &cobra.Command{
	Use:   "defer [task]",
	Short: "Zero-Autonomy AI. Every decision is yours.",
	Long: `defer decomposes your project into decisions, lets you set how much
you care about each domain, and implements everything while you watch
and challenge in real-time.

Providers (auto-detected from environment):
  Claude Code subprocess  (default, free with subscription)
  OpenAI                  OPENAI_API_KEY
  Groq                    GROQ_API_KEY
  Mistral                 MISTRAL_API_KEY
  Together                TOGETHER_API_KEY
  Ollama                  --provider ollama (local, no key)
  Any OpenAI-compatible   --provider <url> --api-key <key>

`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := api.ResolveProvider(provider, apiKey, model)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  export OPENAI_API_KEY=sk-...")
			fmt.Fprintln(os.Stderr, "  export GROQ_API_KEY=gsk_...")
			fmt.Fprintln(os.Stderr, "  npm install -g @anthropic-ai/claude-code && claude login")
			fmt.Fprintln(os.Stderr, "  defer --provider ollama --model llama3.1")
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

		m := tui.NewModel(task, p, cwd)
		prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
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
	rootCmd.PersistentFlags().BoolVar(&mcpFlag, "mcp", false, "Enable MCP (Model Context Protocol) server connections")
	rootCmd.AddCommand(initCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
