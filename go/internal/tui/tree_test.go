package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/decision"
)

func fiveDecisions() []decision.Decision {
	return []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Language?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Go"}, {Key: "B", Label: "Rust"}}, Source: "user"},
		{ID: "@STA-0002", Category: "Stack", Question: "Framework?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Gin"}, {Key: "B", Label: "Echo"}}, Source: "user"},
		{ID: "@DAT-0001", Category: "Data", Question: "Database?",
			Options: []decision.DecisionOption{{Key: "A", Label: "Postgres"}, {Key: "B", Label: "MySQL"}}, Source: "user"},
		{ID: "@DAT-0002", Category: "Data", Question: "ORM?",
			Options: []decision.DecisionOption{{Key: "A", Label: "GORM"}, {Key: "B", Label: "sqlx"}}, Source: "user"},
		{ID: "@UIX-0001", Category: "UI", Question: "CSS?",
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
	tm, _ = updateTree(t, tm, keyEnter()) // detail for STA-0001

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
	if revise.ID != "@STA-0001" {
		t.Errorf("revise.ID = %q, want STA-0001", revise.ID)
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
	tm, _ = updateTree(t, tm, keyEnter()) // detail for STA-0001 (already answered)

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
	if tm.textInput.Value() != "my custom answer" {
		t.Fatalf("textInput.Value() = %q, want %q", tm.textInput.Value(), "my custom answer")
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
	if tm.textInput.Value() != "hell" {
		t.Errorf("textInput.Value() = %q, want %q", tm.textInput.Value(), "hell")
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
	if tm.textInput.Value() != "" {
		t.Errorf("textInput.Value() = %q, want empty after cancel", tm.textInput.Value())
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
	if ask.ID != "@STA-0001" {
		t.Errorf("ask.ID = %q, want STA-0001", ask.ID)
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
	if why.ID != "@STA-0001" {
		t.Errorf("why.ID = %q, want STA-0001", why.ID)
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
	if suggest.ID != "@STA-0001" {
		t.Errorf("suggest.ID = %q, want STA-0001", suggest.ID)
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
		ID:       "MIS-0001",
		Category: "Misc",
		Question: "Uncategorized implementation decisions",
		Answer:   &miscAnswer,
		Source:   "auto",
	})

	tm := newTree(decs)
	items := tm.decisionItems()

	for _, item := range items {
		if item.ID == "MIS-0001" {
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
	if sel.ID != "@STA-0001" {
		t.Errorf("selected().ID = %q, want STA-0001", sel.ID)
	}

	// Move to last
	tm.cursor = 4
	sel = tm.selected()
	if sel == nil {
		t.Fatal("selected() returned nil at cursor 4")
	}
	if sel.ID != "@UIX-0001" {
		t.Errorf("selected().ID = %q, want UIX-0001", sel.ID)
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
	tm, _ = updateTree(t, tm, keyEnter()) // detail for STA-0001 (2 options)

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
		{"editFeatures", tmEditFeatures},
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

// ========== Search feature tests ==========

func TestSearchModeActivation(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Press "/" to enter search mode
	tm, _ = updateTree(t, tm, keyRunes("/"))
	if !tm.searchMode {
		t.Error("search mode should be active after pressing /")
	}
}

func TestSearchModeEscClearsFilter(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Enter search mode
	tm, _ = updateTree(t, tm, keyRunes("/"))

	// Type a query
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}
	if tm.searchQuery == "" {
		t.Fatal("searchQuery should be set after typing")
	}

	// Esc should clear filter
	tm, _ = updateTree(t, tm, keyEsc())
	if tm.searchMode {
		t.Error("search mode should be inactive after esc")
	}
	if tm.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty after esc", tm.searchQuery)
	}
}

func TestSearchModeEnterKeepsFilter(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Enter search mode and type
	tm, _ = updateTree(t, tm, keyRunes("/"))
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Enter should exit search mode but keep the filter
	tm, _ = updateTree(t, tm, keyEnter())
	if tm.searchMode {
		t.Error("search mode should be inactive after enter")
	}
	if tm.searchQuery != "data" {
		t.Errorf("searchQuery = %q, want %q after enter", tm.searchQuery, "data")
	}
}

func TestSearchFiltersDecisions(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Enter search mode and search for "data"
	tm, _ = updateTree(t, tm, keyRunes("/"))
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Should filter to Data category (DAT-0001, DAT-0002)
	items := tm.decisionItems()
	if len(items) != 2 {
		t.Errorf("filtered items = %d, want 2 (Data category)", len(items))
	}
	for _, item := range items {
		if item.Category != "Data" {
			t.Errorf("unexpected category %q in filtered results", item.Category)
		}
	}
}

func TestSearchFiltersByID(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchQuery = "@STA-0001"

	items := tm.decisionItems()
	if len(items) != 1 {
		t.Errorf("filtered items = %d, want 1", len(items))
	}
	if len(items) > 0 && items[0].ID != "@STA-0001" {
		t.Errorf("filtered item ID = %q, want STA-0001", items[0].ID)
	}
}

func TestSearchFiltersByQuestion(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchQuery = "css"

	items := tm.decisionItems()
	if len(items) != 1 {
		t.Errorf("filtered items = %d, want 1", len(items))
	}
	if len(items) > 0 && items[0].ID != "@UIX-0001" {
		t.Errorf("filtered item ID = %q, want UIX-0001", items[0].ID)
	}
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchQuery = "STACK"

	items := tm.decisionItems()
	if len(items) != 2 {
		t.Errorf("filtered items = %d, want 2 (Stack category)", len(items))
	}
}

func TestSearchCursorClampedAfterFilter(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.cursor = 4 // last item

	// Enter search mode and filter down to 2 items
	tm, _ = updateTree(t, tm, keyRunes("/"))
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Cursor should be clamped to filtered results
	if tm.cursor >= 2 {
		t.Errorf("cursor = %d, want < 2 (only 2 filtered results)", tm.cursor)
	}
}

func TestSearchViewShowsFilteredCount(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchQuery = "data"

	output := tm.viewTree()
	if !strings.Contains(output, "Filtered: 2 results") {
		t.Error("tree view should show 'Filtered: 2 results' when filter is active")
	}
}

func TestSearchViewShowsSearchInput(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchMode = true
	tm.searchInput.Focus()

	output := tm.viewTree()
	if !strings.Contains(output, "Filter decisions") {
		t.Error("tree view should show search input placeholder when search mode is active")
	}
}

// ========== Contextual footer tests ==========

func TestRenderFooterBasic(t *testing.T) {
	actions := []footerAction{
		{"enter", "confirm"},
		{"esc", "cancel"},
	}
	result := renderFooter(actions, 80)
	if !strings.Contains(result, "enter") {
		t.Error("footer should contain 'enter'")
	}
	if !strings.Contains(result, "confirm") {
		t.Error("footer should contain 'confirm'")
	}
	if !strings.Contains(result, "esc") {
		t.Error("footer should contain 'esc'")
	}
}

func TestRenderFooterTruncates(t *testing.T) {
	actions := []footerAction{
		{"a", "first"},
		{"b", "second"},
		{"c", "third"},
		{"d", "fourth"},
	}
	// Very narrow width: only first action should fit
	result := renderFooter(actions, 15)
	if !strings.Contains(result, "first") {
		t.Error("footer should contain at least the first action")
	}
	if strings.Contains(result, "fourth") {
		t.Error("footer should not contain 'fourth' in narrow width")
	}
}

func TestRenderFooterEmpty(t *testing.T) {
	result := renderFooter(nil, 80)
	// Should return just the prefix
	if result != " " {
		t.Errorf("empty footer = %q, want %q", result, " ")
	}
}

func TestTreeFooterShowsSearch(t *testing.T) {
	tm := newTree(fiveDecisions())
	output := tm.viewTree()
	if !strings.Contains(output, "filter") {
		t.Error("tree footer should contain '/ filter'")
	}
}

func TestTreeFooterSearchMode(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.searchMode = true
	tm.searchInput.Focus()

	output := tm.viewTree()
	if !strings.Contains(output, "to filter") {
		t.Error("search mode footer should contain 'to filter'")
	}
	if !strings.Contains(output, "confirm") {
		t.Error("search mode footer should contain 'confirm'")
	}
}

func TestDetailFooterShowsActions(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmDetail
	output := tm.viewDetail()
	if !strings.Contains(output, "custom") {
		t.Error("detail footer should contain 'custom'")
	}
	if !strings.Contains(output, "shuffle") {
		t.Error("detail footer should contain 'shuffle'")
	}
}

func TestReviseFooterShowsSubmitCancel(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmRevise
	output := tm.viewDetail()
	if !strings.Contains(output, "submit") {
		t.Error("revise footer should contain 'submit'")
	}
	if !strings.Contains(output, "cancel") {
		t.Error("revise footer should contain 'cancel'")
	}
}

func TestChatFooterShowsActions(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmChat
	output := tm.viewChat()
	if !strings.Contains(output, "send") {
		t.Error("chat footer should contain 'send'")
	}
	if !strings.Contains(output, "reference") {
		t.Error("chat footer should contain 'reference'")
	}
}

// ========== Split-pane tests ==========

func TestSplitPaneRendersOnWideTerminal(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // enter detail

	if tm.mode != tmDetail {
		t.Fatalf("mode = %d, want tmDetail", tm.mode)
	}

	output := tm.View()
	// Split pane should contain both the tree content and the detail content
	// The detail pane shows the decision ID as a title in the border
	if !strings.Contains(output, "@STA-0001") {
		t.Error("split pane should contain the selected decision ID")
	}
	// The left pane should still show other decisions
	if !strings.Contains(output, "@STA-0002") {
		t.Error("split pane should show tree with other decisions")
	}
}

func TestNarrowTerminalUsesFullScreenDetail(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 80 // narrow: <= 100
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // enter detail

	output := tm.View()
	// Full-screen detail should show the detail-specific footer actions
	if !strings.Contains(output, "shuffle") {
		t.Error("narrow terminal detail should show 'shuffle' action (full-screen detail)")
	}
}

func TestSplitPaneDetailPaneShowsOptions(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // detail for STA-0001

	output := tm.View()
	// Should show options from STA-0001 (Go, Rust)
	if !strings.Contains(output, "Go") {
		t.Error("detail pane should show option 'Go'")
	}
	if !strings.Contains(output, "Rust") {
		t.Error("detail pane should show option 'Rust'")
	}
}

func TestSplitPaneOptionNavigation(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	// Navigate options with j/k -- should still work in split pane
	tm, _ = updateTree(t, tm, keyRunes("j"))
	if tm.optCursor != 1 {
		t.Errorf("optCursor = %d, want 1 after j in split pane", tm.optCursor)
	}
	tm, _ = updateTree(t, tm, keyRunes("k"))
	if tm.optCursor != 0 {
		t.Errorf("optCursor = %d, want 0 after k in split pane", tm.optCursor)
	}
}

func TestSplitPaneReviseGoesFullScreen(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // detail
	tm, _ = updateTree(t, tm, keyRunes("c"))  // revise mode

	if tm.mode != tmRevise {
		t.Fatalf("mode = %d, want tmRevise", tm.mode)
	}

	output := tm.View()
	// Revise mode always uses full-screen detail, which has 'submit' in footer
	if !strings.Contains(output, "submit") {
		t.Error("revise mode should render full-screen detail with 'submit'")
	}
}

func TestSplitPaneAskGoesFullScreen(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // detail
	tm, _ = updateTree(t, tm, keyRunes("a"))  // ask mode

	if tm.mode != tmAsk {
		t.Fatalf("mode = %d, want tmAsk", tm.mode)
	}

	output := tm.View()
	if !strings.Contains(output, "submit") {
		t.Error("ask mode should render full-screen detail with 'submit'")
	}
}

func TestDetailPaneShowsAnswer(t *testing.T) {
	decs := fiveDecisions()
	answer := "Go"
	decs[0].Answer = &answer
	decs[0].Source = "user"

	tm := newTree(decs)
	tm.width = 140
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	output := tm.View()
	if !strings.Contains(output, "Go") {
		t.Error("detail pane should show the current answer")
	}
}

func TestWrapTextBasic(t *testing.T) {
	lines := wrapText("hello world this is a test", 12)
	if len(lines) < 2 {
		t.Errorf("wrapText produced %d lines, want >= 2 for width 12", len(lines))
	}
	for _, l := range lines {
		if len(l) > 12 {
			t.Errorf("wrapped line %q exceeds width 12", l)
		}
	}
}

func TestWrapTextShortString(t *testing.T) {
	lines := wrapText("hello", 80)
	if len(lines) != 1 {
		t.Errorf("wrapText produced %d lines, want 1 for short string", len(lines))
	}
	if lines[0] != "hello" {
		t.Errorf("wrapText = %q, want %q", lines[0], "hello")
	}
}

func TestWrapTextEmpty(t *testing.T) {
	lines := wrapText("", 80)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("wrapText empty = %v, want [\"\"]", lines)
	}
}

// ========== @ID autocomplete tests ==========

func TestGetCompletionsMatchingIDs(t *testing.T) {
	decs := fiveDecisions() // STA-0001, STA-0002, DAT-0001, DAT-0002, UIX-0001
	matches := getCompletions(decs, "STA")
	if len(matches) != 2 {
		t.Errorf("getCompletions(STA) = %d, want 2", len(matches))
	}
	for _, m := range matches {
		if !strings.HasPrefix(m, "@STA") {
			t.Errorf("unexpected match %q for prefix STA", m)
		}
	}
}

func TestGetCompletionsCaseInsensitive(t *testing.T) {
	decs := fiveDecisions()
	matches := getCompletions(decs, "sta")
	if len(matches) != 2 {
		t.Errorf("getCompletions(sta) = %d, want 2 (case-insensitive)", len(matches))
	}
	matches2 := getCompletions(decs, "dat-00")
	if len(matches2) != 2 {
		t.Errorf("getCompletions(dat-00) = %d, want 2", len(matches2))
	}
}

func TestGetCompletionsMax5(t *testing.T) {
	// Create 8 decisions with the same prefix
	var decs []decision.Decision
	for i := 0; i < 8; i++ {
		decs = append(decs, decision.Decision{
			ID:       fmt.Sprintf("TEST-%03d", i+1),
			Category: "Test",
			Question: fmt.Sprintf("Question %d?", i+1),
			Source:   "user",
		})
	}
	matches := getCompletions(decs, "TEST")
	if len(matches) != 5 {
		t.Errorf("getCompletions(TEST) = %d, want 5 (max)", len(matches))
	}
}

func TestGetCompletionsEmptyPartial(t *testing.T) {
	decs := fiveDecisions()
	matches := getCompletions(decs, "")
	if len(matches) != 0 {
		t.Errorf("getCompletions('') = %d, want 0", len(matches))
	}
}

func TestGetCompletionsNoMatch(t *testing.T) {
	decs := fiveDecisions()
	matches := getCompletions(decs, "ZZZZZ")
	if len(matches) != 0 {
		t.Errorf("getCompletions(ZZZZZ) = %d, want 0", len(matches))
	}
}

func TestTabCyclesThroughCompletions(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Switch to chat mode
	tm, _ = updateTree(t, tm, keyTab())
	if tm.mode != tmChat {
		t.Fatalf("mode = %d, want tmChat", tm.mode)
	}

	// Type "@STA" to trigger completions
	for _, ch := range "@STA" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	if len(tm.completions) != 2 {
		t.Fatalf("completions = %d, want 2", len(tm.completions))
	}
	if tm.completionIdx != -1 {
		t.Fatalf("completionIdx = %d, want -1 (none selected yet)", tm.completionIdx)
	}

	// First tab: select index 0
	tm, _ = updateTree(t, tm, keyTab())
	if tm.completionIdx != 0 {
		t.Errorf("completionIdx after 1st tab = %d, want 0", tm.completionIdx)
	}
	if !strings.Contains(tm.chatInput.Value(), "@"+tm.completions[0]) {
		t.Errorf("input = %q, should contain @%s", tm.chatInput.Value(), tm.completions[0])
	}

	// Second tab: select index 1
	tm, _ = updateTree(t, tm, keyTab())
	if tm.completionIdx != 1 {
		t.Errorf("completionIdx after 2nd tab = %d, want 1", tm.completionIdx)
	}

	// Third tab: wraps to index 0
	tm, _ = updateTree(t, tm, keyTab())
	if tm.completionIdx != 0 {
		t.Errorf("completionIdx after 3rd tab = %d, want 0 (wrap)", tm.completionIdx)
	}
}

func TestTabWithNoCompletionsGoesBackToTree(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Switch to chat mode
	tm, _ = updateTree(t, tm, keyTab())
	if tm.mode != tmChat {
		t.Fatalf("mode = %d, want tmChat", tm.mode)
	}

	// Type something without @ prefix
	for _, ch := range "hello" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	if len(tm.completions) != 0 {
		t.Fatalf("completions = %d, want 0", len(tm.completions))
	}

	// Tab should go back to tree
	tm, _ = updateTree(t, tm, keyTab())
	if tm.mode != tmTree {
		t.Errorf("mode = %d, want tmTree (no completions, tab should go back)", tm.mode)
	}
}

func TestCompletionsOverlayRendered(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmChat
	tm.chatInput.Focus()
	tm.chatInput.SetValue("@S-")
	tm.completions = []string{"@STA-0001", "@STA-0002"}
	tm.completionIdx = 0

	output := tm.viewChat()
	if !strings.Contains(output, "@STA-0001") {
		t.Error("chat view should contain @STA-0001 in completions overlay")
	}
	if !strings.Contains(output, "@STA-0002") {
		t.Error("chat view should contain @STA-0002 in completions overlay")
	}
}

func TestCompletionsClearedOnEnter(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Switch to chat mode
	tm, _ = updateTree(t, tm, keyTab())

	// Type "@STA-0001 do something"
	for _, ch := range "@STA-0001 do something" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Submit
	tm, _ = updateTree(t, tm, keyEnter())
	if len(tm.completions) != 0 {
		t.Errorf("completions = %d, want 0 after enter", len(tm.completions))
	}
	if tm.completionIdx != -1 {
		t.Errorf("completionIdx = %d, want -1 after enter", tm.completionIdx)
	}
}

func TestCompletionsClearedOnEsc(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Switch to chat mode
	tm, _ = updateTree(t, tm, keyTab())

	// Type "@STA" to get completions
	for _, ch := range "@STA" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}
	if len(tm.completions) == 0 {
		t.Fatal("expected completions")
	}

	// Esc should clear
	tm, _ = updateTree(t, tm, keyEsc())
	if len(tm.completions) != 0 {
		t.Errorf("completions = %d, want 0 after esc", len(tm.completions))
	}
}

func TestSplitPaneAtBoundary(t *testing.T) {
	// Width exactly 101 should trigger split pane
	tm := newTree(fiveDecisions())
	tm.width = 101
	tm.height = 40
	tm, _ = updateTree(t, tm, keyEnter())

	output := tm.View()
	// Should be split pane (contains both tree items and detail ID)
	if !strings.Contains(output, "@STA-0001") {
		t.Error("width 101 should trigger split pane showing decision ID")
	}

	// Width exactly 100 should NOT trigger split pane
	tm2 := newTree(fiveDecisions())
	tm2.width = 100
	tm2.height = 40
	tm2, _ = updateTree(t, tm2, keyEnter())

	output2 := tm2.View()
	// Full-screen detail has 'shuffle' in footer
	if !strings.Contains(output2, "shuffle") {
		t.Error("width 100 should use full-screen detail with 'shuffle' action")
	}
}

// ========== Action message styling tests ==========

func TestActionEntryRendering(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmChat
	tm.chatInput.Focus()
	tm.chatLog = append(tm.chatLog, ChatEntry{Type: "action", Text: "Changed STA-0001 → Go"})

	output := tm.viewChat()
	if !strings.Contains(output, "Changed STA-0001") {
		t.Error("chat view should display action entry text")
	}
}

func TestActionEntryDistinctFromSystem(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.chatLog = []ChatEntry{
		{Type: "system", Text: "Identifying decisions..."},
		{Type: "action", Text: "Changed STA-0001 → Go"},
	}
	// Both should appear but they render differently (we can at least verify no panic)
	tm.mode = tmChat
	output := tm.viewChat()
	if !strings.Contains(output, "Identifying decisions") {
		t.Error("system entry should appear")
	}
	if !strings.Contains(output, "Changed STA-0001") {
		t.Error("action entry should appear")
	}
}

// ========== Feature hashtag highlight tests ==========

func TestHighlightRefsDecisionAndFeature(t *testing.T) {
	text := "See @STA-0001 and #auth for details"
	result := highlightRefs(text)
	// The result should contain the original tokens (styled)
	if !strings.Contains(result, "@STA-0001") {
		t.Error("highlightRefs should include @STA-0001 content")
	}
	if !strings.Contains(result, "auth") {
		t.Error("highlightRefs should include #auth content")
	}
}

func TestHighlightRefsNoRefs(t *testing.T) {
	text := "No references here"
	result := highlightRefs(text)
	if result != text {
		t.Errorf("highlightRefs with no refs should return unchanged text, got %q", result)
	}
}

// ========== Feature mode toggle tests ==========

func TestFeatureGroupToggle(t *testing.T) {
	tm := newTree(fiveDecisions())

	if tm.groupByFeature {
		t.Fatal("groupByFeature should be false by default")
	}

	// Press 'g' to toggle
	tm, _ = updateTree(t, tm, keyRunes("g"))
	if !tm.groupByFeature {
		t.Error("groupByFeature should be true after pressing g")
	}

	// Press 'g' again to toggle back
	tm, _ = updateTree(t, tm, keyRunes("g"))
	if tm.groupByFeature {
		t.Error("groupByFeature should be false after pressing g again")
	}
}

func TestFeatureGroupViewRenders(t *testing.T) {
	decs := fiveDecisions()
	decs[0].Features = []string{"auth", "backend"}
	decs[1].Features = []string{"backend"}
	decs[2].Features = []string{"data"}

	tm := newTree(decs)
	tm.groupByFeature = true

	output := tm.viewTree()
	// Should show feature group headers
	if !strings.Contains(output, "#auth") {
		t.Error("feature group view should contain #auth header")
	}
	if !strings.Contains(output, "#backend") {
		t.Error("feature group view should contain #backend header")
	}
	if !strings.Contains(output, "(untagged)") {
		t.Error("feature group view should contain (untagged) for decisions without features")
	}
}

func TestFeatureGroupStatusShowsByFeature(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.groupByFeature = true

	output := tm.viewTree()
	if !strings.Contains(output, "by feature") {
		t.Error("status should show 'by feature' when groupByFeature is true")
	}

	tm.groupByFeature = false
	output = tm.viewTree()
	if !strings.Contains(output, "by domain") {
		t.Error("status should show 'by domain' when groupByFeature is false")
	}
}

func TestTreeFooterShowsGroupAndFind(t *testing.T) {
	tm := newTree(fiveDecisions())
	output := tm.viewTree()
	if !strings.Contains(output, "group") {
		t.Error("tree footer should contain 'g group'")
	}
	if !strings.Contains(output, "find") {
		t.Error("tree footer should contain 'ctrl+f find'")
	}
}

// ========== Feature editing tests ==========

func TestFeatureEditMode(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter()) // detail

	// Press 'f' for feature editing
	tm, _ = updateTree(t, tm, keyRunes("f"))
	if tm.mode != tmEditFeatures {
		t.Fatalf("mode = %d, want tmEditFeatures (%d)", tm.mode, tmEditFeatures)
	}
}

func TestFeatureEditModeEscCancels(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, keyEnter())
	tm, _ = updateTree(t, tm, keyRunes("f"))

	// Type something
	for _, ch := range "auth, backend" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Esc should cancel
	tm, _ = updateTree(t, tm, keyEsc())
	if tm.mode != tmDetail {
		t.Errorf("mode = %d, want tmDetail after esc", tm.mode)
	}
	// Features should NOT be changed (original had none)
	sel := tm.selected()
	if sel != nil && len(sel.Features) > 0 {
		t.Error("features should not be changed after cancel")
	}
}

func TestFeatureEditModeSubmit(t *testing.T) {
	decs := fiveDecisions()
	tm := newTree(decs)
	tm, _ = updateTree(t, tm, keyEnter()) // detail for STA-0001
	tm, _ = updateTree(t, tm, keyRunes("f"))

	// Type features
	for _, ch := range "auth, backend, Auth" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// Submit
	var cmd tea.Cmd
	tm, cmd = updateTree(t, tm, keyEnter())

	if tm.mode != tmDetail {
		t.Errorf("mode = %d, want tmDetail after submit", tm.mode)
	}

	// Check that features were deduplicated and lowercased
	for _, d := range tm.decisions {
		if d.ID == "@STA-0001" {
			if len(d.Features) != 2 {
				t.Errorf("features count = %d, want 2 (auth, backend, deduped Auth)", len(d.Features))
			}
			foundAuth := false
			foundBackend := false
			for _, f := range d.Features {
				if f == "auth" {
					foundAuth = true
				}
				if f == "backend" {
					foundBackend = true
				}
			}
			if !foundAuth || !foundBackend {
				t.Errorf("features = %v, want [auth, backend]", d.Features)
			}
			break
		}
	}

	// Should produce SaveFeaturesMsg
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SaveFeaturesMsg); !ok {
			t.Errorf("expected SaveFeaturesMsg, got %T", msg)
		}
	}
}

func TestFeatureEditPreFilled(t *testing.T) {
	decs := fiveDecisions()
	decs[0].Features = []string{"auth", "backend"}

	tm := newTree(decs)
	tm, _ = updateTree(t, tm, keyEnter())
	tm, _ = updateTree(t, tm, keyRunes("f"))

	val := tm.textInput.Value()
	if !strings.Contains(val, "auth") || !strings.Contains(val, "backend") {
		t.Errorf("textInput should be pre-filled with features, got %q", val)
	}
}

func TestDetailFooterShowsFeatures(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.mode = tmDetail
	output := tm.viewDetail()
	if !strings.Contains(output, "features") {
		t.Error("detail footer should contain 'f features'")
	}
}

func TestDetailShowsFeatureTags(t *testing.T) {
	decs := fiveDecisions()
	decs[0].Features = []string{"auth", "backend"}

	tm := newTree(decs)
	tm.mode = tmDetail
	output := tm.viewDetail()
	if !strings.Contains(output, "#auth") {
		t.Error("detail view should show #auth feature tag")
	}
	if !strings.Contains(output, "#backend") {
		t.Error("detail view should show #backend feature tag")
	}
}

// ========== parseFeatureTags tests ==========

func TestParseFeatureTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"basic", "auth, backend", []string{"auth", "backend"}},
		{"with hash", "#auth, #backend", []string{"auth", "backend"}},
		{"dedup", "auth, Auth, AUTH", []string{"auth"}},
		{"trim spaces", "  auth , backend  ,  ui  ", []string{"auth", "backend", "ui"}},
		{"empty", "", nil},
		{"only commas", ", , , ", nil},
		{"mixed", "auth, #backend, Auth, frontend", []string{"auth", "backend", "frontend"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFeatureTags(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseFeatureTags(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseFeatureTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ========== Ctrl+F jump search tests ==========

func TestJumpSearchActivation(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Ctrl+F enters jump search mode
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	if !tm.jumpSearchMode {
		t.Error("jumpSearchMode should be true after ctrl+f")
	}
}

func TestJumpSearchEscCloses(t *testing.T) {
	tm := newTree(fiveDecisions())

	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	if !tm.jumpSearchMode {
		t.Fatal("should be in jump search mode")
	}

	tm, _ = updateTree(t, tm, keyEsc())
	if tm.jumpSearchMode {
		t.Error("jumpSearchMode should be false after esc")
	}
}

func TestJumpSearchMatchesDecisions(t *testing.T) {
	tm := newTree(fiveDecisions())

	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})

	// Type "lang" to match "Language?" question
	for _, ch := range "lang" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	if len(tm.jumpMatches) == 0 {
		t.Fatal("expected jump matches for 'lang'")
	}
	found := false
	for _, jm := range tm.jumpMatches {
		if strings.Contains(jm.Label, "@STA-0001") {
			found = true
			break
		}
	}
	if !found {
		t.Error("jump matches should include STA-0001 (Language?)")
	}
}

func TestJumpSearchMatchesCategories(t *testing.T) {
	tm := newTree(fiveDecisions())

	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	foundCat := false
	for _, jm := range tm.jumpMatches {
		if jm.Type == "category" && jm.Label == "Data" {
			foundCat = true
			break
		}
	}
	if !foundCat {
		t.Error("jump matches should include Data category")
	}
}

func TestJumpSearchMatchesFeatures(t *testing.T) {
	decs := fiveDecisions()
	decs[0].Features = []string{"authentication"}
	tm := newTree(decs)

	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	for _, ch := range "auth" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	foundFeat := false
	for _, jm := range tm.jumpMatches {
		if jm.Type == "feature" && strings.Contains(jm.Label, "authentication") {
			foundFeat = true
			break
		}
	}
	if !foundFeat {
		t.Error("jump matches should include #authentication feature")
	}
}

func TestJumpSearchEnterJumps(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.cursor = 0

	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	// Search for "css" which matches UIX-0001
	for _, ch := range "css" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	if len(tm.jumpMatches) == 0 {
		t.Fatal("expected matches for 'css'")
	}

	// The first match should be the CSS decision
	targetIdx := tm.jumpMatches[0].Index

	// Press enter to jump
	tm, _ = updateTree(t, tm, keyEnter())
	if tm.jumpSearchMode {
		t.Error("jump search should be closed after enter")
	}
	if tm.cursor != targetIdx {
		t.Errorf("cursor = %d, want %d (jumped to match)", tm.cursor, targetIdx)
	}
}

func TestJumpSearchUpDown(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})

	// Search for "sta" to get multiple matches
	for _, ch := range "sta" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	if len(tm.jumpMatches) < 2 {
		t.Fatalf("expected >= 2 matches for 'sta', got %d", len(tm.jumpMatches))
	}

	if tm.jumpCursor != 0 {
		t.Errorf("jumpCursor should start at 0, got %d", tm.jumpCursor)
	}

	// Down
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyDown})
	if tm.jumpCursor != 1 {
		t.Errorf("jumpCursor = %d, want 1 after down", tm.jumpCursor)
	}

	// Up
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyUp})
	if tm.jumpCursor != 0 {
		t.Errorf("jumpCursor = %d, want 0 after up", tm.jumpCursor)
	}
}

