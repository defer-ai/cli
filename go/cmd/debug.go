package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// runDebug executes the full flow synchronously to stdout (no TUI).
func runDebug(task, modelName string, client *api.Client, ccProvider *api.ClaudeCodeProvider, cwd string) error {
	if task == "" {
		return fmt.Errorf("--debug requires a task argument")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := agent.NewManager(client, ccProvider, cwd)

	// --- Decomposition ---
	fmt.Printf("Decomposing: %s\n", task)

	decomposeDone := make(chan struct{})
	var decisions []decision.Decision

	mgr.StartDecomposition(ctx, task, func(ev agent.Event) {
		switch ev.Type {
		case agent.AgentDecisionsReady:
			decisions = ev.Decisions
			close(decomposeDone)
		case agent.AgentStateChanged:
			// Print progress dots
		}
	})

	<-decomposeDone
	fmt.Printf("Decomposition complete: %d decisions\n", len(decisions))
	for _, d := range decisions {
		fmt.Printf("  [%s] %s: %s\n", d.ID, d.Category, d.Question)
	}

	// --- Swarm ---
	fmt.Println("\nRunning swarm expansion...")

	swarmDone := make(chan struct{})
	var swarmCount int

	go func() {
		mgr.RunSwarm(ctx, task, decisions, func(ev agent.Event) {
			if ev.Type == agent.ExecDecisionStored && len(ev.Decisions) > 0 {
				swarmCount += len(ev.Decisions)
				for _, d := range ev.Decisions {
					fmt.Printf("  [swarm] [%s] %s: %s\n", d.ID, d.Category, d.Question)
				}
			}
		})
		close(swarmDone)
	}()

	<-swarmDone
	fmt.Printf("Swarm complete: %d sub-decisions added\n", swarmCount)

	// Merge swarm decisions into decisions list
	allDecs := mgr.AllDecisions()
	decisions = allDecs

	// --- Domain summary ---
	fmt.Println("\nDomain summary:")
	groups := agent.GroupByCategory(decisions)
	for cat, decs := range groups {
		fmt.Printf("  %s: %d decisions\n", cat, len(decs))
	}

	// --- Auto-decide ---
	fmt.Println("\nSetting all priorities to medium")
	priorities := make(map[string]agent.CareLevel)
	for cat := range groups {
		priorities[cat] = agent.CareLevelMedium
	}
	mgr.AutoDecide(priorities)
	decisions = mgr.Agent().Decisions()

	// --- Decision tree ---
	fmt.Println("\nDecision tree:")
	for _, d := range decisions {
		answer := "(pending)"
		if d.Answer != nil {
			answer = *d.Answer
			if d.Delegated {
				answer = "[delegated] " + answer
			}
		}
		src := d.Source
		if src == "" {
			src = "?"
		}
		fmt.Printf("  %s [%s] %s -> %s (source: %s)\n", d.ID, d.Category, d.Question, answer, src)
	}

	// --- Executor ---
	fmt.Println("\nLaunching executor")

	execDone := make(chan struct{})
	mgr.LaunchExecutors(ctx, task, decisions, priorities, func(ev agent.Event) {
		switch ev.Type {
		case agent.ExecStateChanged:
			for _, e := range mgr.Executors() {
				st := e.State()
				if st.Status == agent.DomainExecuting || st.Status == agent.DomainPlanning || st.Status == agent.DomainVerifying {
					status := st.Status.String()
					output := st.Output
					if len(output) > 200 {
						output = "..." + output[len(output)-200:]
					}
					output = strings.ReplaceAll(output, "\n", " ")
					if len(output) > 80 {
						output = output[:80] + "..."
					}
					fmt.Printf("  [%s] %s: %s\n", st.Domain, status, output)
				}
			}
		case agent.ExecDecisionStored:
			for _, d := range ev.Decisions {
				fmt.Printf("  [exec] [%s] %s: %s\n", d.ID, d.Category, d.Question)
			}
		case agent.AllExecutorsDone:
			close(execDone)
		}
	})

	<-execDone
	fmt.Println("\nDone")

	return nil
}
