package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func TestNormalizeQuestion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Backend language?", "backend language?"},
		{"  Backend language?  ", "backend language?"},
		{"Database (LAY-0037 — explicitly pending)", "database"},
		{"Storage (pending)", "storage"},
		{"ORM choice (DAT-0003 - explicit)?", "orm choice?"},
		{"No parens here", "no parens here"},
		{"Multiple (ref-1) parens (ref-2)", "multiple parens"},
		{"Keep (normal parenthetical)", "keep (normal parenthetical)"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeQuestion(tt.input)
			if got != tt.want {
				t.Errorf("normalizeQuestion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuestionsOverlap(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Should overlap: identical or near-identical
		{"backend language choice", "backend language choice", true},
		{"which database to use", "database to use for storage", true},
		{"short", "short", true},

		// Should NOT overlap: different topics
		{"database choice", "database schema design", false},
		{"backend language", "frontend framework", false},
		{"api authentication method", "api rate limiting strategy", false},
		// With 0.85 threshold, minor rephrasing no longer triggers overlap
		{"css framework selection method", "css framework selection approach", false},

		// Edge cases
		{"", "", false},
		{"a", "b", false},
	}
	for _, tt := range tests {
		name := tt.a + " vs " + tt.b
		t.Run(name, func(t *testing.T) {
			got := questionsOverlap(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("questionsOverlap(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSignificantWords(t *testing.T) {
	stop := map[string]bool{
		"a": true, "the": true, "to": true, "for": true,
		"what": true, "which": true, "should": true, "we": true, "use": true,
	}

	tests := []struct {
		input string
		want  int // count of significant words
	}{
		{"what database should we use for the backend", 2}, // database, backend
		{"", 0},
		{"a the to", 0}, // all stop words
		{"backend language choice", 3},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := significantWords(tt.input, stop)
			if len(got) != tt.want {
				t.Errorf("significantWords(%q) = %v (len %d), want len %d", tt.input, got, len(got), tt.want)
			}
		})
	}
}

func makeTestExecutor(decs []decision.Decision, categories []string, priorities map[string]CareLevel) *Executor {
	allDecs := make([]decision.Decision, len(decs))
	copy(allDecs, decs)

	priMap := make(map[string]CareLevel)
	for k, v := range priorities {
		priMap[k] = v
	}

	return &Executor{
		allDecisions:    &allDecs,
		knownCategories: categories,
		priorities:      priMap,
		careLevel:       CareLevelAuto,
		onEvent:         func(Event) {},
	}
}

func TestNormalizeCategoryExactMatch(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack", "Security", "UI"}, nil)

	tests := []struct {
		input string
		want  string
	}{
		{"Stack", "Stack"},
		{"stack", "Stack"},
		{"STACK", "Stack"},
		{"  Stack  ", "Stack"},
		{"Security", "Security"},
		{"UI", "UI"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := e.normalizeCategoryLocked(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCategoryLocked(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCategorySubstringMatch(t *testing.T) {
	e := makeTestExecutor(nil, []string{"CLI", "Storage", "API"}, nil)

	tests := []struct {
		input string
		want  string
	}{
		{"CLI Interface", "CLI"},
		{"Data Storage", "Storage"},
		{"REST API", "API"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := e.normalizeCategoryLocked(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCategoryLocked(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCategoryWordOverlap(t *testing.T) {
	e := makeTestExecutor(nil, []string{"User Interface", "Data Layer"}, nil)

	got := e.normalizeCategoryLocked("Interface Design")
	if got != "User Interface" {
		t.Errorf("normalizeCategoryLocked(\"Interface Design\") = %q, want \"User Interface\"", got)
	}
}

func TestNormalizeCategoryFallback(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack", "Misc"}, nil)

	got := e.normalizeCategoryLocked("Totally Unknown")
	if got != "Misc" {
		t.Errorf("normalizeCategoryLocked(\"Totally Unknown\") = %q, want \"Misc\"", got)
	}
}

func TestNormalizeCategoryNoMiscFallback(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack", "Security"}, nil)

	got := e.normalizeCategoryLocked("Totally Unknown")
	if got != "Stack" {
		t.Errorf("normalizeCategoryLocked(\"Totally Unknown\") = %q, want \"Stack\" (first known)", got)
	}
}

func TestNormalizeCategoryEmpty(t *testing.T) {
	e := makeTestExecutor(nil, nil, nil)

	got := e.normalizeCategoryLocked("Anything")
	if got != "Misc" {
		t.Errorf("normalizeCategoryLocked with no known categories = %q, want \"Misc\"", got)
	}
}

func TestGetCareLevel(t *testing.T) {
	e := makeTestExecutor(nil, nil, map[string]CareLevel{
		"stack":    CareLevelReview,
		"security": CareLevelReview,
	})
	e.careLevel = CareLevelAuto

	tests := []struct {
		category string
		want     CareLevel
	}{
		{"Stack", CareLevelReview},
		{"  STACK  ", CareLevelReview},
		{"security", CareLevelReview},
		{"Unknown", CareLevelAuto}, // falls back to executor default
	}
	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := e.getCareLevel(tt.category)
			if got != tt.want {
				t.Errorf("getCareLevel(%q) = %q, want %q", tt.category, got, tt.want)
			}
		})
	}
}

func TestStoreDecisionDedup(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language?", Answer: strPtr("Go")},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Try to store a duplicate
	e.storeDecision(decision.Decision{
		Category: "Stack",
		Question: "Backend language?",
		Answer:   strPtr("Rust"),
	})

	if len(*e.allDecisions) != 1 {
		t.Errorf("expected 1 decision (dedup), got %d", len(*e.allDecisions))
	}
}

func TestStoreDecisionOverlapDedup(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language choice?", Answer: strPtr("Go")},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Similar but rephrased — should be caught by word overlap
	e.storeDecision(decision.Decision{
		Category: "Stack",
		Question: "Backend language choice selection?",
		Answer:   strPtr("Rust"),
	})

	if len(*e.allDecisions) != 1 {
		t.Errorf("expected 1 decision (overlap dedup), got %d", len(*e.allDecisions))
	}
}

func TestStoreDecisionDifferentNotDeduped(t *testing.T) {
	existing := []decision.Decision{
		{ID: "DAT-0001", Category: "Data", Question: "Database choice?", Answer: strPtr("PostgreSQL")},
	}

	e := makeTestExecutor(existing, []string{"Data"}, nil)

	// Different topic — should NOT be deduped
	e.storeDecision(decision.Decision{
		Category: "Data",
		Question: "Database schema design?",
		Answer:   strPtr("Star schema"),
	})

	if len(*e.allDecisions) != 2 {
		t.Errorf("expected 2 decisions (different topics), got %d", len(*e.allDecisions))
	}
}

func TestStoreDecisionCareLevelReview(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, map[string]CareLevel{
		"stack": CareLevelReview,
	})

	answer := "Go"
	e.storeDecision(decision.Decision{
		Category: "Stack",
		Question: "Backend language?",
		Answer:   &answer,
		Source:   "agent",
	})

	decs := *e.allDecisions
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}

	// Review care level -> answer should be cleared (pending for user)
	if decs[0].Answer != nil {
		t.Errorf("review care level decision should have nil answer, got %q", *decs[0].Answer)
	}
}

func TestStoreDecisionCareLevelAuto(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, map[string]CareLevel{
		"stack": CareLevelAuto,
	})

	answer := "Go"
	e.storeDecision(decision.Decision{
		Category: "Stack",
		Question: "Backend language?",
		Answer:   &answer,
		Source:   "agent",
	})

	decs := *e.allDecisions
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}

	// Auto care level -> answer should be preserved
	if decs[0].Answer == nil || *decs[0].Answer != "Go" {
		t.Error("auto care level decision should keep its answer")
	}
}

func TestParseImplicitChoices(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack", "Data"}, nil)

	input := `Some text before.
[
  {
    "category": "Stack",
    "question": "ORM choice?",
    "options": [{"key": "A", "label": "GORM"}, {"key": "B", "label": "sqlc"}],
    "answer": "B",
    "reasoning": "Type-safe SQL",
    "impact": 6
  },
  {
    "category": "Data",
    "question": "Migration tool?",
    "options": [{"key": "A", "label": "goose"}, {"key": "B", "label": "golang-migrate"}],
    "answer": "A",
    "reasoning": "Simple and effective",
    "impact": 3
  }
]
Some text after.`

	decs := e.parseImplicitChoices(input)
	if len(decs) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decs))
	}

	// First decision
	if decs[0].Question != "ORM choice?" {
		t.Errorf("decs[0].Question = %q", decs[0].Question)
	}
	if decs[0].Answer == nil || *decs[0].Answer != "sqlc" {
		t.Errorf("decs[0].Answer should be 'sqlc' (resolved from key B)")
	}
	if decs[0].Impact != 6 {
		t.Errorf("decs[0].Impact = %d, want 6", decs[0].Impact)
	}
	if len(decs[0].Options) != 2 {
		t.Errorf("decs[0].Options = %d, want 2", len(decs[0].Options))
	}

	// Second decision
	if decs[1].Question != "Migration tool?" {
		t.Errorf("decs[1].Question = %q", decs[1].Question)
	}
	if decs[1].Answer == nil || *decs[1].Answer != "goose" {
		t.Errorf("decs[1].Answer should be 'goose' (resolved from key A)")
	}
}

func TestParseImplicitChoicesDirectLabel(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, nil)

	input := `[{"category": "Stack", "question": "Runtime?", "options": [], "answer": "Node.js", "reasoning": "Fast", "impact": 5}]`

	decs := e.parseImplicitChoices(input)
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Answer == nil || *decs[0].Answer != "Node.js" {
		t.Errorf("answer should be 'Node.js' (direct label), got %v", decs[0].Answer)
	}
}

func TestParseImplicitChoicesInvalidJSON(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, nil)

	decs := e.parseImplicitChoices("This is not JSON at all.")
	if len(decs) != 0 {
		t.Errorf("expected 0 decisions from invalid input, got %d", len(decs))
	}
}

func TestParseImplicitChoicesEmptyQuestion(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, nil)

	input := `[{"category": "Stack", "question": "", "decision": "", "answer": "Go", "impact": 5}]`

	decs := e.parseImplicitChoices(input)
	if len(decs) != 0 {
		t.Errorf("expected 0 decisions (empty question), got %d", len(decs))
	}
}

func TestUpdateDecision(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil, Source: "user"},
		{ID: "DAT-0001", Category: "Data", Question: "Database?", Answer: strPtr("PostgreSQL"), Source: "agent"},
	}

	e := makeTestExecutor(existing, []string{"Stack", "Data"}, nil)

	// Update existing pending decision
	ok := e.UpdateDecision("STA-0001", "Go with Gin")
	if !ok {
		t.Fatal("UpdateDecision should return true for existing ID")
	}

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Go with Gin" {
		t.Errorf("STA-0001 answer = %v, want 'Go with Gin'", decs[0].Answer)
	}
	if decs[0].Source != "agent" {
		t.Errorf("STA-0001 source = %q, want 'agent'", decs[0].Source)
	}
}

