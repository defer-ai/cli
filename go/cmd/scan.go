package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan existing project to discover decisions already made",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()

		p, err := api.ResolveProvider(provider, apiKey, model)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		if debug {
			return runScanDebug(p, cwd)
		}

		// Launch TUI in scan mode
		m := tui.NewScanModel(p, cwd)
		prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		_, err = prog.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScanDebug(scanProvider api.Provider, cwd string) error {
	fmt.Println("Scanning project at", cwd)
	ctx := context.Background()

	userPrompt := fmt.Sprintf(`Scan the project at %s. Start by running Glob to find all files, then Read the key config files (go.mod, package.json, tsconfig.json, Dockerfile, docker-compose.yml, etc.), then Read a few source files to understand the architecture. Output ALL discovered decisions as a `+"```defer-decisions```"+` JSON block.`, cwd)

	events := make(chan api.Event, 100)
	go scanProvider.RunCompletion(ctx, agent.ScanPrompt, userPrompt, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
		case api.EventError:
			if ev.Error != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Error)
			}
			goto done
		case api.EventDone:
			goto done
		}
	}
done:

	// Parse decisions
	decs := agent.ParseScanDecisions(fullText)
	if len(decs) == 0 {
		fmt.Println("No decisions discovered.")
		return nil
	}

	// Mark all as discovered (answered, implicit, source: discovered)
	today := time.Now().Format("2006-01-02")
	for i := range decs {
		if decs[i].Answer == nil {
			// Use the first option as the answer (it's what was discovered)
			if len(decs[i].Options) > 0 {
				answer := decs[i].Options[0].Label
				decs[i].Answer = &answer
			}
		}
		decs[i].Implicit = true
		decs[i].Source = "discovered"
		decs[i].Date = today
	}

	// Save to store
	store, _ := decision.LoadStore(cwd)
	if store == nil {
		store, _ = decision.CreateStore(cwd, "(scanned project)")
	}
	if store != nil {
		store.Decisions = append(store.Decisions, decs...)
		decision.SaveStore(cwd, store)
	}

	fmt.Printf("\nDiscovered %d decisions:\n", len(decs))
	for _, d := range decs {
		answer := "(unknown)"
		if d.Answer != nil {
			answer = *d.Answer
		}
		fmt.Printf("  [%s] %s → %s\n", d.Category, d.Question, answer)
	}

	fmt.Println("\nSaved to .defer/decisions.json")
	fmt.Println("Run `defer` to view and manage decisions, or `defer \"add feature X\"` to build on top.")
	return nil
}
