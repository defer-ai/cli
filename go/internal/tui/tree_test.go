package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/decision"
)

func fiveDecisions() []decision.Decision {
	return []decision.Decision{
		{ID: "S-001", Category: "Stack", Question: "Language?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}, {Key: "B", Label: "Rust"}}, Source: "user"},
		{ID: "S-002", Category: "Stack", Question: "Framework?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Gin"}, {Key: "B", Label: "Echo"}}, Source: "user"},
		{ID: "D-001", Category: "Data", Question: "Database?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Postgres"}, {Key: "B", Label: "MySQL"}}, Source: "user"},
		{ID: "D-002", Category: "Data", Question: "ORM?",
			Options: []decision.DecisionOption{{Key: "A", Label: "GORM"}, {Key: "B", Label: "sqlx"}}, Source: "user"},
		{ID: "U-001", Category: "UI", Question: "CSS?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Tailwind"}, {Key: "B", Label: "CSS Modules"}}, Source: "user"},
	}
}

func newTree(decs []decision.Decision) TreeModel {
	tm := NewTreeModel()
	tm.decisions = decs
	tm.width = 120
	tm.height = 40
	return tm
}

// updateTree wraps tree.Update for cleaner test code.
func updateTree(t *testing.T, m TreeModel, msg tea.Msg) (TreeModel, tea.Cmd) {
	t.Helper()
	return m.Update(msg)
}

func TestTreeNavigation(t *testing.T) {
	tm := newTree(fiveDecisions())

	if tm.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", tm.cursor)
	}

	// Press j 3 times
	for i := 0; i < 3; i++ {
		tm, _ = updateTree(t, tm, keyRunes("j"))
	}
	if tm.cursor != 3 {
		t.Errorf("cursor after 3 j = %d, want 3", tm.cursor)
	}

	// Press k twice
	for i := 0; i < 2; i++ {
		tm, _ = updateTree(t, tm, keyRunes("k"))
	}
	if tm.cursor != 1 {
		t.Errorf("cursor after 2 k = %d, want 1", tm.cursor)
	}

	// Press j 10 times -- should cap at 4 (5 decisions, 0-indexed)
	for i := 0; i < 10; i++ {
		tm, _ = updateTree(t, tm, keyRunes("j"))
	}
	if tm.cursor != 4 {
		t.Errorf("cursor after overflow = %d, want 4", tm.cursor)
	}
}

func TestTreeNavigationAtBoundaries(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Press k at top -- should stay at 0
	tm, _ = updateTree(t, tm, keyRunes("k"))
	if tm.cursor != 0 {
		t.Errorf("cursor after k at top = %d, want 0", tm.cursor)
	}
}

func TestTreeEnterOpensDetail(t *testing.T) {
	tm := newTree(fiveDecisions())

	if tm.mode != tmTree {
		t.Fatalf("initial mode = %d, want tmTree", tm.mode)
	}

	tm, _ = updateTree(t, tm, keyEnter())

	if tm.mode != tmDetail {
		t.Errorf("mode after enter = %d, want tmDetail (%d)", tm.mode, tmDetail)
	}
}

func TestDetailEscGoesBack(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // enter detail

	if tm.mode != tmDetail {
		t.Fatalf("mode = %d, want tmDetail", tm.mode)
	}

	tm, _ = updateTree(t, tm, keyEsc())

	if tm.mode != tmTree {
		t.Errorf("mode after esc = %d, want tmTree (%d)", tm.mode, tmTree)
	}
}

func TestDetailQGoesBack(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // enter detail
	tm, _ = updateTree(t, tm, keyRunes("q"))

	if tm.mode != tmTree {
		t.Errorf("mode after q = %d, want tmTree", tm.mode)
	}
}

func TestDetailOptionPicking(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail for S-001

	if tm.mode != tmDetail {
		t.Fatalf("not in detail mode")
	}

	// Move to option index 1 (second option: "Rust")
	tm, _ = updateTree(t, tm, keyRunes("j"))
	if tm.optCursor != 1 {
		t.Fatalf("optCursor = %d, want 1", tm.optCursor)
	}

	// Press enter to select
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())

	if cmd == nil {
		t.Fatal("expected cmd from option selection")
	}

	msg := cmd()
	revise, ok := msg.(ReviseDecisionMsg)
	if !ok {
		t.Fatalf("expected ReviseDecisionMsg, got %T", msg)
	}
	if revise.ID != "S-001" {
		t.Errorf("revise.ID = %q, want S-001", revise.ID)
	}
	if revise.NewAnswer != "Rust" {
		t.Errorf("revise.NewAnswer = %q, want Rust", revise.NewAnswer)
	}

	if tm.mode != tmTree {
		t.Errorf("mode = %d, want tmTree (should return to tree after pick)", tm.mode)
	}
}