func TestUpdateDecisionOverwrite(t *testing.T) {
	existing := []decision.Decision{
		{ID: "DAT-0001", Category: "Data", Question: "Database?", Answer: strPtr("PostgreSQL"), Source: "auto"},
	}

	e := makeTestExecutor(existing, []string{"Data"}, nil)

	ok := e.UpdateDecision("DAT-0001", "SQLite")
	if !ok {
		t.Fatal("UpdateDecision should return true for existing ID")
	}

	decs := *e.allDecisions
	if *decs[0].Answer != "SQLite" {
		t.Errorf("DAT-0001 answer = %q, want 'SQLite'", *decs[0].Answer)
	}
	if decs[0].Source != "agent" {
		t.Errorf("DAT-0001 source = %q, want 'agent'", decs[0].Source)
	}
}

func TestUpdateDecisionNotFound(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?"},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	ok := e.UpdateDecision("NONEXISTENT-999", "anything")
	if ok {
		t.Fatal("UpdateDecision should return false for nonexistent ID")
	}
}

func TestScanInlineDecisions(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
		{ID: "DAT-0001", Category: "Data", Question: "Database?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack", "Data"}, nil)

	text := `I've decided on the following:
DECISION: STA-0001 = Go with Gin
Some other text here.
DECISION: DAT-0001 = PostgreSQL with pgx`

	e.scanInlineDecisions(text)

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Go with Gin" {
		t.Errorf("STA-0001 answer = %v, want 'Go with Gin'", decs[0].Answer)
	}
	if decs[1].Answer == nil || *decs[1].Answer != "PostgreSQL with pgx" {
		t.Errorf("DAT-0001 answer = %v, want 'PostgreSQL with pgx'", decs[1].Answer)
	}
}

