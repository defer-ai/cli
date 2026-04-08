package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// inlineDecisionRe matches patterns like "DECISION: @STA-0001 = Go with Gin" in executor output.
var inlineDecisionRe = regexp.MustCompile(`DECISION:\s*@?([A-Z]+-\d+)\s*=\s*(.+)`)

// New decision protocol regexes
var decidedRe = regexp.MustCompile(`DECIDED:\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*([^|]+)\|\s*(.+)`)
var pendingRe = regexp.MustCompile(`PENDING:\s*([^|]+)\|\s*([^|]+)\|\s*(.+)\|\s*(.+)`)
var researchRe = regexp.MustCompile(`RESEARCH:\s*([^|]+)\|\s*(.+)`)


// DomainStatus tracks executor progress.
type DomainStatus int

const (
	DomainPending   DomainStatus = iota
	DomainPlanning
	DomainWaiting   // paused, waiting for user to resolve pending decisions
	DomainExecuting
	DomainVerifying
	DomainDone
	DomainError
)

func (s DomainStatus) String() string {
	switch s {
	case DomainPlanning:  return "planning"
	case DomainWaiting:   return "waiting"
	case DomainExecuting: return "executing"
	case DomainVerifying: return "verifying"
	case DomainDone:      return "done"
	case DomainError:     return "error"
	default:              return "pending"
	}
}

// ExecState is the immutable snapshot of a domain executor.
type ExecState struct {
	ID        string
	Domain    string
	CareLevel CareLevel
	Status    DomainStatus
	Output    string
	Error     string
}

// Executor implements one domain.
type Executor struct {
	mu              sync.Mutex
	provider        api.Provider
	cwd             string
	task            string
	domain          string
	careLevel       CareLevel
	priorities       map[string]CareLevel // per-category care levels
	decisions        []decision.Decision
	allDecisions     *[]decision.Decision
	knownCategories  []string
	state            ExecState
	onEvent          func(Event)
	researchResults   []string      // collected research findings to inject in next round
	codebaseManifest  string        // pre-scanned project structure
	ContinueCh        chan struct{} // TUI signals this when all pending decisions are resolved
}

// ExecOpts configures a new executor.
type ExecOpts struct {
	Provider          api.Provider
	CWD               string
	Task              string
	Domain            string
	CareLevel         CareLevel
	Priorities        map[string]CareLevel
	Decisions         []decision.Decision
	AllDecisions      *[]decision.Decision
	OnEvent           func(Event)
	CodebaseManifest  string // pre-scanned project structure to avoid re-exploration
}

// NewExecutor creates a domain executor.
func NewExecutor(opts ExecOpts) *Executor {
	// Extract canonical categories from initial decisions
	catSet := make(map[string]bool)
	var cats []string
	for _, d := range opts.Decisions {
		lower := strings.ToLower(strings.TrimSpace(d.Category))
		if !catSet[lower] {
			catSet[lower] = true
			cats = append(cats, d.Category)
		}
	}

	prios := make(map[string]CareLevel)
	for k, v := range opts.Priorities {
		prios[strings.ToLower(strings.TrimSpace(k))] = v
	}

	return &Executor{
		provider:        opts.Provider,
		cwd:             opts.CWD,
		task:            opts.Task,
		domain:          opts.Domain,
		careLevel:       opts.CareLevel,
		priorities:      prios,
		decisions:       opts.Decisions,
		allDecisions:    opts.AllDecisions,
		knownCategories: cats,
		onEvent:          opts.OnEvent,
		codebaseManifest: opts.CodebaseManifest,
		ContinueCh:       make(chan struct{}, 1),
		state: ExecState{
			ID:        fmt.Sprintf("domain-%s", opts.Domain),
			Domain:    opts.Domain,
			CareLevel: opts.CareLevel,
			Status:    DomainPending,
		},
	}
}

