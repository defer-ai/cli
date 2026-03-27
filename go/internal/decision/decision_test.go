package decision

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	dir := t.TempDir()

	store, err := LoadStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store != nil {
		t.Fatal("expected nil store")
	}

	store, err = CreateStore(dir, "test task")
	if err != nil {
		t.Fatal(err)
	}
	if store.Task != "test task" {
		t.Errorf("task = %q, want %q", store.Task, "test task")
	}

	gi := filepath.Join(dir, ".defer", ".gitignore")
	if _, err := os.Stat(gi); os.IsNotExist(err) {
		t.Error(".gitignore not created")
	}

	answer := "TypeScript"
	store.Decisions = append(store.Decisions, Decision{
		ID:       NextID(nil, "Stack"),
		Category: "Stack",
		Question: "Backend language?",
		Answer:   &answer,
		Source:   "user",
		Date:     "2026-01-01",
	})
	if err := SaveStore(dir, store); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Decisions) != 1 {
		t.Fatalf("got %d decisions, want 1", len(loaded.Decisions))
	}
	if loaded.Decisions[0].StrAnswer() != "TypeScript" {
		t.Errorf("answer = %q, want TypeScript", loaded.Decisions[0].StrAnswer())
	}

	md := filepath.Join(dir, "DECISIONS.md")
	data, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("DECISIONS.md is empty")
	}
}
