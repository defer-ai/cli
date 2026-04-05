package tui

import (
	"testing"

	"github.com/defer-ai/cli/internal/agent"
	"github.com/defer-ai/cli/internal/decision"
)

func priorityDecisions() []decision.Decision {
	return []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Lang?", Source: "user"},
		{ID: "STA-0002", Category: "Stack", Question: "Framework?", Source: "user"},
		{ID: "DAT-0001", Category: "Data", Question: "DB?", Source: "user"},
		{ID: "UII-0001", Category: "UI", Question: "CSS?", Source: "user"},
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

	// All should default to auto
	for _, cat := range pm.categories {
		if pm.priorities[cat] != agent.CareLevelAuto {
			t.Errorf("default priority for %q = %q, want auto", cat, pm.priorities[cat])
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

	if pm.priorities[cat] != agent.CareLevelAuto {
		t.Fatalf("initial = %q, want auto", pm.priorities[cat])
	}

	// l (right) increases to review
	pm, _ = pm.Update(keyRunes("l"))
	if pm.priorities[cat] != agent.CareLevelReview {
		t.Errorf("after right = %q, want review", pm.priorities[cat])
	}

	// l at max stays at review
	pm, _ = pm.Update(keyRunes("l"))
	if pm.priorities[cat] != agent.CareLevelReview {
		t.Errorf("after right x2 = %q, want review (capped)", pm.priorities[cat])
	}

	// h (left) decreases back to auto
	pm, _ = pm.Update(keyRunes("h"))
	if pm.priorities[cat] != agent.CareLevelAuto {
		t.Errorf("after left = %q, want auto", pm.priorities[cat])
	}

	// h at min stays at auto
	pm, _ = pm.Update(keyRunes("h"))
	if pm.priorities[cat] != agent.CareLevelAuto {
		t.Errorf("below min = %q, want auto (capped)", pm.priorities[cat])
	}
}

func TestPrioritiesFullRange(t *testing.T) {
	pm := newPriorities(priorityDecisions())
	cat := pm.categories[0]

	// Walk through all levels: auto -> review -> auto
	pm, _ = pm.Update(keyRunes("l")) // auto -> review
	if pm.priorities[cat] != agent.CareLevelReview {
		t.Errorf("got %q, want review", pm.priorities[cat])
	}

	pm, _ = pm.Update(keyRunes("h")) // review -> auto
	if pm.priorities[cat] != agent.CareLevelAuto {
		t.Errorf("got %q, want auto", pm.priorities[cat])
	}
}

func TestPrioritiesConfirm(t *testing.T) {
	pm := newPriorities(priorityDecisions())

	// Set Stack to review
	pm, _ = pm.Update(keyRunes("l")) // auto -> review

	// Move to Data, stays at auto
	pm, _ = pm.Update(keyRunes("j")) // cursor to Data

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

	if confirmed.Priorities[pm.categories[0]] != agent.CareLevelReview {
		t.Errorf("Stack = %q, want review", confirmed.Priorities[pm.categories[0]])
	}
	if confirmed.Priorities[pm.categories[1]] != agent.CareLevelAuto {
		t.Errorf("Data = %q, want auto", confirmed.Priorities[pm.categories[1]])
	}
	if confirmed.Priorities[pm.categories[2]] != agent.CareLevelAuto {
		t.Errorf("UI = %q, want auto (unchanged)", confirmed.Priorities[pm.categories[2]])
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

	// Set Stack to review, Data stays auto, UI stays auto
	pm, _ = pm.Update(keyRunes("l")) // Stack: auto -> review

	pm, _ = pm.Update(keyRunes("j")) // cursor to Data (stays auto)

	pm, _ = pm.Update(keyRunes("j")) // cursor to UI (stays auto)

	// Verify each is independent
	if pm.priorities["Stack"] != agent.CareLevelReview {
		t.Errorf("Stack = %q, want review", pm.priorities["Stack"])
	}
	if pm.priorities["Data"] != agent.CareLevelAuto {
		t.Errorf("Data = %q, want auto", pm.priorities["Data"])
	}
	if pm.priorities["UI"] != agent.CareLevelAuto {
		t.Errorf("UI = %q, want auto", pm.priorities["UI"])
	}
}

func TestPrioritiesView(t *testing.T) {
	pm := newPriorities(priorityDecisions())
	output := pm.View()
	if output == "" {
		t.Error("View() returned empty")
	}
}
