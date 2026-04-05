package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var (
	commitMessage string
	commitDryRun  bool
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Run git commit with Decision-Ref trailers appended",
	Long: `Run git commit with the given message, automatically appending
Decision-Ref trailers for all decided decisions in the current project.

If no decisions are decided, the commit runs with the message as-is.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check that git exists in PATH
		gitPath, err := exec.LookPath("git")
		if err != nil {
			return fmt.Errorf("git not found in PATH")
		}

		cwd, _ := os.Getwd()

		// Build the full commit message
		msg := commitMessage
		store, err := decision.LoadStore(cwd)
		if err != nil {
			return err
		}

		if store != nil {
			trailers := decision.Trailers(store)
			if trailers != "" {
				msg = msg + "\n\n" + trailers
			}
		}

		if commitDryRun {
			fmt.Print(msg)
			return nil
		}

		// Run git commit
		gitCmd := exec.Command(gitPath, "commit", "-m", msg)
		gitCmd.Dir = cwd
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		gitCmd.Stdin = os.Stdin
		return gitCmd.Run()
	},
}

func init() {
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Commit message (required)")
	_ = commitCmd.MarkFlagRequired("message")
	commitCmd.Flags().BoolVar(&commitDryRun, "dry-run", false, "Print the full message without committing")
	rootCmd.AddCommand(commitCmd)
}
