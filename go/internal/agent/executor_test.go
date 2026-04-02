package agent

import (
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
		// Should overlap: same question rephrased
		{"backend language choice", "backend language choice", true},
		{"which database to use", "database to use for storage", true},
		{"css framework selection method", "css framework selection approach", true},

		// Should NOT overlap: different topics
		{"database choice", "database schema design", false},
		{"backend language", "frontend framework", false},
		{"api authentication method", "api rate limiting strategy", false},

		// Edge cases
		{"", "", false},
		{"a", "b", false},
		{"short", "short", true},
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
		careLevel:       CareLevelMedium,
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
		"stack":    CareLevelHigh,
		"security": CareLevelParanoid,
	})
	e.careLevel = CareLevelMedium

	tests := []struct {
		category string
		want     CareLevel
	}{
		{"Stack", CareLevelHigh},
		{"  STACK  ", CareLevelHigh},
		{"security", CareLevelParanoid},
		{"Unknown", CareLevelMedium}, // falls back to executor default
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
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language?", Answer: strPtr("Go")},
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
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language choice?", Answer: strPtr("Go")},
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
		{ID: "@DAT-0001", Category: "Data", Question: "Database choice?", Answer: strPtr("PostgreSQL")},
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

func TestStoreDecisionCareLevelHigh(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, map[string]CareLevel{
		"stack": CareLevelHigh,
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

	// High care level → answer should be cleared (pending for user)
	if decs[0].Answer != nil {
		t.Errorf("high care level decision should have nil answer, got %q", *decs[0].Answer)
	}
}

func TestStoreDecisionCareLevelLow(t *testing.T) {
	e := makeTestExecutor(nil, []string{"Stack"}, map[string]CareLevel{
		"stack": CareLevelLow,
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

	// Low care level → answer should be preserved
	if decs[0].Answer == nil || *decs[0].Answer != "Go" {
		t.Error("low care level decision should keep its answer")
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
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil, Source: "user"},
		{ID: "@DAT-0001", Category: "Data", Question: "Database?", Answer: strPtr("PostgreSQL"), Source: "agent"},
	}

	e := makeTestExecutor(existing, []string{"Stack", "Data"}, nil)

	// Update existing pending decision
	ok := e.UpdateDecision("@STA-0001", "Go with Gin")
	if !ok {
		t.Fatal("UpdateDecision should return true for existing ID")
	}

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Go with Gin" {
		t.Errorf("@STA-0001 answer = %v, want 'Go with Gin'", decs[0].Answer)
	}
	if decs[0].Source != "agent" {
		t.Errorf("@STA-0001 source = %q, want 'agent'", decs[0].Source)
	}
}

func TestUpdateDecisionOverwrite(t *testing.T) {
	existing := []decision.Decision{
		{ID: "@DAT-0001", Category: "Data", Question: "Database?", Answer: strPtr("PostgreSQL"), Source: "auto"},
	}

	e := makeTestExecutor(existing, []string{"Data"}, nil)

	ok := e.UpdateDecision("@DAT-0001", "SQLite")
	if !ok {
		t.Fatal("UpdateDecision should return true for existing ID")
	}

	decs := *e.allDecisions
	if *decs[0].Answer != "SQLite" {
		t.Errorf("@DAT-0001 answer = %q, want 'SQLite'", *decs[0].Answer)
	}
	if decs[0].Source != "agent" {
		t.Errorf("@DAT-0001 source = %q, want 'agent'", decs[0].Source)
	}
}

func TestUpdateDecisionNotFound(t *testing.T) {
	existing := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Language?"},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	ok := e.UpdateDecision("NONEXISTENT-999", "anything")
	if ok {
		t.Fatal("UpdateDecision should return false for nonexistent ID")
	}
}

func TestScanInlineDecisions(t *testing.T) {
	existing := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
		{ID: "@DAT-0001", Category: "Data", Question: "Database?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack", "Data"}, nil)

	text := `I've decided on the following:
DECISION: STA-0001 = Go with Gin
Some other text here.
DECISION: DAT-0001 = PostgreSQL with pgx`

	e.scanInlineDecisions(text)

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Go with Gin" {
		t.Errorf("@STA-0001 answer = %v, want 'Go with Gin'", decs[0].Answer)
	}
	if decs[1].Answer == nil || *decs[1].Answer != "PostgreSQL with pgx" {
		t.Errorf("@DAT-0001 answer = %v, want 'PostgreSQL with pgx'", decs[1].Answer)
	}
}

func TestScanInlineDecisionsNoMatch(t *testing.T) {
	existing := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Text with no DECISION patterns
	e.scanInlineDecisions("just some regular text without any decision markers")

	decs := *e.allDecisions
	if decs[0].Answer != nil {
		t.Errorf("@STA-0001 should still be pending, got %v", *decs[0].Answer)
	}
}

func TestScanInlineDecisionsPartialMatch(t *testing.T) {
	existing := []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Backend language?", Answer: nil},
	}

	e := makeTestExecutor(existing, []string{"Stack"}, nil)

	// Only one decision matches an existing ID
	text := `DECISION: STA-0001 = Rust
DECISION: MISSING-999 = something`

	e.scanInlineDecisions(text)

	decs := *e.allDecisions
	if decs[0].Answer == nil || *decs[0].Answer != "Rust" {
		t.Errorf("@STA-0001 answer = %v, want 'Rust'", decs[0].Answer)
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
