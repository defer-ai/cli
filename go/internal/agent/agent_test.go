package agent

import (
	"strings"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func TestParseDecisions(t *testing.T) {
	input := "Here are the decisions:\n\n```defer-decisions\n" +
		`[
  {
    "category": "Stack",
    "question": "Backend language?",
    "options": [
      {"key": "A", "label": "TypeScript"},
      {"key": "B", "label": "Python"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Determines the backend"
  },
  {
    "category": "Data",
    "question": "Database?",
    "options": [
      {"key": "A", "label": "PostgreSQL"},
      {"key": "B", "label": "SQLite"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Persistence layer"
  }
]` + "\n```\n\nLet me know."

	decs := parseDecisions(input, nil)
	if len(decs) != 2 {
		t.Fatalf("got %d decisions, want 2", len(decs))
	}

	if decs[0].Category != "Stack" {
		t.Errorf("decs[0].Category = %q, want Stack", decs[0].Category)
	}
	if decs[0].Question != "Backend language?" {
		t.Errorf("decs[0].Question = %q", decs[0].Question)
	}
	if len(decs[0].Options) != 3 {
		t.Errorf("decs[0].Options = %d, want 3", len(decs[0].Options))
	}
	if !strings.HasPrefix(decs[0].ID, "STACK-") {
		t.Errorf("decs[0].ID = %q, want STACK- prefix", decs[0].ID)
	}
	if decs[1].Category != "Data" {
		t.Errorf("decs[1].Category = %q, want Data", decs[1].Category)
	}
	if !strings.HasPrefix(decs[1].ID, "DATA-") {
		t.Errorf("decs[1].ID = %q, want DATA- prefix", decs[1].ID)
	}
	if decs[1].Source != "user" {
		t.Errorf("decs[1].Source = %q, want user", decs[1].Source)
	}

	// IDs must be unique
	if decs[0].ID == decs[1].ID {
		t.Errorf("duplicate IDs: %s", decs[0].ID)
	}
}

func TestParseDecisionsEmpty(t *testing.T) {
	decs := parseDecisions("Hello! How can I help?", nil)
	if len(decs) != 0 {
		t.Fatalf("got %d decisions, want 0", len(decs))
	}
}

func TestAutoDecide(t *testing.T) {
	a := NewAgent("test", nil, ".")
	a.state.Decisions = []decision.Decision{
		{ID: "STACK-001", Category: "Stack", Question: "Language?", Options: []decision.DecisionOption{
			{Key: "A", Label: "TypeScript"},
			{Key: "B", Label: "Choose for me"},
		}},
		{ID: "DATA-001", Category: "Data", Question: "Database?", Options: []decision.DecisionOption{
			{Key: "A", Label: "PostgreSQL"},
			{Key: "B", Label: "SQLite"},
			{Key: "C", Label: "Choose for me"},
		}},
	}

	a.AutoDecide([]string{"STACK-001"})

	if a.state.Decisions[0].Answer == nil {
		t.Fatal("STACK-001 should be answered")
	}
	if *a.state.Decisions[0].Answer != "TypeScript" {
		t.Errorf("STACK-001 answer = %q, want TypeScript", *a.state.Decisions[0].Answer)
	}
	if !a.state.Decisions[0].Delegated {
		t.Error("STACK-001 should be delegated")
	}
	if a.state.Decisions[0].Source != "auto" {
		t.Errorf("source = %q, want auto", a.state.Decisions[0].Source)
	}

	if a.state.Decisions[1].Answer != nil {
		t.Error("DATA-001 should still be pending")
	}

	a.AutoDecide(nil)
	if a.state.Decisions[1].Answer == nil {
		t.Fatal("DATA-001 should now be answered")
	}
	if *a.state.Decisions[1].Answer != "PostgreSQL" {
		t.Errorf("DATA-001 answer = %q, want PostgreSQL", *a.state.Decisions[1].Answer)
	}
}
