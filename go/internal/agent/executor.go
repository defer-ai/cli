package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// DomainStatus tracks executor progress.
type DomainStatus int

const (
	DomainPending   DomainStatus = iota
	DomainPlanning
	DomainExecuting
	DomainVerifying
	DomainDone
	DomainError
)

func (s DomainStatus) String() string {
	switch s {
	case DomainPlanning:  return "planning"
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
	ccProvider      *api.ClaudeCodeProvider
	cwd             string
	task            string
	domain          string
	careLevel       CareLevel
	decisions       []decision.Decision
	allDecisions    *[]decision.Decision
	knownCategories []string // canonical categories from decomposition
	state           ExecState
	onEvent         func(Event)
}

// ExecOpts configures a new executor.
type ExecOpts struct {
	CCProvider   *api.ClaudeCodeProvider
	CWD          string
	Task         string
	Domain       string
	CareLevel    CareLevel
	Decisions    []decision.Decision
	AllDecisions *[]decision.Decision
	OnEvent      func(Event)
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

	return &Executor{
		ccProvider:      opts.CCProvider,
		cwd:             opts.CWD,
		task:            opts.Task,
		domain:          opts.Domain,
		careLevel:       opts.CareLevel,
		decisions:       opts.Decisions,
		allDecisions:    opts.AllDecisions,
		knownCategories: cats,
		onEvent:         opts.OnEvent,
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

// Execute runs the full domain lifecycle.
func (e *Executor) Execute(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			e.setStatus(DomainError, "", fmt.Sprintf("panic: %v", r))
		}
	}()

	decSummary := e.decisionSummary()

	// Phase 1: Planning -- always runs, care level doesn't affect decision count
	e.setStatus(DomainPlanning, "Planning...", "")
	e.plan(ctx, decSummary)

	// Phase 2: Execution
	e.setStatus(DomainExecuting, "", "")
	fullOutput := e.execute(ctx, decSummary)
	if e.state.Status == DomainError {
		return
	}

	// Phase 3: Verification (always runs)
	e.setStatus(DomainVerifying, "Verifying...", "")
	issues := e.verify(ctx, fullOutput, decSummary)
	if issues != "" {
		e.setStatus(DomainExecuting, "Fixing issues...", "")
		fullOutput = e.fix(ctx, issues, decSummary)
	}

	// Phase 4: Extract implicit decisions (always runs)
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

func (e *Executor) decisionSummary() string {
	var lines []string
	for _, d := range e.decisions {
		answer := "(pending)"
		if d.Answer != nil {
			answer = *d.Answer
			if d.Delegated {
				answer = "DELEGATED: " + answer
			}
		}
		lines = append(lines, fmt.Sprintf("%s: %s -> %s", d.ID, d.Question, answer))
	}
	return strings.Join(lines, "\n")
}

// simpleCompletion runs a one-shot completion via subprocess.
func (e *Executor) simpleCompletion(ctx context.Context, systemPrompt, userMsg string) (string, error) {
	cp := api.NewClaudeCodeProvider(e.ccProvider.GetModel())
	events := make(chan api.Event, 100)
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)
	var text string
	for ev := range events {
		if ev.Type == api.EventTextDelta {
			text += ev.Text
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
	systemPrompt := fmt.Sprintf(ExecutePromptTemplate, e.domain, CarePrompts[e.careLevel])

	userMsg := fmt.Sprintf("Task: %s\n\nDomain: %s\nDecisions:\n%s\n\nImplement the %s domain now.",
		e.task, e.domain, decSummary, e.domain)

	events := make(chan api.Event, 100)

	// Use a fresh subprocess provider per domain (isolated sessions)
	cp := api.NewClaudeCodeProvider(e.ccProvider.GetModel())
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
			e.mu.Lock()
			if len(fullText) > 100000 {
				e.state.Output = "...(truncated)\n" + fullText[len(fullText)-100000:]
			} else {
				e.state.Output = fullText
			}
			e.mu.Unlock()
			e.onEvent(Event{Type: ExecStateChanged, ExecutorID: e.state.ID})

		case api.EventDecisionFound:
			if ev.Decision != nil {
				e.storeDecision(*ev.Decision)
			}

		case api.EventError:
			e.setStatus(DomainError, fullText, ev.Error.Error())
			return fullText

		case api.EventDone:
			return fullText
		}
	}
	return fullText
}

func (e *Executor) plan(ctx context.Context, decSummary string) {
	catList := strings.Join(e.knownCategories, ", ")
	msg := fmt.Sprintf("Task: %s\nExisting decisions:\n%s\n\nKNOWN CATEGORIES (you MUST use only these): %s\n\nWhat implementation decisions will you need to make? Use ONLY the categories listed above.",
		e.task, decSummary, catList)

	planPrompt := PlanPrompt + fmt.Sprintf("\n\nCRITICAL: The category field MUST be one of: %s. Do NOT invent new categories.", catList)
	resp, err := e.simpleCompletion(ctx, planPrompt, msg)
	if err != nil {
		return // best effort
	}

	decs := e.parseImplicitChoices(resp)
	for i := range decs {
		decs[i].Reasoning = "[planned] " + decs[i].Reasoning
	}
	e.storeDecisions(decs)
}

func (e *Executor) verify(ctx context.Context, output, decSummary string) string {
	truncated := output
	if len(truncated) > 6000 {
		truncated = "..." + truncated[len(truncated)-6000:]
	}
	msg := fmt.Sprintf("Domain: %s\nTask: %s\nDecisions:\n%s\n\nImplementation:\n%s",
		e.domain, e.task, decSummary, truncated)

	resp, err := e.simpleCompletion(ctx, VerifyPrompt, msg)
	if err != nil {
		return ""
	}

	if strings.Contains(resp, "NEEDS FIX") {
		return resp
	}
	return ""
}

func (e *Executor) fix(ctx context.Context, issues, decSummary string) string {
	systemPrompt := fmt.Sprintf(ExecutePromptTemplate, e.domain, CarePrompts[e.careLevel])
	userMsg := fmt.Sprintf("Verification found issues:\n%s\n\nFix these now.", issues)

	events := make(chan api.Event, 100)
	cp := api.NewClaudeCodeProvider(e.ccProvider.GetModel())
	go cp.RunCompletion(ctx, systemPrompt, userMsg, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
		case api.EventDecisionFound:
			if ev.Decision != nil {
				e.storeDecision(*ev.Decision)
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

	resp, err := e.simpleCompletion(ctx, ExtractPrompt, msg)
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

	// Normalize category
	d.Category = e.normalizeCategoryLocked(d.Category)

	// Dedup by question
	for _, existing := range *e.allDecisions {
		if strings.EqualFold(strings.TrimSpace(existing.Question), strings.TrimSpace(d.Question)) {
			return
		}
	}

	// Regenerate ID using current allDecisions (avoids stale snapshot duplicates)
	d.ID = decision.NextID(*e.allDecisions, d.Category)

	*e.allDecisions = append(*e.allDecisions, d)
	e.onEvent(Event{Type: ExecDecisionStored, ExecutorID: e.state.ID, Decisions: []decision.Decision{d}})
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

func (e *Executor) storeDecisions(decs []decision.Decision) {
	for _, d := range decs {
		e.storeDecision(d)
	}
}

func (e *Executor) normalizeCategory(cat string) string {
	lower := strings.ToLower(strings.TrimSpace(cat))
	for _, d := range *e.allDecisions {
		if strings.EqualFold(strings.TrimSpace(d.Category), lower) {
			return d.Category
		}
	}
	if strings.EqualFold(strings.TrimSpace(e.domain), lower) {
		return e.domain
	}
	return "Misc"
}

func (e *Executor) parseImplicitChoices(text string) []decision.Decision {
	var result []decision.Decision
	today := time.Now().Format("2006-01-02")

	// Try JSON array
	// Find JSON array in text (might be wrapped in markdown)
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		var raw []struct {
			Category  string `json:"category"`
			Decision  string `json:"decision"`  // old format
			Question  string `json:"question"`   // new format
			Answer    string `json:"answer"`     // new format
			Reasoning string `json:"reasoning"`
		}
		if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err == nil {
			for _, item := range raw {
				cat := e.normalizeCategoryLocked(item.Category)

				// Support both old ("decision") and new ("question"/"answer") formats
				q := strings.TrimSpace(item.Question)
				if q == "" {
					q = strings.TrimSpace(item.Decision)
				}
				if q == "" {
					continue
				}

				a := strings.TrimSpace(item.Answer)
				if a == "" {
					a = strings.TrimSpace(item.Reasoning)
				}
				if a == "" || strings.EqualFold(a, q) {
					a = "(agent decided)"
				}

				d := decision.Decision{
					ID:        decision.NextID(*e.allDecisions, cat),
					Category:  cat,
					Question:  q,
					Answer:    &a,
					Implicit:  true,
					Source:    "agent",
					Reasoning: item.Reasoning,
					Date:      today,
				}
				result = append(result, d)
			}
		}
	}

	return result
}
