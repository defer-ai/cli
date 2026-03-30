package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/templates"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [target]",
	Short: "Scaffold Defer config files for your AI tool",
	Long: func() string {
		var b strings.Builder
		b.WriteString("Scaffold Defer config files for your AI tool.\n\n")
		b.WriteString("Available targets:\n")
		for _, t := range templates.TargetList() {
			tmpl := templates.Templates[t]
			b.WriteString(fmt.Sprintf("  %-14s %s → %s\n", t, tmpl.Description, tmpl.Filename))
		}
		b.WriteString("\nExample:\n  defer init claude-code\n  defer init cursor\n  defer init copilot\n  defer init codex\n  defer init universal\n")
		return b.String()
	}(),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := templates.TargetClaudeCode
		if len(args) > 0 {
			target = templates.Target(args[0])
		}

		tmpl, ok := templates.Templates[target]
		if !ok {
			var targets []string
			for _, t := range templates.TargetList() {
				targets = append(targets, string(t))
			}
			return fmt.Errorf("unknown target: %s (available: %s)", target, strings.Join(targets, ", "))
		}

		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, tmpl.Filename)

		// Create parent directory if needed (e.g. .github/ for copilot)
		dir := filepath.Dir(path)
		if dir != cwd {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}

		if err := os.WriteFile(path, []byte(tmpl.Content), 0o644); err != nil {
			return err
		}
		fmt.Printf("Created %s\n", tmpl.Filename)

		if !decision.StoreExists(cwd) {
			if _, err := decision.CreateStore(cwd, "(not started)"); err != nil {
				return err
			}
			fmt.Println("Created .defer/decisions.json")
		}

		fmt.Println("\nDefer mode is active. The AI will decompose tasks into decisions before coding.")
		return nil
	},
}