// State returns an immutable snapshot.
func (e *Executor) State() ExecState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *Executor) setStatus(s DomainStatus, output, errMsg string) {
	e.mu.Lock()
	e.state.Status = s
	if output != "" {
		e.state.Output = output
	}
	e.state.Error = errMsg
	e.mu.Unlock()
	e.onEvent(Event{Type: ExecStateChanged, ExecutorID: e.state.ID})
}

// Execute runs the full domain lifecycle as an iterative loop.
// The agent implements in chunks, pausing whenever decisions need resolution.
func (e *Executor) Execute(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			e.setStatus(DomainError, "", fmt.Sprintf("panic: %v", r))
		}
	}()

	// Wait for any existing pending decisions first
	if e.waitForPendingDecisions(ctx) {
		return
	}

	// Iterative execution: implement → pause for decisions → continue
	// Each round gets fresh context (updated decisions + existing files)
	var fullOutput string
	for round := 0; round < 5; round++ {
		decSummary := e.decisionSummary()

		// Inject research findings from previous round
		if len(e.researchResults) > 0 {
			decSummary += "\n\nResearch findings:\n" + strings.Join(e.researchResults, "\n")
			e.researchResults = nil
		}

		prevDecCount := len(*e.allDecisions)

		e.setStatus(DomainExecuting, fmt.Sprintf("Implementing (round %d)...", round+1), "")
		roundOutput := e.execute(ctx, decSummary)
		fullOutput += roundOutput

		if e.state.Status == DomainError {
			return
		}


		// Check if new decisions appeared (inline + extracted)
		newDecCount := len(*e.allDecisions)
		if newDecCount > prevDecCount {
			e.onEvent(Event{
				Type:         ExecToolActivity,
				ExecutorID:   e.state.ID,
				ToolActivity: fmt.Sprintf("Round %d: %d new decisions discovered", round+1, newDecCount-prevDecCount),
			})
		}

		// Pause for any pending decisions
		if e.waitForPendingDecisions(ctx) {
			return
		}

		// If no new decisions and the agent said "Implementation complete", we're done
		if newDecCount == prevDecCount && strings.Contains(strings.ToLower(roundOutput), "implementation complete") {
			break
		}

		// If no new decisions but implementation isn't complete, continue
		if newDecCount == prevDecCount {
			break // avoid infinite loops — agent didn't produce more decisions
		}
	}

	// Verification
	finalDecSummary := e.decisionSummary()
	e.setStatus(DomainVerifying, "Verifying...", "")
	issues := e.verify(ctx, fullOutput, finalDecSummary)
	if issues != "" {
		e.setStatus(DomainExecuting, "Fixing issues...", "")
		fullOutput = e.fix(ctx, issues, finalDecSummary)
	}

	// Phase 4: Extract implicit decisions
	e.extract(ctx, fullOutput)

	// Phase 5: Build verification
	e.setStatus(DomainVerifying, "Build check...", "")
	ok, buildOutput := e.verifyBuild(ctx)
	if ok {
		fullOutput += "\n[build verification: PASS]"
	} else if buildOutput != "" {
		fullOutput += "\n[build verification: FAIL]\n" + buildOutput
	}

	e.setStatus(DomainDone, fullOutput, "")
}

// waitForPendingDecisions checks if any decisions are pending (unanswered).
// If so, pauses the executor and waits for the TUI to signal ContinueCh.
// Returns true if the context was cancelled (executor should stop).
func (e *Executor) waitForPendingDecisions(ctx context.Context) bool {
	pending := 0
	e.mu.Lock()
	for _, d := range *e.allDecisions {
		if d.Answer == nil {
			pending++
		}
	}
	e.mu.Unlock()

	if pending == 0 {
		return false // nothing to wait for
	}

	// Signal the TUI that we're waiting
	e.setStatus(DomainWaiting, fmt.Sprintf("Waiting for %d pending decisions...", pending), "")
	e.onEvent(Event{Type: ExecWaitingForDecisions, ExecutorID: e.state.ID})

	// Block until the TUI signals all decisions are resolved, or context cancelled
	select {
	case <-ctx.Done():
		return true
	case <-e.ContinueCh:
		return false
	}
}

