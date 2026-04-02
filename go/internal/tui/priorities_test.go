package tui

import (
	"testing"

	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

func priorityDecisions() []decision.Decision {
	return []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Lang?", Source: "user"},
		{ID: "@STA-0002", Category: "Stack", Question: "Framework?", Source: "user"},
		{ID: "@DAT-0001", Category: "Data", Question: "DB?", Source: "user"},
		{ID: "@UIX-0001", Category: "UI", Question: "CSS?", Source: "user"},
	}
}

func newPriorities(decs []decision.Decision) PrioritiesModel {
	pm := NewPrioritiesModel(decs)
	pm.width = 120
	pm.height = 40
	return pm
}

func TestPrioritiesInit(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	if len(pm.categories) != 3 {
		t.Fatalf("categories = %d, want 3", len(pm.categories))
	}

	// All should default to medium
	for _, cat := range pm.categories {
		if pm.priorities[cat] != agent.CareLevelMedium {
			t.Errorf("default priority for %q = %q, want medium", cat, pm.priorities[cat])
		}
	}

	// Verify counts
	if pm.counts["Stack"] != 2 {
		t.Errorf("Stack count = %d, want 2", pm.counts["Stack"])
	}
	if pm.counts["Data"] != 1 {
		t.Errorf("Data count = %d, want 1", pm.counts["Data"])
	}
	if pm.counts["UI"] != 1 {
		t.Errorf("UI count = %d, want 1", pm.counts["UI"])
	}
}

func TestPrioritiesNavigation(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	if pm.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", pm.cursor)
	}

	// j moves down
	pm, _ = pm.Update(keyRunes("j"))
	if pm.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", pm.cursor)
	}

	// k moves back
	pm, _ = pm.Update(keyRunes("k"))
	if pm.cursor != 0 {
		t.Errorf("cursor after k = %d, want 0", pm.cursor)
	}

	// k at top stays at 0
	pm, _ = pm.Update(keyRunes("k"))
	if pm.cursor != 0 {
		t.Errorf("cursor below 0 = %d, want 0", pm.cursor)
	}

	// j past end stays at last
	for i := 0; i < 10; i++ {
		pm, _ = pm.Update(keyRunes("j"))
	}
	if pm.cursor != 2 {
		t.Errorf("cursor overflow = %d, want 2 (3 categories, 0-indexed)", pm.cursor)
	}
}

func TestPrioritiesAdjust(t *testing.T) {
	pm := newPriorities(priorityDecisions())
	cat := pm.categories[0] // "Stack"

	if pm.priorities[cat] != agent.CareLevelMedium {
		t.Fatalf("initial = %q, want medium", pm.priorities[cat])
	}

	// l (right) increases
	pm, _ = pm.Update(keyRunes("l"))
	if pm.priorities[cat] != agent.CareLevelHigh {
		t.Errorf("after right = %q, want high", pm.priorities[cat])
	}

	// l again
	pm, _ = pm.Update(keyRunes("l"))
	if pm.priorities[cat] != agent.CareLevelParanoid {
		t.Errorf("after right x2 = %q, want paranoid", pm.priorities[cat])
	}

	// l at max stays at paranoid
	pm, _ = pm.Update(keyRunes("l"))
	if pm.priorities[cat] != agent.CareLevelParanoid {
		t.Errorf("after right x3 = %q, want paranoid (capped)", pm.priorities[cat])
	}

	// h (left) decreases
	pm, _ = pm.Update(keyRunes("h"))
	if pm.priorities[cat] != agent.CareLevelHigh {
		t.Errorf("after left = %q, want high", pm.priorities[cat])
	}

	// All the way to skip
	for i := 0; i < 10; i++ {
		pm, _ = pm.Update(keyRunes("h"))
	}
	if pm.priorities[cat] != agent.CareLevelSkip {
		t.Errorf("at min = %q, want skip", pm.priorities[cat])
	}

	// h at min stays at skip
	pm, _ = pm.Update(keyRunes("h"))
	if pm.priorities[cat] != agent.CareLevelSkip {
		t.Errorf("below min = %q, want skip (capped)", pm.priorities[cat])
	}
}

