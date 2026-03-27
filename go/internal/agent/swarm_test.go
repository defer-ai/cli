package agent

import (
	"strings"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func TestSwarmParseDecisionsCleanJSON(t *testing.T) {
	text := `[
		{"category": "API", "decision": "REST vs GraphQL?", "options": ["REST", "GraphQL", "gRPC"], "reasoning": "Determines client integration"},
		{"category": "API", "decision": "Auth mechanism?", "options": ["JWT", "Session", "API Key"], "reasoning": "Security model"}
	]`
	decs := ParseSwarmDecisions(text, "Fallback", nil)
	if len(decs) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decs))
	}
	if decs[0].Question != "REST vs GraphQL?" {
		t.Errorf("first decision question = %q", decs[0].Question)
	}
	// Category is always forced to defaultCategory, not what model returned
	if decs[0].Category != "Fallback" {
		t.Errorf("first decision category = %q, want Fallback (forced)", decs[0].Category)
	}
	if len(decs[0].Options) != 3 {
		t.Errorf("first decision options count = %d, want 3", len(decs[0].Options))
	}
	if decs[0].Source != "agent" {
		t.Errorf("source = %q, want agent", decs[0].Source)
	}
}

func TestSwarmParseDecisionsMarkdownWrapped(t *testing.T) {
	text := "Here are the decisions:\n```json\n" +
		`[{"category": "Data", "decision": "ORM choice?", "options": ["GORM", "sqlx"], "reasoning": "DB access"}]` +
		"\n```\n"
	decs := ParseSwarmDecisions(text, "Data", nil)
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Question != "ORM choice?" {
		t.Errorf("decision question = %q", decs[0].Question)
	}
}

func TestSwarmParseDecisionsPreamble(t *testing.T) {
	text := "I'll decompose the API domain into sub-decisions:\n\n" +
		`[{"category": "API", "decision": "Rate limiting strategy?", "options": ["Token bucket", "Fixed window"], "reasoning": "Prevents abuse"}]`
	decs := ParseSwarmDecisions(text, "API", nil)
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
}

func TestSwarmParseDecisionsMalformed(t *testing.T) {
	text := "This is not JSON at all, just random text."
	decs := ParseSwarmDecisions(text, "X", nil)
	if decs != nil {
		t.Errorf("expected nil for malformed input, got %d decisions", len(decs))
	}
}

func TestSwarmParseDecisionsMalformedJSON(t *testing.T) {
	text := `[{"category": "API", "decision": "broken`
	decs := ParseSwarmDecisions(text, "X", nil)
	if decs != nil {
		t.Errorf("expected nil for broken JSON, got %v", decs)
	}
}

func TestSwarmParseDecisionsEmptyDecisionField(t *testing.T) {
	text := `[{"category": "API", "decision": "", "options": ["A"], "reasoning": "test"}]`
	decs := ParseSwarmDecisions(text, "API", nil)
	if len(decs) != 0 {
		t.Errorf("expected 0 decisions for empty decision field, got %d", len(decs))
	}
}

func TestSwarmParseDecisionsFallbackCategory(t *testing.T) {
	text := `[{"category": "", "decision": "Which framework?", "options": ["A"], "reasoning": "test"}]`
	decs := ParseSwarmDecisions(text, "MyDomain", nil)
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if decs[0].Category != "MyDomain" {
		t.Errorf("category = %q, want MyDomain (fallback)", decs[0].Category)
	}
}

func TestSwarmGroupByCategory(t *testing.T) {
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

func TestSwarmDecisionIDs(t *testing.T) {
	text := `[
		{"category": "API", "decision": "Auth?", "options": ["JWT"], "reasoning": "r1"},
		{"category": "API", "decision": "Rate limit?", "options": ["Token bucket"], "reasoning": "r2"},
		{"category": "Data", "decision": "DB?", "options": ["Postgres"], "reasoning": "r3"}
	]`
	decs := ParseSwarmDecisions(text, "Fallback", nil)
	if len(decs) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decs))
	}

	// All IDs must be unique
	seen := make(map[string]bool)
	for _, d := range decs {
		if seen[d.ID] {
			t.Errorf("duplicate ID: %s", d.ID)
		}
		seen[d.ID] = true

		// IDs should contain category prefix
		if d.Category == "API" && !strings.HasPrefix(d.ID, "API-") {
			t.Errorf("API decision ID %q should start with API-", d.ID)
		}
		if d.Category == "Data" && !strings.HasPrefix(d.ID, "DATA-") {
			t.Errorf("Data decision ID %q should start with DATA-", d.ID)
		}
	}
}

func TestSwarmMergeWithExisting(t *testing.T) {
	existing := []decision.Decision{
		{ID: "API-001", Category: "API", Question: "Auth mechanism?"},
		{ID: "API-002", Category: "API", Question: "Rate limiting strategy?"},
	}

	text := `[
		{"category": "API", "decision": "Auth mechanism?", "options": ["JWT"], "reasoning": "dup"},
		{"category": "API", "decision": "Pagination approach?", "options": ["Cursor", "Offset"], "reasoning": "new"}
	]`

	decs := ParseSwarmDecisions(text, "API", existing)

	// Should only get the new one, "Auth mechanism?" is a dup
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision (deduped), got %d", len(decs))
	}
	if decs[0].Question != "Pagination approach?" {
		t.Errorf("expected Pagination approach?, got %q", decs[0].Question)
	}
}

func TestSwarmMergeWithExistingCaseInsensitive(t *testing.T) {
	existing := []decision.Decision{
		{ID: "API-001", Category: "API", Question: "Auth Mechanism?"},
	}

	text := `[
		{"category": "API", "decision": "auth mechanism?", "options": ["JWT"], "reasoning": "dup"}
	]`

	decs := ParseSwarmDecisions(text, "API", existing)
	if len(decs) != 0 {
		t.Errorf("expected 0 decisions (case-insensitive dedup), got %d", len(decs))
	}
}

func TestSwarmParseDecisionsImplicitFlag(t *testing.T) {
	text := `[{"category": "API", "decision": "Caching?", "options": ["Redis"], "reasoning": "perf"}]`
	decs := ParseSwarmDecisions(text, "API", nil)
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	if !decs[0].Implicit {
		t.Error("swarm decisions should be implicit")
	}
}
