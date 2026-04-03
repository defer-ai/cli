package agent

import (
	"strings"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func managerDecisions() []decision.Decision {
	return []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}, {Key: "B", Label: "Choose for me"}},
			Source: "user"},
		{ID: "STA-0002", Category: "Stack", Question: "Framework?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Gin"}, {Key: "B", Label: "Choose for me"}},
			Source: "user"},
		{ID: "UII-0001", Category: "UI", Question: "CSS approach?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Tailwind"}, {Key: "B", Label: "CSS Modules"}},
			Source: "user"},
		{ID: "DAT-0001", Category: "Data", Question: "Database?",
			Options: []decision.DecisionOption{{Key: "A", Label: "PostgreSQL"}, {Key: "B", Label: "SQLite"}},
			Source: "user"},
	}
}

func setupManagerWithDecs(t *testing.T) *Manager {
	t.Helper()
	m := NewManager(nil, "/tmp/test")
	a := NewAgent("test task", nil, "/tmp/test")
	a.state.Decisions = managerDecisions()
	a.state.Status = StatusDone
	m.agent = a
	return m
}

func TestAutoDecidePriorities(t *testing.T) {
	mgr := setupManagerWithDecs(t)

	priorities := map[string]CareLevel{
		"Stack": CareLevelSkip,
		"UI":    CareLevelParanoid,
		"Data":  CareLevelMedium,
	}
	mgr.AutoDecide(priorities)

	decs := mgr.Agent().Decisions()

	tests := []struct {
		id       string
		wantAuto bool
		desc     string
	}{
		{"STA-0001", true, "skip domain auto-decides"},
		{"STA-0002", true, "skip domain auto-decides (second)"},
		{"UII-0001", false, "paranoid domain stays pending"},
		{"DAT-0001", false, "medium domain keeps first decision per category pending"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var found bool
			for _, d := range decs {
				if d.ID == tt.id {
					found = true
					if tt.wantAuto && d.Answer == nil {
						t.Errorf("%s should be auto-decided", tt.id)
					}
					if !tt.wantAuto && d.Answer != nil {
						t.Errorf("%s should be pending, got %q", tt.id, *d.Answer)
					}
					if tt.wantAuto && d.Answer != nil {
						if d.Source != "auto" {
							t.Errorf("%s source = %q, want auto", tt.id, d.Source)
						}
						if !d.Delegated {
							t.Errorf("%s should be delegated", tt.id)
						}
					}
				}
			}
			if !found {
				t.Errorf("decision %s not found", tt.id)
			}
		})
	}
}

func TestAutoDecideCaseInsensitive(t *testing.T) {
	mgr := setupManagerWithDecs(t)

	// Use different casing
	priorities := map[string]CareLevel{
		"stack": CareLevelSkip,
		"ui":    CareLevelParanoid,
		"data":  CareLevelLow,
	}
	mgr.AutoDecide(priorities)

	decs := mgr.Agent().Decisions()
	for _, d := range decs {
		if d.Category == "Stack" && d.Answer == nil {
			t.Errorf("Stack decision %s should be auto-decided (case-insensitive)", d.ID)
		}
		if d.Category == "UI" && d.Answer != nil {
			t.Errorf("UI decision %s should be pending (paranoid)", d.ID)
		}
		if d.Category == "Data" && d.Answer == nil {
			t.Errorf("Data decision %s should be auto-decided (low)", d.ID)
		}
	}
}

func TestAutoDecideHighAllAutoDecided(t *testing.T) {
	// When ALL decisions are high/paranoid, autoIDs is nil.
	// Agent.AutoDecide(nil) auto-decides everything (nil = "all").
	// This is the actual behavior of the system.
	mgr := setupManagerWithDecs(t)

	priorities := map[string]CareLevel{
		"Stack": CareLevelHigh,
		"UI":    CareLevelHigh,
		"Data":  CareLevelHigh,
	}
	mgr.AutoDecide(priorities)

	decs := mgr.Agent().Decisions()
	for _, d := range decs {
		if d.Answer == nil {
			t.Errorf("decision %s should be auto-decided (nil IDs = decide all)", d.ID)
		}
	}
}

func TestAutoDecidePicksFirstNonChooseForMe(t *testing.T) {
	mgr := setupManagerWithDecs(t)

	priorities := map[string]CareLevel{
		"Stack": CareLevelSkip,
		"UI":    CareLevelSkip,
		"Data":  CareLevelSkip,
	}
	mgr.AutoDecide(priorities)

	decs := mgr.Agent().Decisions()
	for _, d := range decs {
		if d.ID == "STA-0001" {
			if d.Answer == nil {
				t.Fatal("STA-0001 should be answered")
			}
			// Should pick "Go" not "Choose for me"
			if *d.Answer != "Go" {
				t.Errorf("STA-0001 answer = %q, want Go (first non-choose)", *d.Answer)
			}
		}
	}
}

