package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func TestAutoDecideWithPriorities(t *testing.T) {
	a := NewAgent("test", nil, ".")
	a.state.Decisions = []decision.Decision{
		{ID: "STACK-001", Category: "Stack", Question: "Language?", Options: []decision.DecisionOption{
			{Key: "A", Label: "TypeScript"},
			{Key: "B", Label: "Choose for me"},
		}},
		{ID: "UI-001", Category: "UI", Question: "Framework?", Options: []decision.DecisionOption{
			{Key: "A", Label: "React"},
			{Key: "B", Label: "Vue"},
		}},
		{ID: "DATA-001", Category: "Data", Question: "Database?", Options: []decision.DecisionOption{
			{Key: "A", Label: "PostgreSQL"},
		}},
	}

	// Skip Stack, paranoid UI, medium Data
	priorities := map[string]CareLevel{
		"Stack": CareLevelSkip,
		"UI":    CareLevelParanoid,
		"Data":  CareLevelMedium,
	}

	// Simulate Manager.AutoDecide
	priMap := make(map[string]CareLevel)
	for k, v := range priorities {
		priMap[strings.ToLower(strings.TrimSpace(k))] = v
	}

	var autoIDs []string
	for _, d := range a.Decisions() {
		if d.Answer != nil {
			continue
		}
		level := priMap[strings.ToLower(strings.TrimSpace(d.Category))]
		t.Logf("Decision %s category=%q level=%q", d.ID, d.Category, level)
		if level != CareLevelParanoid && level != CareLevelHigh {
			autoIDs = append(autoIDs, d.ID)
		}
	}

	t.Logf("AutoDecide IDs: %v", autoIDs)
	a.AutoDecide(autoIDs)

	for _, d := range a.Decisions() {
		t.Logf("After: %s answer=%v delegated=%v", d.ID, d.Answer, d.Delegated)
	}

	// Stack should be auto-decided (skip)
	if a.state.Decisions[0].Answer == nil {
		t.Error("STACK-001 should be auto-decided")
	}

	// UI should still be pending (paranoid)
	if a.state.Decisions[1].Answer != nil {
		t.Error("UI-001 should still be pending (paranoid)")
	}

	// Data should be auto-decided (medium)
	if a.state.Decisions[2].Answer == nil {
		t.Error("DATA-001 should be auto-decided")
	}

	fmt.Println("Test passed!")
}