func (e *Executor) decisionSummary() string {
	var lines []string
	for _, d := range e.decisions {
		if d.Answer == nil {
			continue // skip pending decisions -- don't implement things the user hasn't decided
		}
		answer := *d.Answer
		lines = append(lines, fmt.Sprintf("%s: %s -> %s", d.ID, d.Question, answer))
	}
	return strings.Join(lines, "\n")
}

// freshProvider creates a new provider for isolated sessions.
// For ClaudeCodeProvider, creates a fresh subprocess; for stateless HTTP providers, reuses the provider.
//
// The fresh Claude Code provider is always started in StrictMode for the
// executor phase: Bash is removed from the toolkit, a PreToolUse hook on
// Write/Edit emits a DECIDED-before-next-tool reminder, and the system
// prompt gets an appendix explaining the restriction. A 5×2 Flask bench
// showed this improves inline narration (mean 25→30 DECIDED per run),
// pins tool-anchored ratio at 14% (vs 0% plain) and makes the executor
// ~14% faster by eliminating bash-bypass roundtrips.
func (e *Executor) freshProvider() api.Provider {
	if orig, ok := e.provider.(*api.ClaudeCodeProvider); ok {
		cp := api.NewClaudeCodeProviderWithCWD(e.provider.GetModel(), e.cwd)
		cp.Effort = orig.Effort
		cp.StrictMode = true
		return cp
	}
	return e.provider // stateless HTTP providers can be reused
}

// simpleCompletion runs a one-shot completion, forwarding tool events to the feed.
func (e *Executor) simpleCompletion(ctx context.Context, systemPrompt, userMsg string) (string, error) {
	cp := e.freshProvider()
	events := make(chan api.Event, 100)
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)
	var text string
	for ev := range events {
		if ev.Type == api.EventTextDelta {
			text += ev.Text
		}
		// Forward tool calls to the feed
		if ev.Type == api.EventToolCallStart && ev.ToolCall != nil {
			e.onEvent(Event{
				Type:         ExecToolActivity,
				ExecutorID:   e.state.ID,
				ToolActivity: ev.ToolCall.HumanDescription(),
			})
		}
		// Forward permission requests to the TUI
		if ev.Type == api.EventPermissionRequest && ev.PermissionReq != nil {
			e.onEvent(Event{
				Type:          ExecPermissionRequest,
				ExecutorID:    e.state.ID,
				PermissionReq: ev.PermissionReq,
			})
		}
		if ev.Type == api.EventDone || ev.Type == api.EventError {
			if ev.Error != nil {
				return text, ev.Error
			}
			break
		}
	}
	return text, nil
}