func TestScanInlineDecisionsNoMatch(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Text with no DECISION patterns
	e.scanInlineDecisions("just some regular text without any decision markers")

	decs := *e.allDecisions
	if decs[0].Answer != nil {
		t.Errorf("STA-0001 should still be pending, got %v", *decs[0].Answer)
	}
}

func TestScanInlineDecisionsPartialMatch(t *testing.T) {
	existing := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Only one decision matches an existing ID
	text := `DECISION: STA-0001 = Rust
DECISION: MISSING-999 = something`

	e.scanInlineDecisions(text)

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Rust" {
		t.Errorf("STA-0001 answer = %v, want 'Rust'", decs[0].Answer)
	}
}

func TestInlineDecisionRegex(t *testing.T) {
	tests := []struct {
		input  string
		wantID string
		wantAn string
	}{
		{"DECISION: STA-0001 = Go with Gin", "STA-0001", "Go with Gin"},
		{"DECISION:  DAT-0042  =  PostgreSQL ", "DAT-0042", "PostgreSQL"},
		{"DECISION: UIX-0001 = Tailwind CSS v4", "UIX-0001", "Tailwind CSS v4"},
	}

	for _, tt := range tests {
		m := inlineDecisionRe.FindStringSubmatch(tt.input)
		if m == nil {
			t.Errorf("no match for %q", tt.input)
			continue
		}
		if len(m) != 3 {
			t.Errorf("expected 3 groups for %q, got %d", tt.input, len(m))
			continue
		}
		gotID := strings.TrimSpace(m[1])
		gotAn := strings.TrimSpace(m[2])
		if gotID != tt.wantID {
			t.Errorf("input %q: ID = %q, want %q", tt.input, gotID, tt.wantID)
		}
		if gotAn != tt.wantAn {
			t.Errorf("input %q: answer = %q, want %q", tt.input, gotAn, tt.wantAn)
		}
	}
}