func TestDetailOptionPickingOnAnsweredDecision(t *testing.T) {
	decs := fiveDecisions()
	answer := "Go"
	decs[0].Answer = &answer
	decs[0].Source = "auto"

	tm := newTree(decs)
	tm, _ = updateTree(t, tm, keyEnter()) // detail for S-001 (already answered)

	// Navigate to second option and pick it
	tm, _ = updateTree(t, tm, keyRunes("j"))
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())

	if cmd == nil {
		t.Fatal("expected cmd -- should be able to change answered decisions")
	}

	msg := cmd()
	revise, ok := msg.(ReviseDecisionMsg)
	if !ok {
		t.Fatalf("expected ReviseDecisionMsg, got %T", msg)
	}
	if revise.NewAnswer != "Rust" {
		t.Errorf("revise.NewAnswer = %q, want Rust", revise.NewAnswer)
	}
}

func TestDetailCustomRevise(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	// Press 'c' for custom
	tm, _ = updateTree(t, tm, keyRunes("c"))
	if tm.mode != tmRevise {
		t.Fatalf("mode = %d, want tmRevise (%d)", tm.mode, tmRevise)
	}

	// Type custom answer
	for _, ch := range "my custom answer" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}
	if tm.textBuf != "my custom answer" {
		t.Fatalf("textBuf = %q, want %q", tm.textBuf, "my custom answer")
	}

	// Press enter
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())

	if cmd == nil {
		t.Fatal("expected cmd from custom revise")
	}

	msg := cmd()
	revise, ok := msg.(ReviseDecisionMsg)
	if !ok {
		t.Fatalf("expected ReviseDecisionMsg, got %T", msg)
	}
	if revise.NewAnswer != "my custom answer" {
		t.Errorf("revise.NewAnswer = %q, want %q", revise.NewAnswer, "my custom answer")
	}
	if tm.mode != tmTree {
		t.Errorf("mode = %d, want tmTree after submitting revise", tm.mode)
	}
}

func TestDetailCustomReviseBackspace(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter())
	tm, _ = updateTree(t, tm, keyRunes("c"))

	// Type then backspace
	for _, ch := range "hello" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyBackspace})
	if tm.textBuf != "hell" {
		t.Errorf("textBuf = %q, want %q", tm.textBuf, "hell")
	}
}

func TestDetailCustomReviseEscCancels(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter())
	tm, _ = updateTree(t, tm, keyRunes("c"))

	for _, ch := range "test" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	tm, _ = updateTree(t, tm, keyEsc())
	if tm.mode != tmDetail {
		t.Errorf("mode = %d, want tmDetail after esc from revise", tm.mode)
	}
	if tm.textBuf != "" {
		t.Errorf("textBuf = %q, want empty after cancel", tm.textBuf)
	}
}

func TestDetailAsk(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	// Press 'a' to ask
	tm, _ = updateTree(t, tm, keyRunes("a"))
	if tm.mode != tmAsk {
		t.Fatalf("mode = %d, want tmAsk (%d)", tm.mode, tmAsk)
	}

	// Type question
	for _, ch := range "why this" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Press enter
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())
	if cmd == nil {
		t.Fatal("expected cmd from ask")
	}

	msg := cmd()
	ask, ok := msg.(AskDecisionMsg)
	if !ok {
		t.Fatalf("expected AskDecisionMsg, got %T", msg)
	}
	if ask.ID != "S-001" {
		t.Errorf("ask.ID = %q, want S-001", ask.ID)
	}
	if ask.Question != "why this" {
		t.Errorf("ask.Question = %q, want %q", ask.Question, "why this")
	}
	if tm.mode != tmDetail {
		t.Errorf("mode = %d, want tmDetail after ask submit", tm.mode)
	}
}

func TestDetailWhy(t *testing.T) {
	decs := fiveDecisions()
	decs[0].Answer = strPtr("Go")

	tm := newTree(decs)
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyRunes("w"))

	if cmd == nil {
		t.Fatal("expected cmd from why")
	}

	msg := cmd()
	why, ok := msg.(WhyDecisionMsg)
	if !ok {
		t.Fatalf("expected WhyDecisionMsg, got %T", msg)
	}
	if why.ID != "S-001" {
		t.Errorf("why.ID = %q, want S-001", why.ID)
	}
}

func TestDetailWhyWithNoAnswerUsesOptionCursor(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	// optCursor defaults to 0, so it should pick the first option label
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyRunes("w"))

	if cmd == nil {
		t.Fatal("expected cmd from why with option cursor")
	}

	msg := cmd()
	why, ok := msg.(WhyDecisionMsg)
	if !ok {
		t.Fatalf("expected WhyDecisionMsg, got %T", msg)
	}
	if why.Label != "Go" {
		t.Errorf("why.Label = %q, want Go", why.Label)
	}
}

