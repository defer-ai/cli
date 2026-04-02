package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// Status of the decomposition agent.
type Status int

const (
	StatusIdle Status = iota
	StatusThinking
	StatusDone
	StatusError
)

// State is the immutable snapshot of the decomposition agent.
type State struct {
	Task      string
	Status    Status
	Decisions []decision.Decision
	Output    string
	Error     string
}

// Agent handles task decomposition.
type Agent struct {
	mu       sync.Mutex
	provider api.Provider
	cwd      string
	state    State
}

// NewAgent creates a decomposition agent.
func NewAgent(task string, provider api.Provider, cwd string) *Agent {
	return &Agent{
		provider: provider,
		cwd:      cwd,
		state: State{
			Task:   task,
			Status: StatusIdle,
		},
	}
}

// State returns an immutable snapshot.
func (a *Agent) State() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.state
	s.Decisions = make([]decision.Decision, len(a.state.Decisions))
	copy(s.Decisions, a.state.Decisions)
	return s
}

// Decompose runs the decomposition in a goroutine. Results sent via callback.
func (a *Agent) Decompose(ctx context.Context, onEvent func(Event)) {
	a.mu.Lock()
	a.state.Status = StatusThinking
	a.mu.Unlock()
	onEvent(Event{Type: AgentStateChanged})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.mu.Lock()
				a.state.Status = StatusError
				a.state.Error = fmt.Sprintf("panic: %v", r)
				a.mu.Unlock()
				onEvent(Event{Type: AgentStateChanged})
			}
		}()

		a.runDecompositionSubprocess(ctx, onEvent, 0)
	}()
}

// runDecompositionSubprocess uses Claude Code subprocess for decomposition.
func (a *Agent) runDecompositionSubprocess(ctx context.Context, onEvent func(Event), retries int) {
	events := make(chan api.Event, 100)

	go a.provider.RunCompletion(ctx, DecomposePrompt, a.state.Task, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
			a.mu.Lock()
			a.state.Output = fullText
			a.mu.Unlock()
			onEvent(Event{Type: AgentStateChanged})

		case api.EventPermissionRequest:
			if ev.PermissionReq != nil {
				onEvent(Event{Type: ExecPermissionRequest, PermissionReq: ev.PermissionReq})
			}

		case api.EventError:
			a.mu.Lock()
			a.state.Status = StatusError
			a.state.Error = ev.Error.Error()
			a.mu.Unlock()
			onEvent(Event{Type: AgentStateChanged})
			return

		case api.EventDone:
			decisions := parseDecisions(fullText, a.state.Decisions)

			if len(decisions) == 0 && retries < 2 {
				a.provider.ResetSession()
				prompt := "You did not output a ```defer-decisions JSON block. This is required. Output the decisions now.\n\nOriginal task: " + a.state.Task
				a.runDecompositionSubprocessRetry(ctx, onEvent, retries+1, prompt)
				return
			}

			// Add Misc catch-all
			hasMisc := false
			for _, d := range decisions {
				if strings.EqualFold(d.Category, "misc") {
					hasMisc = true
					break
				}
			}
			if !hasMisc && len(decisions) > 0 {
				miscAnswer := "(catch-all category)"
				decisions = append(decisions, decision.Decision{
					ID:        decision.NextID(decisions, "Misc"),
					Category:  "Misc",
					Question:  "Uncategorized implementation decisions",
					Answer:    &miscAnswer,
					Delegated: true,
					Implicit:  true,
					Source:    "auto",
					Date:      time.Now().Format("2006-01-02"),
				})
			}

			a.mu.Lock()
			a.state.Decisions = decisions
			a.state.Status = StatusDone
			a.mu.Unlock()
			onEvent(Event{Type: AgentDecisionsReady, Decisions: decisions})
			return
		}
	}
}