func (e *Executor) execute(ctx context.Context, decSummary string) string {
	systemPrompt := fmt.Sprintf(ExecutePromptForVariant(), e.domain, CarePrompts[e.careLevel])

	// Build care level context for the prompt
	var careLevelCtx strings.Builder
	for cat, level := range e.priorities {
		careLevelCtx.WriteString(fmt.Sprintf("%s: %s\n", cat, string(level)))
	}

	// Build codebase manifest context
	manifestCtx := ""
	if e.codebaseManifest != "" {
		manifestCtx = "\n\nExisting project structure (DO NOT re-explore these — they are already known):\n" + e.codebaseManifest + "\n"
	}

	userMsg := fmt.Sprintf("Task: %s\n\nProject directory: %s\nDomain: %s\n\nCare levels per domain:\n%s\nDecisions:\n%s%s\nContinue from where the previous round left off. Do not recreate existing files.\n\nImplement the %s domain now. All files must go in %s or its subdirectories.",
		e.task, e.cwd, e.domain, careLevelCtx.String(), decSummary, manifestCtx, e.domain, e.cwd)

	events := make(chan api.Event, 100)

	// Use a fresh provider per domain (isolated sessions)
	cp := e.freshProvider()
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)

	var fullText string
	var lineBuf string       // accumulates text until newline for line-based parsing
	var writeCount int       // tracks write/edit tool calls this round
	prevDecCount := len(*e.allDecisions)

	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
			lineBuf += ev.Text
			e.mu.Lock()
			if len(fullText) > 100000 {
				e.state.Output = "...(truncated)\n" + fullText[len(fullText)-100000:]
			} else {
				e.state.Output = fullText
			}
			e.mu.Unlock()
			e.onEvent(Event{Type: ExecStateChanged, ExecutorID: e.state.ID})

			// Process complete lines from the buffer
			for {
				idx := strings.Index(lineBuf, "\n")
				if idx == -1 {
					break
				}
				line := lineBuf[:idx]
				lineBuf = lineBuf[idx+1:]
				e.processDecisionLine(line)
			}

		case api.EventToolCallStart:
			// Track write operations to detect under-reporting
			if ev.ToolCall != nil && ev.ToolCall.IsMajorAction() {
				writeCount++
			}
			if ev.ToolCall != nil {
				e.onEvent(Event{
					Type:         ExecToolActivity,
					ExecutorID:   e.state.ID,
					ToolActivity: ev.ToolCall.HumanDescription(),
				})
			}

		case api.EventPermissionRequest:
			if ev.PermissionReq != nil {
				e.onEvent(Event{
					Type:          ExecPermissionRequest,
					ExecutorID:    e.state.ID,
					PermissionReq: ev.PermissionReq,
				})
			}

		case api.EventError:
			e.setStatus(DomainError, fullText, ev.Error.Error())
			return fullText

		case api.EventDone:
			// Flush any remaining partial line
			if lineBuf != "" {
				e.processDecisionLine(lineBuf)
				lineBuf = ""
			}

			// Post-round enforcement: if the agent made write actions but
			// reported no decisions, run a lightweight extraction pass to
			// recover the decisions it should have reported inline.
			newDecCount := len(*e.allDecisions) - prevDecCount
			if writeCount > 0 && newDecCount == 0 {
				e.onEvent(Event{
					Type:         ExecToolActivity,
					ExecutorID:   e.state.ID,
					ToolActivity: fmt.Sprintf("Agent wrote %d files but reported 0 decisions — extracting...", writeCount),
				})
				e.extract(ctx, fullText)
			}

			return fullText
		}
	}
	return fullText
}

// processDecisionLine parses a single complete line for DECIDED/PENDING/RESEARCH/DECISION patterns.
func (e *Executor) processDecisionLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// DECIDED: category | question | answer | alternatives | reasoning
	if matches := decidedRe.FindStringSubmatch(line); len(matches) == 6 {
		category := strings.TrimSpace(matches[1])
		question := strings.TrimSpace(matches[2])
		answer := strings.TrimSpace(matches[3])
		alternatives := strings.TrimSpace(matches[4])
		reasoning := strings.TrimSpace(matches[5])

		opts := parseAlternatives(answer, alternatives)
		d := decision.Decision{
			ID:        decision.NextID(*e.allDecisions, category),
			Category:  e.normalizeCategoryLocked(category),
			Question:  question,
			Answer:    &answer,
			Options:   opts,
			Reasoning: reasoning,
			Source:    "agent",
			Impact:    3,
			Date:      time.Now().Format("2006-01-02"),
		}
		d.MarkCreated()
		d.OriginalSource = "agent"
		d.AnsweredAt = time.Now().UTC().Format(time.RFC3339)
		e.storeDecisionAndSave(d)
		return
	}

	// PENDING: category | question | A) opt1, B) opt2 | context
	if matches := pendingRe.FindStringSubmatch(line); len(matches) == 5 {
		category := strings.TrimSpace(matches[1])
		question := strings.TrimSpace(matches[2])
		optionsStr := strings.TrimSpace(matches[3])
		ctx := strings.TrimSpace(matches[4])

		opts := parsePendingOptions(optionsStr)
		d := decision.Decision{
			ID:       decision.NextID(*e.allDecisions, category),
			Category: e.normalizeCategoryLocked(category),
			Question: question,
			Options:  opts,
			Context:  ctx,
			Source:   "agent",
			Date:     time.Now().Format("2006-01-02"),
		}
		d.MarkCreated()
		e.storeDecisionAndSave(d)
		return
	}

	// RESEARCH: question | what to investigate
	if matches := researchRe.FindStringSubmatch(line); len(matches) == 3 {
		query := strings.TrimSpace(matches[1]) + " " + strings.TrimSpace(matches[2])
		responseCh := make(chan string, 1)
		e.onEvent(Event{
			Type:               ExecResearchRequest,
			ExecutorID:         e.state.ID,
			ResearchQuery:      query,
			ResearchResponseCh: responseCh,
		})
		go func() {
			result := <-responseCh
			e.mu.Lock()
			e.researchResults = append(e.researchResults, result)
			e.mu.Unlock()
		}()
		return
	}

	// DECISION: @STA-0001 = answer (inline update)
	e.scanInlineDecisions(line)
}

