package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/templates"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [target]",
	Short: "Scaffold Defer config files",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := templates.TargetClaudeCode
		if len(args) > 0 {
			target = templates.Target(args[0])
		}

		tmpl, ok := templates.Templates[target]
		if !ok {
			return fmt.Errorf("unknown target: %s (available: claude-code, universal)", target)
		}

		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, tmpl.Filename)

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

		fmt.Println("\nDefer mode is active.")
		return nil
	},
}
