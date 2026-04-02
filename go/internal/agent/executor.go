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

// inlineDecisionRe matches patterns like "DECISION: STACK-001 = Go with Gin" in executor output.
var inlineDecisionRe = regexp.MustCompile(`DECISION:\s*([A-Z]+-\d+)\s*=\s*(.+)`)


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
	provider        api.Provider
	cwd             string
	task            string
	domain          string
	careLevel       CareLevel
	priorities      map[string]CareLevel // per-category care levels
	decisions       []decision.Decision
	allDecisions    *[]decision.Decision
	knownCategories []string
	state           ExecState
	onEvent         func(Event)
}

// ExecOpts configures a new executor.
type ExecOpts struct {
	Provider     api.Provider
	CWD          string
	Task         string
	Domain       string
	CareLevel    CareLevel
	Priorities   map[string]CareLevel
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
func (e *Executor) freshProvider() api.Provider {
	if cc, ok := e.provider.(*api.ClaudeCodeProvider); ok {
		return api.NewClaudeCodeProvider(cc.GetModel())
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
	systemPrompt := fmt.Sprintf(ExecutePromptTemplate, e.domain, CarePrompts[e.careLevel])

	userMsg := fmt.Sprintf("Task: %s\n\nDomain: %s\nDecisions:\n%s\n\nImplement the %s domain now.",
		e.task, e.domain, decSummary, e.domain)

	events := make(chan api.Event, 100)

	// Use a fresh provider per domain (isolated sessions)
	cp := e.freshProvider()
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

			// Scan for inline decision updates (e.g. "DECISION: STACK-001 = Go with Gin")
			e.scanInlineDecisions(ev.Text)

		case api.EventToolCallStart:
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
	msg := fmt.Sprintf("Task: %s\n\nALREADY DECIDED (do NOT repeat these):\n%s\n\nKNOWN CATEGORIES (you MUST use only these): %s\n\nWhat NEW implementation decisions still need to be made? Do NOT rephrase or reference existing decisions. Only list decisions that are NOT in the 'already decided' list above.",
		e.task, decSummary, catList)

	planPrompt := PlanPrompt + fmt.Sprintf("\n\nCRITICAL: The category field MUST be one of: %s. Do NOT invent new categories. Do NOT repeat or rephrase any decision from the ALREADY DECIDED list.", catList)
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
	cp := e.freshProvider()
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

	// Dedup by question (normalize: strip parenthetical references, lowercase, trim)
	newQ := normalizeQuestion(d.Question)
	for _, existing := range *e.allDecisions {
		existQ := normalizeQuestion(existing.Question)
		if existQ == newQ {
			return
		}
		// Word-level similarity: if 70%+ of words overlap, treat as duplicate
		if questionsOverlap(newQ, existQ) {
			return
		}
	}

	// Regenerate ID
	d.ID = decision.NextID(*e.allDecisions, d.Category)

	// Apply care level: high/paranoid decisions stay pending for user review
	level := e.getCareLevel(d.Category)
	if level == CareLevelHigh || level == CareLevelParanoid {
		d.Answer = nil
		d.Delegated = false
		d.Source = "agent"
	}

	// Skip: don't even log non-major decisions
	if level == CareLevelSkip && d.Implicit {
		return
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
	return e.careLevel // fall back to executor's default
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
	// Remove parenthetical references like "(LAYOUT-037 — explicitly pending)"
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

// questionsOverlap returns true if two normalized questions share 70%+ of their significant words.
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
	return float64(overlap)/float64(smaller) >= 0.7
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
			(*e.allDecisions)[i].Answer = &answer
			(*e.allDecisions)[i].Source = "agent"
			return true
		}
	}
	return false
}

// scanInlineDecisions scans text for patterns like "DECISION: STACK-001 = Go with Gin"
// and calls UpdateDecision for each match.
func (e *Executor) scanInlineDecisions(text string) {
	matches := inlineDecisionRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) == 3 {
			id := strings.TrimSpace(m[1])
			answer := strings.TrimSpace(m[2])
			if id != "" && answer != "" {
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
			Answer    string `json:"answer"`
			Reasoning string `json:"reasoning"`
			Impact    int    `json:"impact"`
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
					Impact:    item.Impact,
					Date:      today,
				}
				result = append(result, d)
			}
		}
	}

	return result
}