func (e *Executor) verify(ctx context.Context, output, decSummary string) string {
	truncated := output
	if len(truncated) > 6000 {
		truncated = "..." + truncated[len(truncated)-6000:]
	}
	msg := fmt.Sprintf("Domain: %s\nTask: %s\nDecisions:\n%s\n\nImplementation:\n%s",
		e.domain, e.task, decSummary, truncated)

	resp, err := e.simpleCompletion(ctx, VerifyPromptForVariant(), msg)
	if err != nil {
		return ""
	}

	if strings.Contains(resp, "NEEDS FIX") {
		return resp
	}
	return ""
}

func (e *Executor) fix(ctx context.Context, issues, decSummary string) string {
	systemPrompt := fmt.Sprintf(ExecutePromptForVariant(), e.domain, CarePrompts[e.careLevel])
	userMsg := fmt.Sprintf("Verification found issues:\n%s\n\nFix these now.", issues)

	events := make(chan api.Event, 100)
	cp := e.freshProvider()
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
		case api.EventPermissionRequest:
			if ev.PermissionReq != nil {
				e.onEvent(Event{
					Type:          ExecPermissionRequest,
					ExecutorID:    e.state.ID,
					PermissionReq: ev.PermissionReq,
				})
			}
		case api.EventDone, api.EventError:
			return fullText
		}
	}
	return fullText
}

func (e *Executor) extract(ctx context.Context, output string) {
	truncated := output
	if len(truncated) > 4000 {
		truncated = "..." + truncated[len(truncated)-4000:]
	}
	msg := fmt.Sprintf("Domain: %s\n\nImplementation output:\n%s", e.domain, truncated)

	resp, err := e.simpleCompletion(ctx, ExtractPromptForVariant(), msg)
	if err != nil {
		return
	}

	decs := e.parseImplicitChoices(resp)
	e.storeDecisions(decs)
}

// verifyBuild detects project type and runs build/test commands.
func (e *Executor) verifyBuild(ctx context.Context) (bool, string) {
	type check struct {
		marker  string
		buildCmd string
		testCmd  string
	}

	checks := []check{
		{"go.mod", "go build ./...", "go test ./... -count=1"},
		{"package.json", "npm run build", "npm test"},
		{"Makefile", "make", ""},
	}

	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(e.cwd, c.marker)); err != nil {
			continue
		}

		var results []string
		allOk := true

		if c.buildCmd != "" {
			buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			cmd := exec.CommandContext(buildCtx, "sh", "-c", c.buildCmd)
			cmd.Dir = e.cwd
			out, err := cmd.CombinedOutput()
			cancel()
			if err != nil {
				allOk = false
				output := string(out)
				if len(output) > 2000 {
					output = output[len(output)-2000:]
				}
				results = append(results, fmt.Sprintf("build (%s): FAIL\n%s", c.buildCmd, output))
			} else {
				results = append(results, fmt.Sprintf("build (%s): OK", c.buildCmd))
			}
		}

		if c.testCmd != "" {
			testCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
			cmd := exec.CommandContext(testCtx, "sh", "-c", c.testCmd)
			cmd.Dir = e.cwd
			out, err := cmd.CombinedOutput()
			cancel()
			if err != nil {
				allOk = false
				output := string(out)
				if len(output) > 2000 {
					output = output[len(output)-2000:]
				}
				results = append(results, fmt.Sprintf("test (%s): FAIL\n%s", c.testCmd, output))
			} else {
				results = append(results, fmt.Sprintf("test (%s): OK", c.testCmd))
			}
		}

		return allOk, strings.Join(results, "\n")
	}

	return true, "" // no build system detected, pass by default
}

