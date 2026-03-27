package agent

import (
	"context"
	"sort"
	"strings"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// Manager coordinates decomposition and domain execution.
type Manager struct {
	client     *api.Client              // direct API (may be nil)
	ccProvider *api.ClaudeCodeProvider   // subprocess fallback (may be nil)
	cwd        string
	agent      *Agent
	executors  []*Executor
	allDecs    []decision.Decision
	store      *decision.DecisionStore
}

// NewManager creates a manager. Pass client OR ccProvider.
func NewManager(client *api.Client, ccProvider *api.ClaudeCodeProvider, cwd string) *Manager {
	return &Manager{
		client:     client,
		ccProvider: ccProvider,
		cwd:        cwd,
	}
}

// StartDecomposition begins task decomposition.
func (m *Manager) StartDecomposition(ctx context.Context, task string, onEvent func(Event)) {
	m.agent = NewAgent(task, m.client, m.ccProvider, m.cwd)
	m.agent.Decompose(ctx, onEvent)
}

// Agent returns the decomposition agent.
func (m *Manager) Agent() *Agent {
	return m.agent
}

// Executors returns all domain executors.
func (m *Manager) Executors() []*Executor {
	return m.executors
}

// LaunchExecutors starts per-domain execution. Runs sequentially in a goroutine.
func (m *Manager) LaunchExecutors(ctx context.Context, task string, decisions []decision.Decision, priorities map[string]CareLevel, onEvent func(Event)) {
	m.allDecs = make([]decision.Decision, len(decisions))
	copy(m.allDecs, decisions)

	// Group by category
	groups := make(map[string][]decision.Decision)
	var catOrder []string
	for _, d := range decisions {
		cat := d.Category
		if _, ok := groups[cat]; !ok {
			catOrder = append(catOrder, cat)
		}
		groups[cat] = append(groups[cat], d)
	}

	// Sort by care level (skip first, paranoid last)
	levelOrder := map[CareLevel]int{
		CareLevelSkip: 0, CareLevelLow: 1, CareLevelMedium: 2,
		CareLevelHigh: 3, CareLevelParanoid: 4,
	}
	sort.Slice(catOrder, func(i, j int) bool {
		li := levelOrder[priorities[catOrder[i]]]
		lj := levelOrder[priorities[catOrder[j]]]
		return li < lj
	})

	// Single executor implements everything with full context
	unified := NewExecutor(ExecOpts{
		Client:       m.client,
		CCProvider:   m.ccProvider,
		CWD:          m.cwd,
		Task:         task,
		Domain:       "Implementation",
		CareLevel:    CareLevelMedium,
		Decisions:    decisions,
		AllDecisions: &m.allDecs,
		OnEvent:      onEvent,
	})
	m.executors = []*Executor{unified}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Don't crash
			}
		}()

		unified.Execute(ctx)
		m.persistDecisions(task)
		onEvent(Event{Type: AllExecutorsDone})
	}()
}

// AutoDecide auto-answers decisions for non-paranoid/high domains.
func (m *Manager) AutoDecide(priorities map[string]CareLevel) {
	if m.agent == nil {
		return
	}
	// Build case-insensitive priority lookup
	priMap := make(map[string]CareLevel)
	for k, v := range priorities {
		priMap[strings.ToLower(strings.TrimSpace(k))] = v
	}

	var autoIDs []string
	for _, d := range m.agent.Decisions() {
		if d.Answer != nil {
			continue
		}
		level := priMap[strings.ToLower(strings.TrimSpace(d.Category))]
		if level != CareLevelParanoid && level != CareLevelHigh {
			autoIDs = append(autoIDs, d.ID)
		}
	}
	m.agent.AutoDecide(autoIDs)
}

// RunSwarm runs the Haiku subagent swarm to expand domains into sub-decisions.
func (m *Manager) RunSwarm(ctx context.Context, task string, decisions []decision.Decision, onEvent func(Event)) {
	swarm := NewSwarm(m.client, m.ccProvider)
	swarm.ExpandDomains(ctx, task, decisions, func(subDecs []decision.Decision) {
		// Add sub-decisions to allDecs with dedup
		for _, d := range subDecs {
			dup := false
			for _, existing := range m.allDecs {
				if strings.EqualFold(strings.TrimSpace(existing.Question), strings.TrimSpace(d.Question)) {
					dup = true
					break
				}
			}
			if !dup {
				m.allDecs = append(m.allDecs, d)
			}
		}
		onEvent(Event{Type: ExecDecisionStored, Decisions: subDecs})
	})
}

// AllDecisions returns the current shared decision list.
func (m *Manager) AllDecisions() []decision.Decision {
	return m.allDecs
}

// SyncDecisions updates allDecs from the authoritative tree decisions (preserves user changes).
func (m *Manager) SyncDecisions(treeDecs []decision.Decision) {
	// Build map of tree decisions by ID (user-edited are authoritative)
	byID := make(map[string]*decision.Decision)
	for i := range treeDecs {
		byID[treeDecs[i].ID] = &treeDecs[i]
	}

	// Update allDecs with any user changes
	for i := range m.allDecs {
		if td, ok := byID[m.allDecs[i].ID]; ok {
			m.allDecs[i].Answer = td.Answer
			m.allDecs[i].Source = td.Source
			m.allDecs[i].Delegated = td.Delegated
		}
	}

	// Add any tree decisions not yet in allDecs
	existing := make(map[string]bool)
	for _, d := range m.allDecs {
		existing[d.ID] = true
	}
	for _, d := range treeDecs {
		if !existing[d.ID] {
			m.allDecs = append(m.allDecs, d)
		}
	}
}

func (m *Manager) persistDecisions(task string) {
	store, _ := decision.LoadStore(m.cwd)
	if store == nil {
		store, _ = decision.CreateStore(m.cwd, task)
	}
	if store == nil {
		return
	}
	store.Decisions = m.AllDecisions()
	decision.SaveStore(m.cwd, store)
}