func TestInlineDecisionRegexNoMatch(t *testing.T) {
	noMatch := []string{
		"regular text",
		"DECISION: no-match = something",     // lowercase ID
		"DECISION: 123 = something",           // no prefix
		"decision: STA-0001 = something",     // lowercase DECISION
	}

	for _, input := range noMatch {
		m := inlineDecisionRe.FindStringSubmatch(input)
		if m != nil {
			t.Errorf("should not match %q, but got %v", input, m)
		}
	}
}

func strPtr(s string) *string { return &s }

func TestProcessDecisionLineDecided(t *testing.T) {
	e := makeTestExecutor(nil, nil, nil)

	e.processDecisionLine("DECIDED: Stack | Backend language? | Go | Rust, Python | Standard choice for CLI tools")

	decs := *e.allDecisions
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Question != "Backend language" {
		t.Errorf("question = %q, want 'Backend language'", decs[0].Question)
	}
	if decs[0].Answer == nil || *decs[0].Answer != "Go" {
		t.Errorf("answer = %v, want 'Go'", decs[0].Answer)
	}
	if decs[0].Reasoning != "Standard choice for CLI tools" {
		t.Errorf("reasoning = %q", decs[0].Reasoning)
	}
	if decs[0].OriginalSource != "agent" {
		t.Errorf("originalSource = %q, want 'agent'", decs[0].OriginalSource)
	}
}