func (e *Executor) storeDecision(d decision.Decision) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Strip trailing question mark from question text
	d.Question = strings.TrimRight(d.Question, "?")
	d.Question = strings.TrimSpace(d.Question)

	// Normalize category
	d.Category = e.normalizeCategoryLocked(d.Category)

	// Dedup by question. When a match is found, we don't silently drop the
	// new decision — we reconcile. The incoming decision usually comes from
	// the extract phase, which observes what the code ACTUALLY does, while
	// the existing one often came from the decompose phase, which recorded
	// the AGENT'S PLAN. The two can diverge (e.g. decompose said "Go 1.22
	// latest stable", the agent actually wrote "go 1.21"). In that case the
	// extract value is ground truth and should overwrite the stale plan.
	newQ := normalizeQuestion(d.Question)
	for i := range *e.allDecisions {
		existQ := normalizeQuestion((*e.allDecisions)[i].Question)
		matched := existQ == newQ || questionsOverlap(newQ, existQ)
		if !matched {
			continue
		}
		existing := &(*e.allDecisions)[i]
		// If the new decision has a concrete answer that differs from the
		// existing one, treat it as ground truth and update in place.
		if d.Answer != nil && *d.Answer != "" {
			newAns := *d.Answer
			existingAns := ""
			if existing.Answer != nil {
				existingAns = *existing.Answer
			}
			if existingAns != newAns {
				existing.SetAnswer(newAns, "agent")
				if d.Reasoning != "" {
					existing.Reasoning = d.Reasoning
				}
				// Emit a stored event so listeners see the reconciliation.
				e.onEvent(Event{
					Type:       ExecDecisionStored,
					ExecutorID: e.state.ID,
					Decisions:  []decision.Decision{*existing},
				})
			}
		}
		return
	}

	// Regenerate ID
	d.ID = decision.NextID(*e.allDecisions, d.Category)
	d.MarkCreated()

	// Apply care level
	level := e.getCareLevel(d.Category)
	switch level {
	case CareLevelReview:
		// Review: clear answer, set pending for user
		d.Answer = nil
		d.Delegated = false
		d.Source = "agent"
	case CareLevelAuto:
		// Auto: keep the auto answer as-is; track metadata if answered
		if d.Answer != nil && d.OriginalSource == "" {
			d.OriginalSource = d.Source
			if d.OriginalSource == "" {
				d.OriginalSource = "auto"
			}
		}
	}

	*e.allDecisions = append(*e.allDecisions, d)
	e.onEvent(Event{Type: ExecDecisionStored, ExecutorID: e.state.ID, Decisions: []decision.Decision{d}})
}

// getCareLevel returns the care level for a category.
func (e *Executor) getCareLevel(category string) CareLevel {
	key := strings.ToLower(strings.TrimSpace(category))
	if level, ok := e.priorities[key]; ok {
		return level
	}
	if e.careLevel != "" {
		return e.careLevel
	}
	return CareLevelAuto // default
}

