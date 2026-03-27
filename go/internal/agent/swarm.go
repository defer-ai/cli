package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// Swarm runs parallel Haiku agents to decompose domains into sub-decisions.
type Swarm struct {
	ccProvider *api.ClaudeCodeProvider
	client     *api.Client
}

// NewSwarm creates a swarm. Pass client OR ccProvider.
func NewSwarm(client *api.Client, ccProvider *api.ClaudeCodeProvider) *Swarm {
	return &Swarm{
		client:     client,
		ccProvider: ccProvider,
	}
}

const swarmSystemPromptTemplate = `You decompose a single software domain into specific implementation decisions.

CRITICAL: Every decision MUST use category "%s". Do NOT create other categories.

Output ONLY a JSON array. No explanation, no markdown, no preamble.
Each decision: {"category": "%s", "decision": "...", "options": ["A", "B", "C"], "reasoning": "..."}
Output 8-12 decisions. Be specific and technical.`

// ExpandDomains runs parallel Haiku agents for each decision category.
// onDecisions is called with sub-decisions as they arrive from each domain.
func (s *Swarm) ExpandDomains(ctx context.Context, task string, decisions []decision.Decision, onDecisions func([]decision.Decision)) {
	groups := GroupByCategory(decisions)

	var wg sync.WaitGroup
	for cat, decs := range groups {
		wg.Add(1)
		go func(category string, catDecs []decision.Decision) {
			defer wg.Done()

			var lines []string
			for _, d := range catDecs {
				lines = append(lines, d.Question)
			}

			sysPrompt := fmt.Sprintf(swarmSystemPromptTemplate, category, category)
			userPrompt := "Domain: " + category + "\nExisting decisions: " + strings.Join(lines, "; ") + "\nTask: " + task + "\nDecompose into 8-12 specific sub-decisions for the " + category + " domain ONLY."

			text := s.runHaikuCompletion(ctx, sysPrompt, userPrompt)
			if text == "" {
				return
			}

			// Force all decisions into this category regardless of what the model returned
			subDecs := ParseSwarmDecisions(text, category, decisions)
			if len(subDecs) > 0 {
				onDecisions(subDecs)
			}
		}(cat, decs)
	}
	wg.Wait()
}

// runHaikuCompletion runs a single Haiku completion via Claude Code subprocess or direct API.
func (s *Swarm) runHaikuCompletion(ctx context.Context, sysPrompt, userPrompt string) string {
	if s.client == nil && s.ccProvider != nil {
		cp := api.NewClaudeCodeProvider("haiku")
		events := make(chan api.Event, 100)
		go cp.RunCompletion(ctx, sysPrompt, userPrompt, events)

		var text string
		for ev := range events {
			switch ev.Type {
			case api.EventTextDelta:
				text += ev.Text
			case api.EventDone, api.EventError:
				return text
			}
		}
		return text
	}

	if s.client != nil {
		resp, err := api.SimpleCompletion(ctx, s.client, sysPrompt, userPrompt)
		if err != nil {
			return ""
		}
		return resp
	}

	return ""
}

// GroupByCategory groups decisions by their category.
func GroupByCategory(decisions []decision.Decision) map[string][]decision.Decision {
	groups := make(map[string][]decision.Decision)
	for _, d := range decisions {
		groups[d.Category] = append(groups[d.Category], d)
	}
	return groups
}

// ParseSwarmDecisions extracts decision objects from swarm model output.
func ParseSwarmDecisions(text, defaultCategory string, existing []decision.Decision) []decision.Decision {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return nil
	}

	var raw []struct {
		Category  string   `json:"category"`
		Decision  string   `json:"decision"`
		Options   []string `json:"options"`
		Reasoning string   `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err != nil {
		return nil
	}

	today := time.Now().Format("2006-01-02")
	all := make([]decision.Decision, len(existing))
	copy(all, existing)

	const maxPerDomain = 12
	var result []decision.Decision
	seen := make(map[string]bool)
	for _, ex := range existing {
		seen[strings.ToLower(strings.TrimSpace(ex.Question))] = true
	}

	for _, item := range raw {
		// ALWAYS use the domain's category -- never trust model output
		cat := defaultCategory

		// Cap decisions per domain
		if len(result) >= maxPerDomain {
			break
		}
		q := item.Decision
		if q == "" {
			continue
		}

		// Dedup
		key := strings.ToLower(strings.TrimSpace(q))
		if seen[key] {
			continue
		}
		seen[key] = true

		// Build options
		var opts []decision.DecisionOption
		for i, o := range item.Options {
			letter := string(rune('A' + i))
			opts = append(opts, decision.DecisionOption{Key: letter, Label: o})
		}

		d := decision.Decision{
			ID:        decision.NextID(all, cat),
			Category:  cat,
			Question:  q,
			Options:   opts,
			Reasoning: item.Reasoning,
			Implicit:  true,
			Source:    "agent",
			Date:      today,
		}
		all = append(all, d)
		result = append(result, d)
	}

	return result
}