func TestDetailShuffle(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyRunes("s"))

	if cmd == nil {
		t.Fatal("expected cmd from shuffle")
	}

	msg := cmd()
	suggest, ok := msg.(SuggestDecisionMsg)
	if !ok {
		t.Fatalf("expected SuggestDecisionMsg, got %T", msg)
	}
	if suggest.ID != "S-001" {
		t.Errorf("suggest.ID = %q, want S-001", suggest.ID)
	}
}

func TestTabTogglesChatFocus(t *testing.T) {
	tm := newTree(fiveDecisions())

	if tm.chatFocused {
		t.Fatal("chat should not be focused initially")
	}

	tm, _ = updateTree(t, tm, keyTab())
	if !tm.chatFocused {
		t.Error("chat should be focused after tab")
	}

	tm, _ = updateTree(t, tm, keyTab())
	if tm.chatFocused {
		t.Error("chat should not be focused after second tab")
	}
}

func TestChatInputEscUnfocuses(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyTab()) // focus chat
	if !tm.chatFocused {
		t.Fatal("chat should be focused")
	}
	tm, _ = updateTree(t, tm, keyEsc()) // unfocus
	if tm.chatFocused {
		t.Error("esc should unfocus chat")
	}
}

func TestDecisionItemsFiltersMisc(t *testing.T) {
	decs := fiveDecisions()
	miscAnswer := "(catch-all)"
	decs = append(decs, decision.Decision{
		ID:       "MISC-001",
		Category: "Misc",
		Question: "Uncategorized implementation decisions",
		Answer:   &miscAnswer,
		Source:   "auto",
	})

	tm := newTree(decs)
	items := tm.decisionItems()

	for _, item := range items {
		if item.ID == "MISC-001" {
			t.Error("Misc placeholder should be filtered from decisionItems()")
		}
	}

	if len(items) != 5 {
		t.Errorf("decisionItems() = %d, want 5 (6 minus misc)", len(items))
	}
}

func TestSelectedWithinBounds(t *testing.T) {
	tm := newTree(fiveDecisions())

	sel := tm.selected()
	if sel == nil {
		t.Fatal("selected() returned nil at cursor 0")
	}
	if sel.ID != "S-001" {
		t.Errorf("selected().ID = %q, want S-001", sel.ID)
	}

	// Move to last
	tm.cursor = 4
	sel = tm.selected()
	if sel == nil {
		t.Fatal("selected() returned nil at cursor 4")
	}
	if sel.ID != "U-001" {
		t.Errorf("selected().ID = %q, want U-001", sel.ID)
	}
}

func TestSelectedOutOfBounds(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.cursor = 100

	sel := tm.selected()
	if sel != nil {
		t.Error("selected() should return nil for out-of-bounds cursor")
	}
}

func TestEmptyTreeNoSelection(t *testing.T) {
	tm := newTree(nil)
	sel := tm.selected()
	if sel != nil {
		t.Error("selected() should return nil for empty tree")
	}
}

func TestEnterOnEmptyTreeNoPanic(t *testing.T) {
	tm := newTree(nil)
	tm, _ = updateTree(t, tm, keyEnter())

	// Should stay in tree mode, no panic
	if tm.mode != tmTree {
		t.Errorf("mode = %d, want tmTree for empty tree", tm.mode)
	}
}

func TestOptionCursorBoundary(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail for S-001 (2 options)

	// Navigate past last option
	for i := 0; i < 10; i++ {
		tm, _ = updateTree(t, tm, keyRunes("j"))
	}
	if tm.optCursor != 1 {
		t.Errorf("optCursor = %d, want 1 (capped at last option)", tm.optCursor)
	}

	// Navigate above first option
	for i := 0; i < 10; i++ {
		tm, _ = updateTree(t, tm, keyRunes("k"))
	}
	if tm.optCursor != 0 {
		t.Errorf("optCursor = %d, want 0 (capped at first option)", tm.optCursor)
	}
}

func TestViewRendersAllModes(t *testing.T) {
	modes := []struct {
		name string
		mode treeMode
	}{
		{"tree", tmTree},
		{"detail", tmDetail},
		{"revise", tmRevise},
		{"ask", tmAsk},
		{"chat", tmChat},
	}

	for _, mm := range modes {
		t.Run(mm.name, func(t *testing.T) {
			tm := newTree(fiveDecisions())
			tm.mode = mm.mode
			output := tm.View()
			if output == "" {
				t.Errorf("View() returned empty for mode %s", mm.name)
			}
		})
	}
}

func TestCustomReviseEmptyDoesNothing(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter())
	tm, _ = updateTree(t, tm, keyRunes("c"))

	// Press enter without typing anything
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())

	if cmd != nil {
		t.Error("empty custom revise should not produce a cmd")
	}
}
