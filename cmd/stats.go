package cmd

import (
	"fmt"
	"os"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show decision analytics for the current session",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()

		store, err := decision.LoadStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}
		if store == nil {
			fmt.Fprintln(os.Stderr, "No defer session found in the current directory.")
			return nil
		}

		s := decision.ComputeStats(store)
		fmt.Print(decision.FormatStats(s))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