func TestJumpSearchDoesNotFilterTree(t *testing.T) {
	tm := newTree(fiveDecisions())

	// Activate jump search and type something
	tm, _ = updateTree(t, tm, tea.KeyMsg{Type: tea.KeyCtrlF})
	for _, ch := range "data" {
		tm, _ = updateTree(t, tm, keyRunes(string(ch)))
	}

	// The tree should still show ALL decisions (not filtered)
	if tm.searchQuery != "" {
		t.Error("jump search should not set searchQuery (that's / filter)")
	}
	// decisionItems should still return all 5
	if tm.decisionCount() != 5 {
		t.Errorf("decisionCount = %d, want 5 (jump search should not filter)", tm.decisionCount())
	}
}

func TestJumpSearchViewRendersOverlay(t *testing.T) {
	decs := fiveDecisions()
	tm := newTree(decs)
	tm.jumpSearchMode = true
	tm.jumpSearchInput.Focus()
	tm.jumpSearchInput.SetValue("sta")
	tm.jumpMatches = []jumpMatch{
		{Type: "decision", Label: "@STA-0001 Language?", Index: 0},
		{Type: "decision", Label: "@STA-0002 Framework?", Index: 1},
	}

	output := tm.viewTree()
	if !strings.Contains(output, "@STA-0001") {
		t.Error("jump search overlay should show STA-0001 match")
	}
	if !strings.Contains(output, "@STA-0002") {
		t.Error("jump search overlay should show STA-0002 match")
	}
}