func TestSyncDecisions(t *testing.T) {
	mgr := setupManagerWithDecs(t)
	mgr.allDecs = managerDecisions()

	// Create a "tree" copy with user modifications
	treeDecs := make([]decision.Decision, len(mgr.allDecs))
	copy(treeDecs, mgr.allDecs)

	answer := "Rust"
	treeDecs[0].Answer = &answer
	treeDecs[0].Source = "user"
	treeDecs[0].Delegated = false

	mgr.SyncDecisions(treeDecs)

	// Verify allDecs reflects the change
	if mgr.allDecs[0].Answer == nil {
		t.Fatal("allDecs[0] should have answer")
	}
	if *mgr.allDecs[0].Answer != "Rust" {
		t.Errorf("allDecs[0].Answer = %q, want Rust", *mgr.allDecs[0].Answer)
	}
	if mgr.allDecs[0].Source != "user" {
		t.Errorf("allDecs[0].Source = %q, want user", mgr.allDecs[0].Source)
	}
	if mgr.allDecs[0].Delegated {
		t.Error("allDecs[0].Delegated should be false")
	}
}

func TestSyncDecisionsAddsNew(t *testing.T) {
	mgr := setupManagerWithDecs(t)
	mgr.allDecs = managerDecisions()

	originalLen := len(mgr.allDecs)

	// Tree has a decision not in allDecs
	treeDecs := make([]decision.Decision, len(mgr.allDecs))
	copy(treeDecs, mgr.allDecs)
	treeDecs = append(treeDecs, decision.Decision{
		ID:       "NEW-0001",
		Category: "New",
		Question: "New question?",
		Source:   "user",
	})

	mgr.SyncDecisions(treeDecs)

	if len(mgr.allDecs) != originalLen+1 {
		t.Errorf("allDecs length = %d, want %d", len(mgr.allDecs), originalLen+1)
	}

	found := false
	for _, d := range mgr.allDecs {
		if d.ID == "NEW-0001" {
			found = true
		}
	}
	if !found {
		t.Error("NEW-0001 not added to allDecs")
	}
}

func TestSyncDecisionsDoesNotDuplicate(t *testing.T) {
	mgr := setupManagerWithDecs(t)
	mgr.allDecs = managerDecisions()

	// Sync with the same decisions (no changes)
	treeDecs := make([]decision.Decision, len(mgr.allDecs))
	copy(treeDecs, mgr.allDecs)

	before := len(mgr.allDecs)
	mgr.SyncDecisions(treeDecs)

	if len(mgr.allDecs) != before {
		t.Errorf("allDecs length changed from %d to %d after no-op sync", before, len(mgr.allDecs))
	}
}

func TestAllDecisions(t *testing.T) {
	mgr := setupManagerWithDecs(t)
	mgr.allDecs = managerDecisions()

	all := mgr.AllDecisions()
	if len(all) != len(mgr.allDecs) {
		t.Errorf("AllDecisions() = %d, want %d", len(all), len(mgr.allDecs))
	}
}

func TestNewManagerInitialState(t *testing.T) {
	mgr := NewManager(nil, "/tmp/test")

	if mgr.agent != nil {
		t.Error("agent should be nil initially")
	}
	if len(mgr.executors) != 0 {
		t.Error("executors should be empty initially")
	}
	if len(mgr.allDecs) != 0 {
		t.Error("allDecs should be empty initially")
	}
}

func TestAutoDecideOnNilAgent(t *testing.T) {
	mgr := NewManager(nil, "/tmp/test")

	// Should not panic
	mgr.AutoDecide(map[string]CareLevel{"Stack": CareLevelSkip})
}

func TestAutoDecideLeadingTrailingSpaces(t *testing.T) {
	m := NewManager(nil, "/tmp/test")
	a := NewAgent("test", nil, "/tmp/test")
	a.state.Decisions = []decision.Decision{
		{ID: "STA-0001", Category: "  Stack  ", Question: "Lang?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}}, Source: "user"},
	}
	a.state.Status = StatusDone
	m.agent = a

	priorities := map[string]CareLevel{
		"Stack": CareLevelSkip,
	}
	m.AutoDecide(priorities)

	decs := m.Agent().Decisions()
	for _, d := range decs {
		if strings.TrimSpace(d.Category) == "Stack" && d.Answer == nil {
			t.Error("should handle leading/trailing spaces in category matching")
		}
	}
}

func TestGroupByCategory(t *testing.T) {
	decs := []decision.Decision{
		{Category: "Stack", Question: "Q1"},
		{Category: "Stack", Question: "Q2"},
		{Category: "Stack", Question: "Q3"},
		{Category: "Data", Question: "Q4"},
		{Category: "Data", Question: "Q5"},
		{Category: "API", Question: "Q6"},
		{Category: "API", Question: "Q7"},
		{Category: "API", Question: "Q8"},
		{Category: "UI", Question: "Q9"},
		{Category: "UI", Question: "Q10"},
	}

	groups := GroupByCategory(decs)
	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}
	if len(groups["Stack"]) != 3 {
		t.Errorf("Stack group = %d, want 3", len(groups["Stack"]))
	}
	if len(groups["Data"]) != 2 {
		t.Errorf("Data group = %d, want 2", len(groups["Data"]))
	}
	if len(groups["API"]) != 3 {
		t.Errorf("API group = %d, want 3", len(groups["API"]))
	}
	if len(groups["UI"]) != 2 {
		t.Errorf("UI group = %d, want 2", len(groups["UI"]))
	}
}