func TestPrioritiesFullRange(t *testing.T) {
	pm := newPriorities(priorityDecisions())
	cat := pm.categories[0]

	// Walk through all levels: medium -> low -> skip -> low -> medium -> high -> paranoid
	pm, _ = pm.Update(keyRunes("h")) // medium -> low
	if pm.priorities[cat] != agent.CareLevelLow {
		t.Errorf("got %q, want low", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("h")) // low -> skip
	if pm.priorities[cat] != agent.CareLevelSkip {
		t.Errorf("got %q, want skip", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("l")) // skip -> low
	if pm.priorities[cat] != agent.CareLevelLow {
		t.Errorf("got %q, want low", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("l")) // low -> medium
	if pm.priorities[cat] != agent.CareLevelMedium {
		t.Errorf("got %q, want medium", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("l")) // medium -> high
	if pm.priorities[cat] != agent.CareLevelHigh {
		t.Errorf("got %q, want high", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("l")) // high -> paranoid
	if pm.priorities[cat] != agent.CareLevelParanoid {
		t.Errorf("got %q, want paranoid", pm.priorities[cat])
	}
}

func TestPrioritiesConfirm(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	// Set Stack to paranoid
	pm, _ = pm.Update(keyRunes("l")) // medium -> high
	pm, _ = pm.Update(keyRunes("l")) // high -> paranoid

	// Move to Data, set to skip
	pm, _ = pm.Update(keyRunes("j")) // cursor to Data
	pm, _ = pm.Update(keyRunes("h")) // medium -> low
	pm, _ = pm.Update(keyRunes("h")) // low -> skip

	// Press enter to confirm
	pm, cmd := pm.Update(keyEnter())

	if cmd == nil {
		t.Fatal("expected cmd from enter")
	}

	msg := cmd()
	confirmed, ok := msg.(PrioritiesConfirmedMsg)
	if !ok {
		t.Fatalf("expected PrioritiesConfirmedMsg, got %T", msg)
	}

	if confirmed.Priorities[pm.categories[0]] != agent.CareLevelParanoid {
		t.Errorf("Stack = %q, want paranoid", confirmed.Priorities[pm.categories[0]])
	}
	if confirmed.Priorities[pm.categories[1]] != agent.CareLevelSkip {
		t.Errorf("Data = %q, want skip", confirmed.Priorities[pm.categories[1]])
	}
	if confirmed.Priorities[pm.categories[2]] != agent.CareLevelMedium {
		t.Errorf("UI = %q, want medium (unchanged)", confirmed.Priorities[pm.categories[2]])
	}

	_ = pm
}

func TestPrioritiesEscConfirms(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	pm, cmd := pm.Update(keyEsc())
	if cmd == nil {
		t.Fatal("esc should also confirm priorities")
	}

	msg := cmd()
	if _, ok := msg.(PrioritiesConfirmedMsg); !ok {
		t.Fatalf("expected PrioritiesConfirmedMsg, got %T", msg)
	}
}

func TestPrioritiesPerCategory(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	// Set Stack to high, Data stays medium, UI to skip
	pm, _ = pm.Update(keyRunes("l")) // Stack: medium -> high

	pm, _ = pm.Update(keyRunes("j")) // cursor to Data (stays medium)

	pm, _ = pm.Update(keyRunes("j")) // cursor to UI
	pm, _ = pm.Update(keyRunes("h")) // UI: medium -> low
	pm, _ = pm.Update(keyRunes("h")) // UI: low -> skip

	// Verify each is independent
	if pm.priorities["Stack"] != agent.CareLevelHigh {
		t.Errorf("Stack = %q, want high", pm.priorities["Stack"])
	}
	if pm.priorities["Data"] != agent.CareLevelMedium {
		t.Errorf("Data = %q, want medium", pm.priorities["Data"])
	}
	if pm.priorities["UI"] != agent.CareLevelSkip {
		t.Errorf("UI = %q, want skip", pm.priorities["UI"])
	}
}

func TestPrioritiesView(t *testing.T) {
	pm := newPriorities(priorityDecisions())
	output := pm.View()
	if output == "" {
		t.Error("View() returned empty")
	}
}
