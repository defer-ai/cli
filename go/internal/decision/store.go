package decision

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	deferDir     = ".defer"
	decisionsJSON = "decisions.json"
)

func dirPath(cwd string) string   { return filepath.Join(cwd, deferDir) }
func jsonPath(cwd string) string  { return filepath.Join(cwd, deferDir, decisionsJSON) }
func mdPath(cwd string) string    { return filepath.Join(cwd, "DECISIONS.md") }

// EnsureDir creates the .defer directory and .gitignore if needed.
func EnsureDir(cwd string) error {
	dir := dirPath(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	gi := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gi); os.IsNotExist(err) {
		_ = os.WriteFile(gi, []byte("*\n"), 0o644)
	}
	return nil
}

// StoreExists checks if a decision store exists.
func StoreExists(cwd string) bool {
	_, err := os.Stat(jsonPath(cwd))
	return err == nil
}

// LoadStore loads the decision store from disk. Returns nil if not found.
func LoadStore(cwd string) (*DecisionStore, error) {
	data, err := os.ReadFile(jsonPath(cwd))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var store DecisionStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

// SaveStore writes the store to disk and regenerates DECISIONS.md.
func SaveStore(cwd string, store *DecisionStore) error {
	if err := EnsureDir(cwd); err != nil {
		return err
	}
	store.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath(cwd), data, 0o644); err != nil {
		return err
	}
	return GenerateMarkdown(cwd, store)
}

// CreateStore creates a new empty store.
func CreateStore(cwd string, task string) (*DecisionStore, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	store := &DecisionStore{
		Task:      task,
		Decisions: []Decision{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := SaveStore(cwd, store); err != nil {
		return nil, err
	}
	return store, nil
}
