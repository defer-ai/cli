package decision

import (
	"strings"
	"testing"
)

func TestDiffStoresAdditions(t *testing.T) {
	a := Ptr("Go")
	newStore := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: a},
		},
	}
	entries := DiffStores(nil, newStore)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Type != "added" {
		t.Errorf("expected added, got %s", entries[0].Type)
	}
}

func TestDiffStoresChanges(t *testing.T) {
	old := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: Ptr("Go")},
		},
	}
	new := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: Ptr("Rust")},
		},
	}
	entries := DiffStores(old, new)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Type != "changed" {
		t.Errorf("expected changed, got %s", entries[0].Type)
	}
	if entries[0].OldValue != "Go" || entries[0].NewValue != "Rust" {
		t.Errorf("expected Go->Rust, got %s->%s", entries[0].OldValue, entries[0].NewValue)
	}
}

func TestDiffStoresRemovals(t *testing.T) {
	old := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: Ptr("Go")},
			{ID: "DAT-0001", Category: "Data", Question: "Database?", Answer: Ptr("Postgres")},
		},
	}
	new := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: Ptr("Go")},
		},
	}
	entries := DiffStores(old, new)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Type != "removed" {
		t.Errorf("expected removed, got %s", entries[0].Type)
	}
}

func TestDiffStoresNoChanges(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: Ptr("Go")},
		},
	}
	entries := DiffStores(store, store)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestFormatDiff(t *testing.T) {
	entries := []DiffEntry{
		{Type: "added", ID: "STA-0001", Category: "Stack", Question: "Language?", NewValue: "Go"},
		{Type: "changed", ID: "DAT-0001", Category: "Data", Question: "DB?", OldValue: "SQLite", NewValue: "Postgres"},
	}
	result := FormatDiff(entries, "test project")
	if !strings.Contains(result, "Architectural Decisions Changed") {
		t.Error("missing header")
	}
	if !strings.Contains(result, "@STA-0001") {
		t.Error("missing STA-0001")
	}
	if !strings.Contains(result, "SQLite → Postgres") {
		t.Error("missing change text")
	}
}
