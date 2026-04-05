package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage defer sessions",
	Long:  "List, delete, or export defer sessions found in the current directory tree.",
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all defer sessions found in the current directory and subdirectories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		found := 0

		err := filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			// Skip hidden directories other than .defer
			base := filepath.Base(path)
			if info.IsDir() && strings.HasPrefix(base, ".") && base != ".defer" {
				return filepath.SkipDir
			}
			// Skip node_modules, vendor, etc.
			if info.IsDir() && (base == "node_modules" || base == "vendor" || base == "__pycache__") {
				return filepath.SkipDir
			}
			if !info.IsDir() && base == "decisions.json" && filepath.Base(filepath.Dir(path)) == ".defer" {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil
				}
				var store decision.DecisionStore
				if jsonErr := json.Unmarshal(data, &store); jsonErr != nil {
					return nil
				}

				relDir, _ := filepath.Rel(cwd, filepath.Dir(filepath.Dir(path)))
				if relDir == "" || relDir == "." {
					relDir = "./"
				} else {
					relDir = relDir + "/"
				}

				answered := 0
				pending := 0
				for _, d := range store.Decisions {
					if d.Answer != nil {
						answered++
					} else {
						pending++
					}
				}

				task := store.Task
				if task == "" {
					task = "(no task)"
				}
				if len(task) > 60 {
					task = task[:57] + "..."
				}

				fmt.Printf("  %s\n", relDir)
				fmt.Printf("    Task: %s\n", task)
				fmt.Printf("    Decisions: %d total, %d answered, %d pending\n\n", len(store.Decisions), answered, pending)
				found++
			}
			return nil
		})
		if err != nil {
			return err
		}

		if found == 0 {
			fmt.Println("No defer sessions found.")
		} else {
			fmt.Printf("Found %d session(s).\n", found)
		}
		return nil
	},
}

var sessionsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the .defer/ session in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		deferDir := filepath.Join(cwd, ".defer")

		if _, err := os.Stat(deferDir); os.IsNotExist(err) {
			fmt.Println("No .defer/ directory found in the current directory.")
			return nil
		}

		fmt.Printf("This will delete %s and all session data.\n", deferDir)
		fmt.Print("Are you sure? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}

		if err := os.RemoveAll(deferDir); err != nil {
			return fmt.Errorf("failed to delete .defer/: %w", err)
		}

		// Also remove DECISIONS.md if it exists
		mdPath := filepath.Join(cwd, "DECISIONS.md")
		os.Remove(mdPath) // ignore error if not found

		fmt.Println("Session deleted.")
		return nil
	},
}

var sessionsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the current session's DECISIONS.md to stdout",
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

		// Generate markdown content
		var b strings.Builder

		b.WriteString("# DECISIONS.md\n\n")
		b.WriteString(fmt.Sprintf("> Task: %s\n\n", store.Task))

		var userDecs, aiDecs []decision.Decision
		for _, d := range store.Decisions {
			if d.Implicit {
				aiDecs = append(aiDecs, d)
			} else {
				userDecs = append(userDecs, d)
			}
		}

		b.WriteString("## Decisions\n\n")
		b.WriteString("| ID | Category | Question | Answer | Date |\n")
		b.WriteString("|----|----------|----------|--------|------|\n")
		for _, d := range userDecs {
			answer := "(pending)"
			if d.Answer != nil {
				if d.Delegated {
					answer = "DELEGATED: " + *d.Answer
				} else {
					answer = *d.Answer
				}
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", d.ID, d.Category, d.Question, answer, d.Date))
		}

		if len(aiDecs) > 0 {
			b.WriteString("\n## AI Choices\n\n")
			b.WriteString("| ID | Category | What was decided | Reasoning |\n")
			b.WriteString("|----|----------|------------------|----------|\n")
			for _, d := range aiDecs {
				answer := d.StrAnswer()
				if answer == "" {
					answer = d.Question
				}
				b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", d.ID, d.Category, answer, d.Reasoning))
			}
		}

		fmt.Print(b.String())
		return nil
	},
}

func init() {
	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsDeleteCmd)
	sessionsCmd.AddCommand(sessionsExportCmd)
	rootCmd.AddCommand(sessionsCmd)
}
