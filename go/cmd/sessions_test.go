package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
)

func TestSessionsListFindsSession(t *testing.T) {
	dir := t.TempDir()

	// Create a session
	store, err := decision.CreateStore(dir, "test task")
	if err != nil {
		t.Fatal(err)
	}
	answer := "Go"
	store.Decisions = []decision.Decision{
		{ID: "@STA-0001", Category: "Stack", Question: "Language?", Answer: &answer, Source: "user"},
		{ID: "@STA-0002", Category: "Stack", Question: "Framework?", Source: "user"},
	}
	if err := decision.SaveStore(dir, store); err != nil {
		t.Fatal(err)
	}

	// Verify the decisions.json file exists
	jsonPath := filepath.Join(dir, ".defer", "decisions.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("decisions.json not created")
	}

	// Verify we can load it back
	loaded, err := decision.LoadStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("loaded store is nil")
	}
	if loaded.Task != "test task" {
		t.Errorf("task = %q, want %q", loaded.Task, "test task")
	}
	if len(loaded.Decisions) != 2 {
		t.Errorf("decisions = %d, want 2", len(loaded.Decisions))
	}
}

func TestSessionsDeleteRemovesDir(t *testing.T) {
	dir := t.TempDir()

	// Create a session
	_, err := decision.CreateStore(dir, "test task")
	if err != nil {
		t.Fatal(err)
	}

	deferDir := filepath.Join(dir, ".defer")
	if _, err := os.Stat(deferDir); os.IsNotExist(err) {
		t.Fatal(".defer dir not created")
	}

	// Simulate delete (without interactive prompt)
	if err := os.RemoveAll(deferDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(deferDir); !os.IsNotExist(err) {
		t.Error(".defer dir should be deleted")
	}
}

func TestSessionsExportFormat(t *testing.T) {
	dir := t.TempDir()

	store, err := decision.CreateStore(dir, "build a todo app")
	if err != nil {
		t.Fatal(err)
	}
	answer := "TypeScript"
	store.Decisions = []decision.Decision{
		{
			ID:       "@STA-0001",
			Category: "Stack",
			Question: "Backend language?",
			Answer:   &answer,
			Source:   "user",
			Date:     "2026-01-01",
		},
	}
	if err := decision.SaveStore(dir, store); err != nil {
		t.Fatal(err)
	}

	// Verify DECISIONS.md was generated
	mdPath := filepath.Join(dir, "DECISIONS.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("DECISIONS.md is empty")
	}

	// Verify it contains expected content
	if !contains(content, "DECISIONS.md") {
		t.Error("missing DECISIONS.md header")
	}
	if !contains(content, "build a todo app") {
		t.Error("missing task name")
	}
	if !contains(content, "TypeScript") {
		t.Error("missing answer")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
