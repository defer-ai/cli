package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPresetsReturnsThree(t *testing.T) {
	presets := DefaultPresets()
	if len(presets) != 3 {
		t.Fatalf("expected 3 default presets, got %d", len(presets))
	}
	names := map[string]bool{}
	for _, p := range presets {
		names[p.Name] = true
	}
	for _, want := range []string{"rest-api", "cli-tool", "web-app"} {
		if !names[want] {
			t.Errorf("missing default preset %q", want)
		}
	}
}

func TestDefaultPresetsHaveDecisions(t *testing.T) {
	for _, p := range DefaultPresets() {
		if len(p.Decisions) < 5 {
			t.Errorf("preset %q has only %d decisions, expected at least 5", p.Name, len(p.Decisions))
		}
		if len(p.Decisions) > 8 {
			t.Errorf("preset %q has %d decisions, expected at most 8", p.Name, len(p.Decisions))
		}
		if p.Source != "builtin" {
			t.Errorf("preset %q source = %q, want %q", p.Name, p.Source, "builtin")
		}
		for _, d := range p.Decisions {
			if d.Category == "" {
				t.Errorf("preset %q has a decision with empty category", p.Name)
			}
			if d.Question == "" {
				t.Errorf("preset %q has a decision with empty question", p.Name)
			}
			if len(d.Options) < 2 {
				t.Errorf("preset %q decision %q has fewer than 2 options", p.Name, d.Question)
			}
		}
	}
}

func TestLoadPresetFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `name: test-preset
description: A test preset
decisions:
  - category: Stack
    question: "Language?"
    options:
      - key: A
        label: Go
      - key: B
        label: Rust
    impact: 9
    context: "Foundational"
  - category: Data
    question: "Database?"
    options:
      - key: A
        label: PostgreSQL
      - key: B
        label: SQLite
    impact: 7
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadPresetFile(path)
	if err != nil {
		t.Fatalf("LoadPresetFile failed: %v", err)
	}
	if p.Name != "test-preset" {
		t.Errorf("name = %q, want %q", p.Name, "test-preset")
	}
	if p.Description != "A test preset" {
		t.Errorf("description = %q, want %q", p.Description, "A test preset")
	}
	if len(p.Decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(p.Decisions))
	}
	d := p.Decisions[0]
	if d.Category != "Stack" {
		t.Errorf("decision[0].Category = %q, want %q", d.Category, "Stack")
	}
	if d.Question != "Language?" {
		t.Errorf("decision[0].Question = %q, want %q", d.Question, "Language?")
	}
	if len(d.Options) != 2 {
		t.Errorf("decision[0] has %d options, want 2", len(d.Options))
	}
	if d.Impact != 9 {
		t.Errorf("decision[0].Impact = %d, want 9", d.Impact)
	}
	if d.Context != "Foundational" {
		t.Errorf("decision[0].Context = %q, want %q", d.Context, "Foundational")
	}
}

func TestLoadPresetFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::not valid yaml{{["), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPresetFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadPresetFileMissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noname.yaml")
	content := `description: no name field
decisions: []
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPresetFile(path)
	if err == nil {
		t.Fatal("expected error for preset with missing name, got nil")
	}
}

func TestLoadPresetFileNotFound(t *testing.T) {
	_, err := LoadPresetFile("/nonexistent/path/to/preset.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestDiscoverPresetsFindsProjectLocal(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, ".defer", "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `name: my-project
description: Custom project preset
decisions:
  - category: Custom
    question: "Custom choice?"
    options:
      - key: A
        label: Option A
      - key: B
        label: Option B
    impact: 5
`
	if err := os.WriteFile(filepath.Join(tmplDir, "custom.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	presets := DiscoverPresets(dir)

	found := false
	for _, p := range presets {
		if p.Name == "my-project" {
			found = true
			if p.Source != "project" {
				t.Errorf("project preset source = %q, want %q", p.Source, "project")
			}
			if len(p.Decisions) != 1 {
				t.Errorf("project preset has %d decisions, want 1", len(p.Decisions))
			}
		}
	}
	if !found {
		t.Error("project-local preset 'my-project' not found in DiscoverPresets results")
	}

	// Should also include the 3 builtins
	if len(presets) < 4 {
		t.Errorf("expected at least 4 presets (3 builtin + 1 project), got %d", len(presets))
	}
}

func TestDiscoverPresetsProjectOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, ".defer", "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Override the built-in rest-api preset
	content := `name: rest-api
description: My custom REST API preset
decisions:
  - category: Custom
    question: "Overridden?"
    options:
      - key: A
        label: Yes
      - key: B
        label: No
    impact: 5
`
	if err := os.WriteFile(filepath.Join(tmplDir, "rest-api.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	presets := DiscoverPresets(dir)

	for _, p := range presets {
		if p.Name == "rest-api" {
			if p.Source != "project" {
				t.Errorf("overridden rest-api source = %q, want %q", p.Source, "project")
			}
			if p.Description != "My custom REST API preset" {
				t.Errorf("overridden rest-api description = %q, want custom", p.Description)
			}
			return
		}
	}
	t.Error("rest-api preset not found after override")
}
