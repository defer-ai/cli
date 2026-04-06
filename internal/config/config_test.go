package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Empty(t *testing.T) {
	dir := t.TempDir()

	cfg, err := loadFile(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty", cfg.Model)
	}
	if cfg.Provider != "" {
		t.Errorf("Provider = %q, want empty", cfg.Provider)
	}
	if cfg.DomainCare != nil {
		t.Errorf("DomainCare = %v, want nil", cfg.DomainCare)
	}
}

func TestMerge_ProjectOverridesGlobal(t *testing.T) {
	base := &Config{
		Model:       "claude-3",
		Provider:    "anthropic",
		DefaultCare: "auto",
	}
	override := &Config{
		Model:       "gpt-4",
		DefaultCare: "review",
	}

	merged := merge(base, override)

	if merged.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", merged.Model, "gpt-4")
	}
	if merged.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q (should keep base)", merged.Provider, "anthropic")
	}
	if merged.DefaultCare != "review" {
		t.Errorf("DefaultCare = %q, want %q", merged.DefaultCare, "review")
	}
}

func TestMergeWithFlags(t *testing.T) {
	cfg := &Config{
		Model:    "claude-3",
		Provider: "anthropic",
		APIKey:   "sk-base",
	}

	MergeWithFlags(cfg, "gpt-4", "", "sk-flag")

	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q (empty flag should not override)", cfg.Provider, "anthropic")
	}
	if cfg.APIKey != "sk-flag" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-flag")
	}
}

func TestMerge_DomainCare(t *testing.T) {
	base := &Config{
		DomainCare: map[string]string{
			"auth":    "review",
			"logging": "auto",
		},
	}
	override := &Config{
		DomainCare: map[string]string{
			"logging": "review",
			"billing": "review",
		},
	}

	merged := merge(base, override)

	want := map[string]string{
		"auth":    "review",
		"logging": "review",
		"billing": "review",
	}
	if len(merged.DomainCare) != len(want) {
		t.Fatalf("DomainCare has %d keys, want %d", len(merged.DomainCare), len(want))
	}
	for k, wantV := range want {
		if gotV := merged.DomainCare[k]; gotV != wantV {
			t.Errorf("DomainCare[%q] = %q, want %q", k, gotV, wantV)
		}
	}
}

func TestMerge_HooksConcatenate(t *testing.T) {
	base := &Config{
		Hooks: map[string][]HookConfig{
			"pre-commit": {{Command: "lint"}},
		},
	}
	override := &Config{
		Hooks: map[string][]HookConfig{
			"pre-commit": {{Command: "test"}},
			"post-push":  {{URL: "https://example.com/hook"}},
		},
	}

	merged := merge(base, override)

	if got := len(merged.Hooks["pre-commit"]); got != 2 {
		t.Fatalf("pre-commit hooks = %d, want 2", got)
	}
	if merged.Hooks["pre-commit"][0].Command != "lint" {
		t.Errorf("first pre-commit hook = %q, want %q", merged.Hooks["pre-commit"][0].Command, "lint")
	}
	if merged.Hooks["pre-commit"][1].Command != "test" {
		t.Errorf("second pre-commit hook = %q, want %q", merged.Hooks["pre-commit"][1].Command, "test")
	}
	if got := len(merged.Hooks["post-push"]); got != 1 {
		t.Fatalf("post-push hooks = %d, want 1", got)
	}
}

func TestMerge_SkillDirsConcatenate(t *testing.T) {
	base := &Config{
		Skills: SkillsConfig{Dirs: []string{"/global/skills"}},
	}
	override := &Config{
		Skills: SkillsConfig{Dirs: []string{"/project/skills"}},
	}

	merged := merge(base, override)

	if got := len(merged.Skills.Dirs); got != 2 {
		t.Fatalf("Skills.Dirs len = %d, want 2", got)
	}
	if merged.Skills.Dirs[0] != "/global/skills" {
		t.Errorf("Skills.Dirs[0] = %q, want %q", merged.Skills.Dirs[0], "/global/skills")
	}
	if merged.Skills.Dirs[1] != "/project/skills" {
		t.Errorf("Skills.Dirs[1] = %q, want %q", merged.Skills.Dirs[1], "/project/skills")
	}
}

func TestLoadConfig_FullCascade(t *testing.T) {
	// Set up a fake home directory with a global config.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows uses USERPROFILE

	globalDir := filepath.Join(home, ".defer")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalCfg := Config{
		Model:       "claude-3",
		Provider:    "anthropic",
		DefaultCare: "auto",
		DomainCare:  map[string]string{"auth": "review"},
	}
	writeJSON(t, filepath.Join(globalDir, "config.json"), &globalCfg)

	// Set up project config.
	project := t.TempDir()
	projectDir := filepath.Join(project, ".defer")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	projectCfg := Config{
		Model:      "gpt-4",
		DomainCare: map[string]string{"billing": "review"},
	}
	writeJSON(t, filepath.Join(projectDir, "config.json"), &projectCfg)

	// Load and verify cascade.
	cfg, err := LoadConfig(project)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q (project overrides global)", cfg.Model, "gpt-4")
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q (kept from global)", cfg.Provider, "anthropic")
	}
	if cfg.DefaultCare != "auto" {
		t.Errorf("DefaultCare = %q, want %q (kept from global)", cfg.DefaultCare, "auto")
	}
	if cfg.DomainCare["auth"] != "review" {
		t.Errorf("DomainCare[auth] = %q, want %q", cfg.DomainCare["auth"], "review")
	}
	if cfg.DomainCare["billing"] != "review" {
		t.Errorf("DomainCare[billing] = %q, want %q", cfg.DomainCare["billing"], "review")
	}
}

func TestSaveProjectConfig(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Model:    "claude-3",
		Provider: "anthropic",
	}
	if err := SaveProjectConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(ProjectConfigPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Model != "claude-3" {
		t.Errorf("saved Model = %q, want %q", loaded.Model, "claude-3")
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