func TestProcessDecisionLinePending(t *testing.T) {
	e := makeTestExecutor(nil, nil, nil)

	e.processDecisionLine("PENDING: Auth | Session storage? | A) Redis, B) PostgreSQL, C) In-memory | Affects scaling")

	decs := *e.allDecisions
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Question != "Session storage" {
		t.Errorf("question = %q", decs[0].Question)
	}
	if decs[0].Answer != nil {
		t.Errorf("answer should be nil (pending), got %v", *decs[0].Answer)
	}
	if len(decs[0].Options) < 2 {
		t.Errorf("expected >= 2 options, got %d", len(decs[0].Options))
	}
}

func TestProcessDecisionLineIgnoresPartial(t *testing.T) {
	e := makeTestExecutor(nil, nil, nil)

	// Partial line (no closing fields) should not match
	e.processDecisionLine("DECIDED: Stack | Backend lan")

	decs := *e.allDecisions
	if len(decs) != 0 {
		t.Errorf("expected 0 decisions from partial line, got %d", len(decs))
	}
}

func TestProcessDecisionLineMultiple(t *testing.T) {
	e := makeTestExecutor(nil, nil, nil)

	lines := []string{
		"DECIDED: Stack | Language? | Go | Rust, Python | Team expertise",
		"DECIDED: Data | Database? | PostgreSQL | MySQL, SQLite | Production-grade",
		"PENDING: Auth | Auth method? | A) JWT, B) Session | Security critical",
	}
	for _, line := range lines {
		e.processDecisionLine(line)
	}

	decs := *e.allDecisions
	if len(decs) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decs))
	}
	// First two should be answered
	if decs[0].Answer == nil {
		t.Error("decision 0 should be answered")
	}
	if decs[1].Answer == nil {
		t.Error("decision 1 should be answered")
	}
	// Third should be pending
	if decs[2].Answer != nil {
		t.Error("decision 2 should be pending")
	}
}

func TestStreamingLineSplitSimulation(t *testing.T) {
	// Simulate what happens when a DECIDED line arrives split across two EventTextDelta chunks.
	// The line buffer should accumulate until newline and then parse the complete line.
	e := makeTestExecutor(nil, nil, nil)

	var lineBuf string

	// Chunk 1: partial line
	chunk1 := "Some output\nDECIDED: Stack | Language?"
	lineBuf += chunk1
	for {
		idx := strings.Index(lineBuf, "\n")
		if idx == -1 {
			break
		}
		line := lineBuf[:idx]
		lineBuf = lineBuf[idx+1:]
		e.processDecisionLine(line)
	}
	// Should not have parsed the DECIDED line yet (no newline after it)
	if len(*e.allDecisions) != 0 {
		t.Fatalf("should have 0 decisions after chunk1, got %d", len(*e.allDecisions))
	}

	// Chunk 2: rest of the line + newline
	chunk2 := " | Go | Rust, Python | Standard\nMore output\n"
	lineBuf += chunk2
	for {
		idx := strings.Index(lineBuf, "\n")
		if idx == -1 {
			break
		}
		line := lineBuf[:idx]
		lineBuf = lineBuf[idx+1:]
		e.processDecisionLine(line)
	}
	// Now the DECIDED line should be parsed
	if len(*e.allDecisions) != 1 {
		t.Fatalf("should have 1 decision after chunk2, got %d", len(*e.allDecisions))
	}
	if *(*e.allDecisions)[0].Answer != "Go" {
		t.Errorf("answer = %q, want Go", *(*e.allDecisions)[0].Answer)
	}
}

