package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/defer-ai/cli/internal/decision"
)

// Entry represents a completed session.
type Entry struct {
	Task        string              `json:"task"`
	Decisions   []decision.Decision `json:"decisions"`
	CompletedAt string              `json:"completedAt"`
	Duration    int64               `json:"duration"` // ms
}

func historyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".defer", "history")
	return dir, os.MkdirAll(dir, 0o755)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(text string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(text), "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

// Save persists a completed session.
func Save(task string, decisions []decision.Decision, duration int64) (string, error) {
	dir, err := historyDir()
	if err != nil {
		return "", err
	}

	ts := strings.ReplaceAll(strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339), ":", "-"), ".", "-")
	filename := fmt.Sprintf("%s-%s.json", ts, slugify(task))

	entry := Entry{
		Task:        task,
		Decisions:   decisions,
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
		Duration:    duration,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return "", err
	}
	return filename, os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}

// List returns recent history filenames, newest first.
func List(limit int) ([]string, error) {
	dir, err := historyDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

// Load reads a specific history entry.
func Load(filename string) (*Entry, error) {
	dir, err := historyDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return nil, err
	}
	var entry Entry
	return &entry, json.Unmarshal(data, &entry)
}
