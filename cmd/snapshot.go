package cmd

import (
	"fmt"
	"regexp"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

var snapshotCmd = &cobra.Command{
	Use:    "snapshot",
	Short:  "Render TUI snapshots for visual testing (dev only)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build sample decisions
		decs := []decision.Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Backend framework?", Answer: decision.Ptr("Go with Gin"), Source: "auto", Impact: 9,
				Options: []decision.DecisionOption{{Key: "A", Label: "Go with Gin"}, {Key: "B", Label: "Express"}, {Key: "C", Label: "FastAPI"}}},
			{ID: "STA-0002", Category: "Stack", Question: "Frontend approach?", Answer: decision.Ptr("React + Vite"), Source: "user", Impact: 8,
				Options: []decision.DecisionOption{{Key: "A", Label: "React + Vite"}, {Key: "B", Label: "HTMX"}}},
			{ID: "DAT-0001", Category: "Data", Question: "Primary database?", Impact: 8, DependsOn: []string{"STA-0001"},
				Options: []decision.DecisionOption{{Key: "A", Label: "PostgreSQL"}, {Key: "B", Label: "SQLite"}, {Key: "C", Label: "MongoDB"}}},
			{ID: "AUT-0001", Category: "Auth", Question: "Authentication method?", Impact: 7,
				Options: []decision.DecisionOption{{Key: "A", Label: "JWT"}, {Key: "B", Label: "Session cookies"}, {Key: "C", Label: "OAuth2"}}},
			{ID: "DEP-0001", Category: "Deploy", Question: "Hosting platform?", Answer: decision.Ptr("Fly.io"), Source: "agent", Impact: 5,
				Options: []decision.DecisionOption{{Key: "A", Label: "Fly.io"}, {Key: "B", Label: "AWS"}, {Key: "C", Label: "Self-hosted"}}},
			{ID: "API-0001", Category: "API", Question: "Rate limiting strategy?", Source: "invalidated", Impact: 4,
				Options: []decision.DecisionOption{{Key: "A", Label: "Token bucket"}, {Key: "B", Label: "Sliding window"}}},
		}

		// Chat log entries
		chatLog := []tui.ChatEntry{
			{Type: "user", Text: "build a URL shortener with analytics"},
			{Type: "system", Text: "Analyzing project and identifying decisions..."},
			{Type: "topic", Text: "Explore codebase structure", Children: []tui.ChatEntry{
				{Type: "subtool", Text: "Read(package.json)"},
				{Type: "subtool", Text: "Read(go.mod)"},
				{Type: "subtool", Text: "Glob(src/**/*)"},
			}},
			{Type: "agent", Text: "Found 6 decisions across 5 categories. The project uses Go with Gin on the backend."},
			{Type: "topic", Text: "Write(cmd/main.go)"},
			{Type: "topic", Text: "Write(internal/api/routes.go)"},
			{Type: "action", Text: "Changed DAT-0001 → PostgreSQL"},
			{Type: "system", Text: "Implementation complete."},
		}

		// Render tree model at different widths
		for _, width := range []int{120, 80} {
			tree := tui.NewTreeModel()
			tree.SetDecisions(decs)
			tree.SetChatLog(chatLog)
			tree.SetSize(width, 40)
			tree.SetFocusPanel(tui.FocusTree)

			view := tree.View()
			clean := stripANSI(view)

			fmt.Printf("=== TREE VIEW (width=%d, focus=tree) ===\n", width)
			fmt.Println(clean)
			fmt.Println()

			// Switch to chat focus
			tree.SetFocusPanel(tui.FocusChat)
			view = tree.View()
			clean = stripANSI(view)

			fmt.Printf("=== TREE VIEW (width=%d, focus=chat) ===\n", width)
			fmt.Println(clean)
			fmt.Println()
		}

		// Render mascot moods
		fmt.Println("=== MASCOT MOODS ===")
		for _, mood := range []tui.MascotMood{tui.MoodIdle, tui.MoodActive, tui.MoodDone, tui.MoodError} {
			fmt.Printf("--- %d ---\n", mood)
			fmt.Println(stripANSI(tui.RenderMascot(mood, 0)))
			fmt.Println()
		}

		// Render onboarding wizard
		fmt.Println("=== ONBOARDING (provider select) ===")
		ob := tui.NewOnboardingModel()
		fmt.Println(stripANSI(ob.View()))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
}

// decisionSummaryForSnapshot creates a brief text summary for display
func decisionSummaryForSnapshot(decs []decision.Decision) string {
	answered, pending := 0, 0
	for _, d := range decs {
		if d.Answer != nil {
			answered++
		} else {
			pending++
		}
	}
	return fmt.Sprintf("%d total, %d answered, %d pending", len(decs), answered, pending)
}
