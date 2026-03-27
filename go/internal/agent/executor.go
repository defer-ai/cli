package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
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
	mu           sync.Mutex
	client       *api.Client
	ccProvider   *api.ClaudeCodeProvider
	cwd          string
	task         string
	domain       string
	careLevel    CareLevel
	decisions    []decision.Decision
	allDecisions *[]decision.Decision
	state        ExecState
	onEvent      func(Event)
}

// ExecOpts configures a new executor.
type ExecOpts struct {
	Client       *api.Client
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
	return &Executor{
		client:       opts.Client,
		ccProvider:   opts.CCProvider,
		cwd:          opts.CWD,
		task:         opts.Task,
		domain:       opts.Domain,
		careLevel:    opts.CareLevel,
		decisions:    opts.Decisions,
		allDecisions: opts.AllDecisions,
		onEvent:      opts.OnEvent,
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

func (e *Executor) useSubprocess() bool {
	return e.client == nil && e.ccProvider != nil
}

// simpleCompletion runs a one-shot completion via API or subprocess.
func (e *Executor) simpleCompletion(ctx context.Context, systemPrompt, userMsg string) (string, error) {
	if e.useSubprocess() {
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
	return api.SimpleCompletion(ctx, e.client, systemPrompt, userMsg)
}

func (e *Executor) execute(ctx context.Context, decSummary string) string {
	systemPrompt := fmt.Sprintf(ExecutePromptTemplate, e.domain, CarePrompts[e.careLevel])

	userMsg := fmt.Sprintf("Task: %s\n\nDomain: %s\nDecisions:\n%s\n\nImplement the %s domain now.",
		e.task, e.domain, decSummary, e.domain)

	events := make(chan api.Event, 100)

	if e.useSubprocess() {
		// Use a fresh subprocess provider per domain (isolated sessions)
		cp := api.NewClaudeCodeProvider(e.ccProvider.GetModel())
		go cp.RunCompletion(ctx, systemPrompt, userMsg, events)
	} else {
		go api.RunAgentLoop(ctx, api.RunConfig{
			Client:       e.client,
			SystemPrompt: systemPrompt,
			Messages: []anthropic.MessageParam{
				{
					Role: anthropic.MessageParamRoleUser,
					Content: []anthropic.ContentBlockParamUnion{
						{OfText: &anthropic.TextBlockParam{Text: userMsg}},
					},
				},
			},
			ToolSet:      api.AllTools,
			CWD:          e.cwd,
			MaxTurns:     50,
			Domain:       e.domain,
			AllDecisions: *e.allDecisions,
		}, events)
	}

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
	msg := fmt.Sprintf("Task: %s\nDomain: %s\nExisting decisions:\n%s\n\nWhat implementation decisions will you need to make?",
		e.task, e.domain, decSummary)

	resp, err := e.simpleCompletion(ctx, PlanPrompt, msg)
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
	if e.useSubprocess() {
		cp := api.NewClaudeCodeProvider(e.ccProvider.GetModel())
		go cp.RunCompletion(ctx, systemPrompt, userMsg, events)
	} else {
		go api.RunAgentLoop(ctx, api.RunConfig{
			Client:       e.client,
			SystemPrompt: systemPrompt,
			Messages: []anthropic.MessageParam{
				{
					Role: anthropic.MessageParamRoleUser,
					Content: []anthropic.ContentBlockParamUnion{
						{OfText: &anthropic.TextBlockParam{Text: userMsg}},
					},
				},
			},
			ToolSet:      api.AllTools,
			CWD:          e.cwd,
			MaxTurns:     20,
			Domain:       e.domain,
			AllDecisions: *e.allDecisions,
		}, events)
	}

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
	for _, d := range *e.allDecisions {
		if strings.EqualFold(strings.TrimSpace(d.Category), lower) {
			return d.Category
		}
	}
	return cat
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
			Decision  string `json:"decision"`
			Reasoning string `json:"reasoning"`
		}
		if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err == nil {
			for _, item := range raw {
				cat := e.normalizeCategory(item.Category)
				answer := item.Decision
				if answer == "" {
					answer = item.Reasoning
				}
				result = append(result, decision.Decision{
					ID:        decision.NextID(*e.allDecisions, cat),
					Category:  cat,
					Question:  item.Decision,
					Answer:    &answer,
					Implicit:  true,
					Source:    "agent",
					Reasoning: item.Reasoning,
					Date:      today,
				})
			}
		}
	}

	return result
}
