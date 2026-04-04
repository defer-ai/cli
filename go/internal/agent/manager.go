package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/decision"
)

// Manager coordinates decomposition and domain execution.
type Manager struct {
	provider    api.Provider
	cwd         string
	agent       *Agent
	executors   []*Executor
	allDecs     []decision.Decision
	store       *decision.DecisionStore
	execCancel  context.CancelFunc // cancels the current executor run
}

// NewManager creates a manager.
func NewManager(provider api.Provider, cwd string) *Manager {
	return &Manager{
		provider: provider,
		cwd:      cwd,
	}
}

// StartDecomposition begins task decomposition.
func (m *Manager) StartDecomposition(ctx context.Context, task string, onEvent func(Event)) {
	m.agent = NewAgent(task, m.provider, m.cwd)
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
// Cancels any previously running executor before starting.
func (m *Manager) LaunchExecutors(ctx context.Context, task string, decisions []decision.Decision, priorities map[string]CareLevel, onEvent func(Event)) {
	// Cancel previous executor run if still active
	if m.execCancel != nil {
		m.execCancel()
	}
	execCtx, cancel := context.WithCancel(ctx)
	m.execCancel = cancel

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

	// Sort by care level (auto first, review last)
	levelOrder := map[CareLevel]int{
		CareLevelAuto: 0, CareLevelReview: 1,
	}
	sort.Slice(catOrder, func(i, j int) bool {
		li := levelOrder[priorities[catOrder[i]]]
		lj := levelOrder[priorities[catOrder[j]]]
		return li < lj
	})

	// Generate codebase manifest to avoid redundant exploration
	manifest := generateCodebaseManifest(m.cwd)

	// Single executor implements everything with full context
	unified := NewExecutor(ExecOpts{
		Provider:         m.provider,
		CWD:              m.cwd,
		Task:             task,
		Domain:           "Implementation",
		CareLevel:        CareLevelAuto,
		Priorities:       priorities,
		Decisions:        decisions,
		AllDecisions:     &m.allDecs,
		OnEvent:          onEvent,
		CodebaseManifest: manifest,
	})
	m.executors = []*Executor{unified}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Don't crash
			}
		}()

		unified.Execute(execCtx)
		// Only report done if not cancelled (i.e., not replaced by a new launch)
		if execCtx.Err() == nil {
			m.persistDecisions(task)
			onEvent(Event{Type: AllExecutorsDone})
		}
	}()
}

// AutoDecide auto-answers decisions for auto-level domains.
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
		catKey := strings.ToLower(strings.TrimSpace(d.Category))
		level := priMap[catKey]

		switch level {
		case CareLevelAuto:
			// Auto-decide all decisions
			autoIDs = append(autoIDs, d.ID)
		case CareLevelReview:
			// Leave all decisions pending for user review
		default:
			// Default to auto behavior
			autoIDs = append(autoIDs, d.ID)
		}
	}
	m.agent.AutoDecide(autoIDs)
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

// generateCodebaseManifest scans the project directory and returns a compact
// file listing so executors don't waste tokens re-exploring the codebase.
func generateCodebaseManifest(cwd string) string {
	var lines []string
	maxFiles := 100 // cap to prevent huge manifests

	err := filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		// Skip hidden dirs, .defer, node_modules, .git, vendor
		name := info.Name()
		if info.IsDir() {
			if name == ".git" || name == ".defer" || name == "node_modules" || name == "vendor" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(lines) >= maxFiles {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(cwd, path)
		if rel == "" {
			return nil
		}
		size := info.Size()
		lines = append(lines, fmt.Sprintf("  %s (%d bytes)", rel, size))
		return nil
	})
	if err != nil {
		return ""
	}
	if len(lines) == 0 {
		return "(empty project)"
	}
	return strings.Join(lines, "\n")
}

func (m *Manager) persistDecisions(task string) {
	store, err := decision.LoadStore(m.cwd)
	if err != nil {
		return // disk error, skip persistence
	}
	if store == nil {
		store, err = decision.CreateStore(m.cwd, task)
		if err != nil {
			return
		}
	}
	store.Decisions = m.AllDecisions()
	_ = decision.SaveStore(m.cwd, store) // best-effort write
}

// GroupByCategory groups decisions by their category.
func GroupByCategory(decisions []decision.Decision) map[string][]decision.Decision {
	groups := make(map[string][]decision.Decision)
	for _, d := range decisions {
		groups[d.Category] = append(groups[d.Category], d)
	}
	return groups
}
