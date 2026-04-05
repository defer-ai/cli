package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var (
	prNumber  string
	diffOnly  bool
	prBasePath string
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Post decision diff to a GitHub PR",
	Long: `Show what decisions changed and optionally post as a GitHub PR comment.

Examples:
  defer review                                # print decision diff
  defer review --pr 42                        # post to PR #42
  defer review --pr 42 --base ../old-project  # compare against another project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Load current store
		store, err := decision.LoadStore(cwd)
		if err != nil {
			return fmt.Errorf("load decisions: %w", err)
		}
		if store == nil {
			return fmt.Errorf("no decision store found in current directory")
		}

		// Load baseline store (optional)
		var baseline *decision.DecisionStore
		if prBasePath != "" {
			baseline, err = decision.LoadStore(prBasePath)
			if err != nil {
				return fmt.Errorf("load baseline decisions from %s: %w", prBasePath, err)
			}
		} else {
			// Try to get baseline from git: load decisions.json from HEAD~1
			baseline = loadBaselineFromGit(cwd)
		}

		// Compute diff
		entries := decision.DiffStores(baseline, store)
		body := decision.FormatDiff(entries, store.Task)

		if diffOnly || prNumber == "" {
			fmt.Println(body)
			if prNumber == "" && !diffOnly {
				fmt.Fprintln(os.Stderr, "\nTip: use --pr <number> to post as a PR comment")
			}
			return nil
		}

		// Post to GitHub PR
		return postPRComment(prNumber, body)
	},
}

func loadBaselineFromGit(cwd string) *decision.DecisionStore {
	// Try to read decisions.json from the previous commit
	gitCmd := exec.Command("git", "show", "HEAD~1:.defer/decisions.json")
	gitCmd.Dir = cwd
	output, err := gitCmd.Output()
	if err != nil {
		return nil // No baseline available
	}

	var store decision.DecisionStore
	if err := json.Unmarshal(output, &store); err != nil {
		return nil
	}
	return &store
}

func postPRComment(pr, body string) error {
	// Try gh CLI first
	ghPath, err := exec.LookPath("gh")
	if err == nil {
		cmd := exec.Command(ghPath, "pr", "comment", pr, "--body", body)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gh pr comment failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Posted decision diff to PR %s\n", pr)
		return nil
	}

	// Fall back to GITHUB_TOKEN + direct API
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("no GitHub auth found\n\n  Option 1: install gh CLI and run 'gh auth login'\n  Option 2: export GITHUB_TOKEN=ghp_...")
	}

	// Detect repo from git remote
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fmt.Errorf("cannot detect GitHub repo from git remote: %w", err)
	}
	repo := parseGitHubRepo(strings.TrimSpace(string(remote)))
	if repo == "" {
		return fmt.Errorf("cannot parse GitHub repo from remote URL: %s", strings.TrimSpace(string(remote)))
	}

	// POST /repos/{owner}/{repo}/issues/{pr}/comments
	payload, _ := json.Marshal(map[string]string{"body": body})
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%s/comments", repo, pr)

	curlCmd := exec.Command("curl", "-s", "-X", "POST",
		"-H", "Authorization: token "+token,
		"-H", "Accept: application/vnd.github.v3+json",
		"-d", string(payload),
		url)
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr
	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Posted decision diff to PR %s\n", pr)
	return nil
}

func parseGitHubRepo(remote string) string {
	// Handle SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(remote, "git@github.com:") {
		repo := strings.TrimPrefix(remote, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo
	}
	// Handle HTTPS: https://github.com/owner/repo.git
	if strings.Contains(remote, "github.com/") {
		parts := strings.SplitN(remote, "github.com/", 2)
		if len(parts) == 2 {
			repo := strings.TrimSuffix(parts[1], ".git")
			return repo
		}
	}
	return ""
}

func init() {
	reviewCmd.Flags().StringVar(&prNumber, "pr", "", "PR number or URL")
	reviewCmd.Flags().BoolVar(&diffOnly, "diff-only", false, "Print diff without posting")
	reviewCmd.Flags().StringVar(&prBasePath, "base", "", "Baseline project path for comparison")
	rootCmd.AddCommand(reviewCmd)
}