func (e *Executor) normalizeCategoryLocked(cat string) string {
	lower := strings.ToLower(strings.TrimSpace(cat))

	// 1. Exact match against known categories from decomposition
	for _, known := range e.knownCategories {
		if strings.EqualFold(strings.TrimSpace(known), lower) {
			return known
		}
	}

	// 2. Substring match: "cli interface" matches "CLI", "data storage" matches "Storage"
	for _, known := range e.knownCategories {
		kl := strings.ToLower(strings.TrimSpace(known))
		if strings.Contains(lower, kl) || strings.Contains(kl, lower) {
			return known
		}
	}

	// 3. Word overlap: find category with most matching words
	catWords := strings.Fields(lower)
	bestMatch := ""
	bestScore := 0
	for _, known := range e.knownCategories {
		knownWords := strings.Fields(strings.ToLower(known))
		score := 0
		for _, cw := range catWords {
			for _, kw := range knownWords {
				if cw == kw || strings.HasPrefix(cw, kw) || strings.HasPrefix(kw, cw) {
					score++
				}
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = known
		}
	}
	if bestScore > 0 {
		return bestMatch
	}

	// 4. No match found -- use "Misc" if it exists, otherwise first known non-Misc category
	for _, known := range e.knownCategories {
		if strings.EqualFold(known, "Misc") {
			return known
		}
	}
	if len(e.knownCategories) > 0 {
		return e.knownCategories[0]
	}
	return "Misc"
}

// normalizeQuestion strips parenthetical references and normalizes for dedup comparison.
func normalizeQuestion(q string) string {
	// Remove parenthetical references like "(LAY-0037 — explicitly pending)"
	result := q
	for {
		start := strings.LastIndex(result, "(")
		if start < 0 {
			break
		}
		end := strings.Index(result[start:], ")")
		if end < 0 {
			break
		}
		inner := result[start+1 : start+end]
		// Only strip if it looks like a reference (contains an ID pattern or "pending"/"explicit")
		if strings.Contains(inner, "-") || strings.Contains(strings.ToLower(inner), "pending") || strings.Contains(strings.ToLower(inner), "explicit") {
			result = result[:start] + result[start+end+1:]
		} else {
			break
		}
	}
	// Collapse multiple spaces into one
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	// Remove space before trailing punctuation
	result = strings.TrimSpace(result)
	for _, p := range []string{" ?", " !", " ."} {
		result = strings.ReplaceAll(result, p, strings.TrimSpace(p))
	}
	return strings.ToLower(strings.TrimSpace(result))
}

// questionsOverlap returns true if two normalized questions share 85%+ of their significant words.
// Ignores common stop words to focus on meaningful terms.
func questionsOverlap(a, b string) bool {
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "for": true, "to": true,
		"of": true, "in": true, "on": true, "is": true, "be": true,
		"and": true, "or": true, "what": true, "which": true, "how": true,
		"should": true, "we": true, "use": true, "with": true, "this": true,
	}

	wordsA := significantWords(a, stopWords)
	wordsB := significantWords(b, stopWords)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return false
	}

	// Count overlapping words
	setB := make(map[string]bool, len(wordsB))
	for _, w := range wordsB {
		setB[w] = true
	}
	overlap := 0
	for _, w := range wordsA {
		if setB[w] {
			overlap++
		}
	}

	// Check overlap ratio against the smaller set
	smaller := len(wordsA)
	if len(wordsB) < smaller {
		smaller = len(wordsB)
	}
	return float64(overlap)/float64(smaller) >= 0.85
}

func significantWords(s string, stop map[string]bool) []string {
	var words []string
	for _, w := range strings.Fields(s) {
		w = strings.Trim(w, "?.,!:;\"'")
		if len(w) > 1 && !stop[w] {
			words = append(words, w)
		}
	}
	return words
}

func (e *Executor) storeDecisionAndSave(d decision.Decision) {
	e.storeDecision(d) // existing method (dedup, normalize, care level)

	// Immediately persist to disk
	store, _ := decision.LoadStore(e.cwd)
	if store == nil {
		store, _ = decision.CreateStore(e.cwd, e.task)
	}
	if store != nil {
		e.mu.Lock()
		store.Decisions = *e.allDecisions
		e.mu.Unlock()
		decision.SaveStore(e.cwd, store)
	}
}

func (e *Executor) storeDecisions(decs []decision.Decision) {
	for _, d := range decs {
		e.storeDecision(d)
	}
}