func (a *Agent) runDecompositionSubprocessRetry(ctx context.Context, onEvent func(Event), retries int, prompt string) {
	events := make(chan api.Event, 100)
	go a.provider.RunCompletion(ctx, DecomposePrompt, prompt, events)

	var fullText string
	for ev := range events {
		switch ev.Type {
		case api.EventTextDelta:
			fullText += ev.Text
		case api.EventPermissionRequest:
			if ev.PermissionReq != nil {
				onEvent(Event{Type: ExecPermissionRequest, PermissionReq: ev.PermissionReq})
			}
		case api.EventDone:
			decisions := parseDecisions(fullText, a.state.Decisions)
			if len(decisions) == 0 {
				a.mu.Lock()
				a.state.Status = StatusDone
				a.state.Output = fullText
				a.mu.Unlock()
				onEvent(Event{Type: AgentDecisionsReady})
				return
			}
			a.mu.Lock()
			a.state.Decisions = decisions
			a.state.Status = StatusDone
			a.mu.Unlock()
			onEvent(Event{Type: AgentDecisionsReady, Decisions: decisions})
			return
		case api.EventError:
			a.mu.Lock()
			a.state.Status = StatusError
			a.state.Error = ev.Error.Error()
			a.mu.Unlock()
			onEvent(Event{Type: AgentStateChanged})
			return
		}
	}
}

// AutoDecide auto-answers pending decisions by picking the first real option.
func (a *Agent) AutoDecide(ids []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	today := time.Now().Format("2006-01-02")
	for i := range a.state.Decisions {
		d := &a.state.Decisions[i]
		if d.Answer != nil {
			continue
		}
		if len(ids) > 0 && !idSet[d.ID] {
			continue
		}
		// Pick first non-"choose for me" option
		var answer string
		for _, opt := range d.Options {
			if !strings.Contains(strings.ToLower(opt.Label), "choose for me") {
				answer = opt.Label
				break
			}
		}
		if answer == "" && len(d.Options) > 0 {
			answer = d.Options[0].Label
		}
		if answer == "" {
			answer = "auto-decided"
		}
		d.Answer = &answer
		d.Delegated = true
		d.Source = "auto"
		d.Date = today
	}
}

// Decisions returns the current decision list.
func (a *Agent) Decisions() []decision.Decision {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]decision.Decision, len(a.state.Decisions))
	copy(out, a.state.Decisions)
	return out
}

// AddDecisions merges new decisions (with dedup).
func (a *Agent) AddDecisions(decs []decision.Decision) {
	a.mu.Lock()
	defer a.mu.Unlock()

	existing := make(map[string]bool)
	for _, d := range a.state.Decisions {
		existing[strings.ToLower(strings.TrimSpace(d.Question))] = true
	}
	for _, d := range decs {
		key := strings.ToLower(strings.TrimSpace(d.Question))
		if existing[key] {
			continue
		}
		existing[key] = true
		a.state.Decisions = append(a.state.Decisions, d)
	}
}

// --- Parsing ---

// ParseScanDecisions extracts decisions from a scan response.
func ParseScanDecisions(text string) []decision.Decision {
	return parseDecisions(text, nil)
}

var decisionBlockRe = regexp.MustCompile("```defer-decisions\\s*\n([\\s\\S]*?)\n```")

func parseDecisions(text string, existing []decision.Decision) []decision.Decision {
	match := decisionBlockRe.FindStringSubmatch(text)
	if match == nil {
		return nil
	}

	var raw []struct {
		Category  string `json:"category"`
		Question  string `json:"question"`
		Options   []struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		} `json:"options"`
		Context   string   `json:"context"`
		Impact    int      `json:"impact"`
		DependsOn []string `json:"dependsOn"`
	}

	if err := json.Unmarshal([]byte(match[1]), &raw); err != nil {
		return nil
	}

	today := time.Now().Format("2006-01-02")
	var result []decision.Decision
	all := append(existing, result...)

	for _, item := range raw {
		cat := item.Category
		if cat == "" {
			cat = "General"
		}
		opts := make([]decision.DecisionOption, len(item.Options))
		for i, o := range item.Options {
			opts[i] = decision.DecisionOption{Key: o.Key, Label: o.Label}
		}
		d := decision.Decision{
			ID:        decision.NextID(all, cat),
			Category:  cat,
			Question:  item.Question,
			Options:   opts,
			Context:   item.Context,
			Impact:    item.Impact,
			DependsOn: item.DependsOn,
			Source:    "user",
			Date:      today,
		}
		all = append(all, d)
		result = append(result, d)
	}

	return result
}
