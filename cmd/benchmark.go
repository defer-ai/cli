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

		// Apply --effort to the provider if it's a Claude Code provider.
		// Benchmark bypasses the normal TUI entry point that handles this,
		// so wire it up directly here.
		if cc, ok := p.(*api.ClaudeCodeProvider); ok && effort != "" {
			cc.SetEffort(effort)
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

	// --- Phase 2: Set care levels ---
	// All categories set to auto for deterministic measurement. Previously
	// the bench randomly flipped the first category (via Go map iteration)
	// to "review", which caused that category's inline DECIDED lines to be
	// demoted to PENDING by storeDecision's care-level filter — a huge
	// source of run-to-run variance in the "inline" metric. All-auto lets
	// every inline DECIDED land as intended.
	fmt.Println("\n═══ PHASE 2: CARE LEVELS ═══")
	groups := agent.GroupByCategory(decisions)
	priorities := make(map[string]agent.CareLevel)
	for cat := range groups {
		priorities[cat] = agent.CareLevelAuto
		fmt.Printf("  %-12s → auto\n", cat)
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
	// Track unique DECIDED/PENDING lines as they appear in agent output.
	// Each line counted exactly once, in the order it first appears.
	seenLines := map[string]bool{}
	var inlineDecidedOrder []int // for each DECIDED line, the toolCallCount at the moment it appeared
	var writePositions []int     // toolCallCount at the moment each Write/Edit happened
	var inlineDecidedCount int
	var inlinePendingCount int

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
				// Scan output for new DECIDED/PENDING lines (each counted once)
				for _, line := range strings.Split(st.Output, "\n") {
					line = strings.TrimSpace(line)
					if line == "" || seenLines[line] {
						continue
					}
					if strings.HasPrefix(line, "DECIDED:") {
						seenLines[line] = true
						inlineDecidedCount++
						inlineDecidedOrder = append(inlineDecidedOrder, toolCallCount)
					} else if strings.HasPrefix(line, "PENDING:") {
						seenLines[line] = true
						inlinePendingCount++
					}
				}
			}

		case agent.ExecToolActivity:
			toolCallCount++
			desc := strings.ToLower(ev.ToolActivity)
			// HumanDescription uses "Creating X" (Write), "Editing X" (Edit),
			// "Running: ..." (Bash). Match the verbs.
			isFileWrite := strings.HasPrefix(desc, "creating ") || strings.HasPrefix(desc, "editing ")
			if isFileWrite || strings.HasPrefix(desc, "running:") {
				writeCount++
			}
			if isFileWrite {
				// Record Write/Edit positions so we can check whether the
				// agent followed the tool-anchored protocol (DECIDED line
				// emitted between this write and the next tool call).
				writePositions = append(writePositions, toolCallCount)
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
	fmt.Printf("Tool calls:       %d total, %d writes/edits/bash\n", toolCallCount, writeCount)
	fmt.Printf("Inline reports:   %d DECIDED, %d PENDING (in agent's text stream)\n", inlineDecidedCount, inlinePendingCount)

	// === KEY METRICS ===
	fmt.Println("\n═══ INLINE DECISION METRICS ═══")
	// inline_ratio: how many of the final decisions were reported inline vs recovered by extract()
	inlineRatio := 0.0
	if fromExec > 0 {
		inlineRatio = float64(inlineDecidedCount) / float64(fromExec)
		if inlineRatio > 1.0 {
			inlineRatio = 1.0 // duplicate dedup may exceed
		}
	}
	fmt.Printf("inline_ratio:      %.2f  (inline DECIDED / total executor decisions)\n", inlineRatio)
	if writeCount > 0 {
		dpw := float64(inlineDecidedCount) / float64(writeCount)
		fmt.Printf("decisions_per_write: %.2f  (inline DECIDED / write actions)\n", dpw)
	}

	// Distribution: are inline decisions spread across the run, or batched?
	if len(inlineDecidedOrder) >= 2 && toolCallCount > 0 {
		// Compute median position (as fraction of run)
		// and "early half" vs "late half" counts.
		half := toolCallCount / 2
		early := 0
		late := 0
		for _, pos := range inlineDecidedOrder {
			if pos <= half {
				early++
			} else {
				late++
			}
		}
		fmt.Printf("distribution:      %d in first half / %d in second half of run\n", early, late)
		if late > 3*early {
			fmt.Println("                   ⚠ heavily batched at the end")
		} else if early > 3*late {
			fmt.Println("                   ⚠ heavily front-loaded (decisions before any work?)")
		} else {
			fmt.Println("                   ✓ reasonably distributed")
		}
	}

	// Coverage analysis
	fmt.Println("\n═══ COVERAGE ANALYSIS ═══")
	if writeCount > 0 && fromExec == 0 {
		fmt.Println("⚠ PROBLEM: Executor made writes but reported ZERO decisions!")
		fmt.Println("  The agent is not following the DECIDED/PENDING/RESEARCH protocol.")
	} else if writeCount > 0 && float64(fromExec)/float64(writeCount) < 0.5 {
		fmt.Printf("⚠ LOW COVERAGE: %d writes but only %d decisions (%.0f%% ratio)\n",
			writeCount, fromExec, float64(fromExec)/float64(writeCount)*100)
	} else if fromExec > 0 {
		fmt.Printf("✓ Coverage OK: %d decisions from %d writes (%.0f%% ratio)\n",
			fromExec, writeCount, float64(fromExec)/float64(writeCount)*100)
	}

	// Per-quartile breakdown — finer-grained than the binary "first half /
	// second half" check above. Lets us tell "constant flow" from
	// "front-loaded" from "tail-batched".
	if len(inlineDecidedOrder) >= 1 && toolCallCount > 0 {
		quartiles := [4]int{}
		for _, pos := range inlineDecidedOrder {
			q := (pos * 4) / toolCallCount
			if q > 3 {
				q = 3
			}
			quartiles[q]++
		}
		fmt.Printf("quartiles:         Q1=%d Q2=%d Q3=%d Q4=%d (inline DECIDED count per quartile of run)\n",
			quartiles[0], quartiles[1], quartiles[2], quartiles[3])
	}

	// Tool-anchored behavior: for each Write/Edit, did a DECIDED line appear
	// before the next tool call? This is the key metric for the prompt
	// protocol the "anchor" and "full" variants target — it measures real
	// between-tool-call narration, not batching in planning blocks.
	if len(writePositions) > 0 {
		decidedAtPos := map[int]int{}
		for _, pos := range inlineDecidedOrder {
			decidedAtPos[pos]++
		}
		anchored := 0
		totalAnchoredDecided := 0
		for _, wp := range writePositions {
			if n, ok := decidedAtPos[wp]; ok && n > 0 {
				anchored++
				totalAnchoredDecided += n
			}
		}
		anchorRatio := float64(anchored) / float64(len(writePositions))
		fmt.Printf("tool_anchored:     %d/%d writes (%.0f%%) had a DECIDED emitted before the next tool call\n",
			anchored, len(writePositions), anchorRatio*100)
		fmt.Printf("anchored_decided:  %d DECIDED lines landed between a write and the next tool\n",
			totalAnchoredDecided)
	}

	// Inline positions as a comma-separated list of toolCallCount values at
	// the moment each DECIDED line was first seen. Lets external tools
	// reconstruct the full distribution.
	posStrs := make([]string, 0, len(inlineDecidedOrder))
	for _, pos := range inlineDecidedOrder {
		posStrs = append(posStrs, fmt.Sprintf("%d", pos))
	}
	fmt.Printf("inline_positions:  %s (out of %d tool calls)\n",
		strings.Join(posStrs, ","), toolCallCount)

	// Machine-readable summary line for scripting
	anchoredWrites := 0
	anchoredDecidedLines := 0
	if len(writePositions) > 0 {
		decidedAtPos := map[int]int{}
		for _, pos := range inlineDecidedOrder {
			decidedAtPos[pos]++
		}
		for _, wp := range writePositions {
			if n, ok := decidedAtPos[wp]; ok && n > 0 {
				anchoredWrites++
				anchoredDecidedLines += n
			}
		}
	}
	anchorRatio := 0.0
	if len(writePositions) > 0 {
		anchorRatio = float64(anchoredWrites) / float64(len(writePositions))
	}
	fmt.Printf("\nBENCH_RESULT total=%d inline=%d writes=%d file_writes=%d tools=%d inline_ratio=%.3f decisions_per_write=%.3f anchored_writes=%d anchored_decided=%d anchor_ratio=%.3f\n",
		fromExec, inlineDecidedCount, writeCount, len(writePositions), toolCallCount, inlineRatio,
		func() float64 {
			if writeCount == 0 {
				return 0
			}
			return float64(inlineDecidedCount) / float64(writeCount)
		}(),
		anchoredWrites, anchoredDecidedLines, anchorRatio,
	)

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