// UpdateDecision finds a decision in allDecisions by ID and updates its answer.
// Sets the source to "agent". Returns true if the decision was found and updated.
func (e *Executor) UpdateDecision(id string, answer string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range *e.allDecisions {
		if (*e.allDecisions)[i].ID == id {
			(*e.allDecisions)[i].SetAnswer(answer, "agent")
			return true
		}
	}
	return false
}

// scanInlineDecisions scans text for patterns like "DECISION: STA-0001 = Go with Gin"
// and calls UpdateDecision for each match.
func (e *Executor) scanInlineDecisions(text string) {
	matches := inlineDecisionRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) == 3 {
			id := strings.TrimSpace(m[1])
			answer := strings.TrimSpace(m[2])
			if id != "" && answer != "" {
				// IDs are stored with @ prefix
				if !strings.HasPrefix(id, "@") {
					// IDs stored without prefix
				}
				e.UpdateDecision(id, answer)
			}
		}
	}
}

func (e *Executor) parseImplicitChoices(text string) []decision.Decision {
	var result []decision.Decision
	today := time.Now().Format("2006-01-02")

	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		var raw []struct {
			Category  string `json:"category"`
			Decision  string `json:"decision"`
			Question  string `json:"question"`
			Options   []struct {
				Key   string `json:"key"`
				Label string `json:"label"`
			} `json:"options"`
			Answer    string   `json:"answer"`
			Reasoning string   `json:"reasoning"`
			Features  []string `json:"features"`
			Impact    int      `json:"impact"`
		}
		if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err == nil {
			for _, item := range raw {
				cat := e.normalizeCategoryLocked(item.Category)

				q := strings.TrimSpace(item.Question)
				if q == "" {
					q = strings.TrimSpace(item.Decision)
				}
				if q == "" {
					continue
				}

				// Build options
				var opts []decision.DecisionOption
				for _, o := range item.Options {
					if o.Label != "" {
						key := o.Key
						if key == "" {
							key = string(rune('A' + len(opts)))
						}
						opts = append(opts, decision.DecisionOption{Key: key, Label: o.Label})
					}
				}

				// Resolve the answer: "answer" field is the KEY (A, B, C) of the chosen option
				var answerPtr *string
				answerKey := strings.TrimSpace(item.Answer)
				if answerKey != "" && len(opts) > 0 {
					// Look up the option by key
					for _, opt := range opts {
						if strings.EqualFold(opt.Key, answerKey) {
							answerPtr = &opt.Label
							break
						}
					}
					// If answer is a label directly (old format), use it
					if answerPtr == nil {
						answerPtr = &answerKey
					}
				} else if answerKey != "" {
					answerPtr = &answerKey
				}

				d := decision.Decision{
					ID:        decision.NextID(*e.allDecisions, cat),
					Category:  cat,
					Question:  q,
					Options:   opts,
					Answer:    answerPtr,
					Implicit:  true,
					Source:    "agent",
					Reasoning: item.Reasoning,
					Features:  item.Features,
					Impact:    item.Impact,
					Date:      today,
				}
				result = append(result, d)
			}
		}
	}

	return result
}

// parseAlternatives builds options from the chosen answer and alternatives string.
func parseAlternatives(chosen string, alts string) []decision.DecisionOption {
	opts := []decision.DecisionOption{{Key: "A", Label: chosen}}
	for i, alt := range strings.Split(alts, ",") {
		label := strings.TrimSpace(alt)
		if label != "" && label != chosen {
			opts = append(opts, decision.DecisionOption{
				Key:   string(rune('B' + i)),
				Label: label,
			})
		}
	}
	return opts
}

// parsePendingOptions parses "A) option1, B) option2, C) option3" into options.
func parsePendingOptions(optStr string) []decision.DecisionOption {
	var opts []decision.DecisionOption
	re := regexp.MustCompile(`([A-Z])\)\s*([^,]+)`)
	for _, m := range re.FindAllStringSubmatch(optStr, -1) {
		opts = append(opts, decision.DecisionOption{
			Key:   m[1],
			Label: strings.TrimSpace(m[2]),
		})
	}
	return opts
}
