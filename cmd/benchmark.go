package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:    "benchmark [task]",
	Short:  "Benchmark decision tracking reliability (dev only)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		task := args[0]
		cwd, _ := os.Getwd()

		p, err := api.ResolveProvider(provider, apiKey, model)
		if err != nil {
			return err
		}

		return runBenchmark(task, p, cwd)
	},
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
}

func runBenchmark(task string, provider api.Provider, cwd string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	mgr := agent.NewManager(provider, cwd)

	// --- Phase 1: Decomposition ---
	fmt.Println("═══ PHASE 1: DECOMPOSITION ═══")
	fmt.Printf("Task: %s\n\n", task)

	decomposeDone := make(chan struct{})
	var decisions []decision.Decision

	mgr.StartDecomposition(ctx, task, func(ev agent.Event) {
		switch ev.Type {
		case agent.AgentDecisionsReady:
			decisions = ev.Decisions
			close(decomposeDone)
		}
	})

	select {
	case <-decomposeDone:
	case <-ctx.Done():
		return fmt.Errorf("decomposition timed out")
	}

	fmt.Printf("Decomposed into %d decisions:\n", len(decisions))
	for _, d := range decisions {
		status := "PENDING"
		if d.Answer != nil {
			status = fmt.Sprintf("PRE-ANSWERED: %s", *d.Answer)
		}
		fmt.Printf("  [%s] %-12s %s → %s\n", d.ID, d.Category, d.Question, status)
	}

	// --- Phase 2: Set mixed care levels ---
	fmt.Println("\n═══ PHASE 2: CARE LEVELS ═══")
	groups := agent.GroupByCategory(decisions)
	priorities := make(map[string]agent.CareLevel)

	// Set first category to "review", rest to "auto"
	first := true
	for cat := range groups {
		if first {
			priorities[cat] = agent.CareLevelReview
			fmt.Printf("  %-12s → REVIEW (will require user input)\n", cat)
			first = false
		} else {
			priorities[cat] = agent.CareLevelAuto
			fmt.Printf("  %-12s → auto\n", cat)
		}
	}

	mgr.AutoDecide(priorities)
	decisions = mgr.Agent().Decisions()

	pendingBefore := 0
	decidedBefore := 0
	for _, d := range decisions {
		if d.IsPending() {
			pendingBefore++
		} else {
			decidedBefore++
		}
	}
	fmt.Printf("\nAfter auto-decide: %d decided, %d pending\n", decidedBefore, pendingBefore)

	// --- Phase 3: Execution (auto-answer pending decisions after 5s) ---
	fmt.Println("\n═══ PHASE 3: EXECUTION ═══")
	fmt.Println("Launching executor. Pending decisions will be auto-answered after 5s.")
	fmt.Println("Monitoring for DECIDED/PENDING/RESEARCH protocol compliance...")
	fmt.Println()

	execDone := make(chan struct{})
	var execDecisions []decision.Decision
	toolCallCount := 0
	writeCount := 0
	decidedLineCount := 0
	pendingLineCount := 0

	// Continuously auto-answer pending decisions every 3s
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			allDecs := mgr.AllDecisions()
			answered := 0
			for i := range allDecs {
				if allDecs[i].IsPending() {
					answer := "auto-stress-test"
					if len(allDecs[i].Options) > 0 {
						answer = allDecs[i].Options[0].Label
					}
					allDecs[i].SetAnswer(answer, "user")
					answered++
				}
			}
			if answered > 0 {
				fmt.Printf("  [bench] Auto-answered %d pending decisions\n", answered)
				for _, e := range mgr.Executors() {
					select {
					case e.ContinueCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	mgr.LaunchExecutors(ctx, task, decisions, priorities, func(ev agent.Event) {
		switch ev.Type {
		case agent.ExecStateChanged:
			for _, e := range mgr.Executors() {
				st := e.State()
				output := st.Output
				// Count DECIDED/PENDING lines in output
				for _, line := range strings.Split(output, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "DECIDED:") {
						decidedLineCount++
					}
					if strings.HasPrefix(line, "PENDING:") {
						pendingLineCount++
					}
				}
			}

		case agent.ExecToolActivity:
			toolCallCount++
			desc := strings.ToLower(ev.ToolActivity)
			if strings.Contains(desc, "write") || strings.Contains(desc, "edit") || strings.Contains(desc, "bash") {
				writeCount++
			}

		case agent.ExecDecisionStored:
			for _, d := range ev.Decisions {
				execDecisions = append(execDecisions, d)
				status := "DECIDED"
				if d.IsPending() {
					status = "PENDING"
				}
				fmt.Printf("  [exec] %s [%s] %s: %s\n", status, d.ID, d.Category, d.Question)
			}

		case agent.ExecWaitingForDecisions:
			fmt.Printf("  [exec] ⏸ PAUSED — waiting for pending decisions\n")

		case agent.AllExecutorsDone:
			close(execDone)
		}
	})

	select {
	case <-execDone:
	case <-ctx.Done():
		fmt.Println("\n  [bench] Execution timed out")
	}

	// --- Phase 4: Report ---
	fmt.Println("\n═══ RESULTS ═══")

	allDecs := mgr.AllDecisions()
	totalDecs := len(allDecs)
	fromDecompose := len(decisions)
	fromExec := len(execDecisions)
	pending := 0
	decided := 0
	for _, d := range allDecs {
		if d.IsPending() {
			pending++
		} else {
			decided++
		}
	}

	fmt.Printf("Decisions:        %d total (%d from decompose, %d from executor)\n", totalDecs, fromDecompose, fromExec)
	fmt.Printf("Status:           %d decided, %d still pending\n", decided, pending)
	fmt.Printf("Tool calls:       %d total, %d writes\n", toolCallCount, writeCount)
	fmt.Printf("Protocol lines:   %d DECIDED, %d PENDING found in output\n", decidedLineCount, pendingLineCount)

	// Coverage analysis
	fmt.Println("\n═══ COVERAGE ANALYSIS ═══")
	if writeCount > 0 && fromExec == 0 {
		fmt.Println("⚠ PROBLEM: Executor made writes but reported ZERO decisions!")
		fmt.Println("  The agent is not following the DECIDED/PENDING/RESEARCH protocol.")
	} else if writeCount > 0 && float64(fromExec)/float64(writeCount) < 0.1 {
		fmt.Printf("⚠ LOW COVERAGE: %d writes but only %d decisions (%.0f%% ratio)\n",
			writeCount, fromExec, float64(fromExec)/float64(writeCount)*100)
		fmt.Println("  The agent is making many implicit decisions without documenting them.")
	} else if fromExec > 0 {
		fmt.Printf("✓ Coverage OK: %d decisions from %d writes (%.0f%% ratio)\n",
			fromExec, writeCount, float64(fromExec)/float64(writeCount)*100)
	}

	if pendingBefore > 0 {
		fmt.Printf("\nReview decisions: %d were set to REVIEW care level\n", pendingBefore)
		if decidedLineCount == 0 && pendingLineCount == 0 {
			fmt.Println("⚠ PROBLEM: No DECIDED/PENDING lines found in executor output at all!")
			fmt.Println("  The prompt may not be working, or lines are being split across chunks.")
		}
	}

	// Dump all final decisions
	fmt.Println("\n═══ ALL DECISIONS ═══")
	for _, d := range allDecs {
		answer := "(pending)"
		if d.Answer != nil {
			answer = *d.Answer
		}
		src := d.Source
		if src == "" {
			src = "?"
		}
		fmt.Printf("  [%s] %-12s %s → %s (source: %s, revisions: %d)\n",
			d.ID, d.Category, d.Question, answer, src, d.RevisionCount)
	}

	return nil
}