// TestSyncDecisionsFromDiskPicksUpMCPWrites is the regression guard for
// the bug where MCP subprocess writes to .defer/decisions.json never
// made it to the TUI. The MCP server writes directly to disk from its
// subprocess, so the in-memory allDecisions slice goes stale unless
// the executor explicitly reloads after each tool call. This test
// simulates that scenario: pre-seed a store on disk with a new
// decision the executor doesn't know about, call
// syncDecisionsFromDisk, and verify it's been appended to allDecisions
// and that an ExecDecisionStored event was emitted.
func TestSyncDecisionsFromDiskPicksUpMCPWrites(t *testing.T) {
	tmpDir := t.TempDir()
	// Write a store containing a decision the executor has never seen.
	answer := "bin/server"
	store := &decision.DecisionStore{
		Task: "test",
		Decisions: []decision.Decision{
			{ID: "BUI-0001", Category: "Build", Question: "binary location",
				Answer: &answer, Source: "agent"},
		},
	}
	if err := decision.SaveStore(tmpDir, store); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}

	// Build an executor pointing at the tmpDir with an empty in-memory
	// decisions slice — simulating "MCP just wrote, but the executor
	// hasn't synced yet".
	allDecs := []decision.Decision{}
	var captured []Event
	e := &Executor{
		cwd:          tmpDir,
		allDecisions: &allDecs,
		onEvent:      func(ev Event) { captured = append(captured, ev) },
		state:        ExecState{ID: "test-exec", Domain: "Build"},
	}

	e.syncDecisionsFromDisk()

	// The new decision should now be in the in-memory slice.
	if len(*e.allDecisions) != 1 {
		t.Fatalf("after sync: expected 1 decision in allDecisions, got %d", len(*e.allDecisions))
	}
	if (*e.allDecisions)[0].ID != "BUI-0001" {
		t.Errorf("synced decision ID = %q, want BUI-0001", (*e.allDecisions)[0].ID)
	}

	// And an ExecDecisionStored event should have fired so the TUI
	// picks it up.
	var storedEvents int
	for _, ev := range captured {
		if ev.Type == ExecDecisionStored {
			storedEvents++
		}
	}
	if storedEvents != 1 {
		t.Errorf("ExecDecisionStored events = %d, want 1", storedEvents)
	}
}

// TestSyncDecisionsFromDiskSkipsKnownIDs — if a decision is already in
// the in-memory slice, syncing should leave it alone and NOT emit a
// duplicate ExecDecisionStored event. This prevents event storms when
// sync is called on every tool call but no new decisions exist.
func TestSyncDecisionsFromDiskSkipsKnownIDs(t *testing.T) {
	tmpDir := t.TempDir()
	answer := "Go"
	store := &decision.DecisionStore{
		Task: "test",
		Decisions: []decision.Decision{
			{ID: "STA-0001", Category: "Stack", Question: "lang", Answer: &answer, Source: "agent"},
		},
	}
	if err := decision.SaveStore(tmpDir, store); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}

	// Pre-populate the in-memory slice with the same decision (by ID).
	allDecs := []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "lang", Answer: &answer, Source: "user"},
	}
	var captured []Event
	e := &Executor{
		cwd:          tmpDir,
		allDecisions: &allDecs,
		onEvent:      func(ev Event) { captured = append(captured, ev) },
		state:        ExecState{ID: "test-exec", Domain: "Stack"},
	}

	e.syncDecisionsFromDisk()

	// Slice should still have exactly 1 entry — the existing one, not
	// duplicated from disk.
	if len(*e.allDecisions) != 1 {
		t.Errorf("allDecisions should still have 1 entry, got %d", len(*e.allDecisions))
	}
	// No events should have fired.
	for _, ev := range captured {
		if ev.Type == ExecDecisionStored {
			t.Error("ExecDecisionStored should not fire when the id already exists in memory")
		}
	}
}

// TestSyncDecisionsFromDiskHandlesMissingStore — if .defer/decisions.json
// doesn't exist, sync should be a no-op instead of panicking. This is
// the startup case where the executor tool call fires before anything
// has been written.
func TestSyncDecisionsFromDiskHandlesMissingStore(t *testing.T) {
	tmpDir := t.TempDir()
	// No .defer/decisions.json written.
	allDecs := []decision.Decision{}
	var captured []Event
	e := &Executor{
		cwd:          tmpDir,
		allDecisions: &allDecs,
		onEvent:      func(ev Event) { captured = append(captured, ev) },
		state:        ExecState{ID: "test-exec", Domain: "Test"},
	}

	// Should not panic, should not produce events, should not modify the
	// slice.
	e.syncDecisionsFromDisk()

	if len(*e.allDecisions) != 0 {
		t.Errorf("allDecisions should still be empty, got %d entries", len(*e.allDecisions))
	}
	if len(captured) != 0 {
		t.Errorf("no events should fire, got %d", len(captured))
	}
	// Silence unused import warning if filepath isn't referenced elsewhere.
	_ = filepath.Join
}