func TestJumpSearchFooter(t *testing.T) {
	tm := newTree(fiveDecisions())
	tm.jumpSearchMode = true
	tm.jumpSearchInput.Focus()

	output := tm.viewTree()
	if !strings.Contains(output, "jump") {
		t.Error("jump search footer should contain 'jump'")
	}
	if !strings.Contains(output, "close") {
		t.Error("jump search footer should contain 'close'")
	}
}

// ========== computeJumpMatches tests ==========

func TestComputeJumpMatchesEmpty(t *testing.T) {
	tm := newTree(fiveDecisions())
	matches := tm.computeJumpMatches("")
	if len(matches) != 0 {
		t.Errorf("computeJumpMatches('') = %d, want 0", len(matches))
	}
}

func TestComputeJumpMatchesMax8(t *testing.T) {
	// Create many decisions to test the limit
	var decs []decision.Decision
	for i := 0; i < 20; i++ {
		decs = append(decs, decision.Decision{
			ID:       fmt.Sprintf("TST-%04d", i+1),
			Category: "Test",
			Question: fmt.Sprintf("Test question %d?", i+1),
			Source:   "user",
		})
	}
	tm := newTree(decs)
	matches := tm.computeJumpMatches("test")
	if len(matches) > 8 {
		t.Errorf("computeJumpMatches should return max 8 matches, got %d", len(matches))
	}
}
