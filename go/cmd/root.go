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
	model string
	debug bool
)

var rootCmd = &cobra.Command{
	Use:   "defer [task]",
	Short: "Zero-Autonomy AI. Every decision is yours.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hasAPIKey := api.IsConfigured()
		hasClaude := api.IsClaudeInstalled()

		if !hasAPIKey && !hasClaude {
			fmt.Fprintln(os.Stderr, "Error: No AI provider available.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Option 1: Set ANTHROPIC_API_KEY for direct API (tool interception)")
			fmt.Fprintln(os.Stderr, "  export ANTHROPIC_API_KEY=sk-ant-...")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Option 2: Install Claude Code (free with subscription)")
			fmt.Fprintln(os.Stderr, "  npm install -g @anthropic-ai/claude-code && claude login")
			os.Exit(1)
		}

		task := ""
		if len(args) > 0 {
			task = args[0]
		}

		var client *api.Client
		var ccProvider *api.ClaudeCodeProvider

		if hasAPIKey {
			client = api.NewClient(model)
		} else {
			ccProvider = api.NewClaudeCodeProvider(model)
		}

		cwd, _ := os.Getwd()

		if debug {
			return runDebug(task, model, client, ccProvider, cwd)
		}

		m := tui.NewModel(task, client, ccProvider, cwd)

		p := tea.NewProgram(m,
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&model, "model", "sonnet", "Model to use (sonnet, opus, haiku)")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Run headless (no TUI), print all output to stdout")
	rootCmd.AddCommand(initCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
