package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	deferDir      = ".defer"
	decisionsJSON = "decisions.json"
	lockFile      = "decisions.lock"
)

func dirPath(cwd string) string  { return filepath.Join(cwd, deferDir) }
func jsonPath(cwd string) string { return filepath.Join(cwd, deferDir, decisionsJSON) }
func lockPath(cwd string) string { return filepath.Join(cwd, deferDir, lockFile) }
func mdPath(cwd string) string   { return filepath.Join(cwd, "DECISIONS.md") }

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

// SaveStore writes the store to disk atomically and regenerates DECISIONS.md.
func SaveStore(cwd string, store *DecisionStore) error {
	if err := EnsureDir(cwd); err != nil {
		return err
	}
	store.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: write to .tmp then rename
	target := jsonPath(cwd)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp) // cleanup on rename failure
		return err
	}
	return GenerateMarkdown(cwd, store)
}

// WithStoreLock acquires an advisory file lock on .defer/decisions.lock,
// executes fn, and releases the lock. This prevents concurrent processes
// from corrupting the store.
func WithStoreLock(cwd string, fn func() error) error {
	if err := EnsureDir(cwd); err != nil {
		return err
	}

	fl := flock.New(lockPath(cwd))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquire store lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("timeout waiting for store lock (another defer process may be writing)")
	}
	defer fl.Unlock()

	return fn()
}

// SaveStoreLocked acquires the lock, saves the store, and releases the lock.
func SaveStoreLocked(cwd string, store *DecisionStore) error {
	return WithStoreLock(cwd, func() error {
		return SaveStore(cwd, store)
	})
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
