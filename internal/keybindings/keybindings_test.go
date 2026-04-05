package keybindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDefaultBindingsContainsAllActions(t *testing.T) {
	b := DefaultBindings()

	expected := []Action{
		ActionNavigateUp, ActionNavigateDown,
		ActionInspect, ActionBack,
		ActionSearch, ActionChat,
		ActionCustom, ActionShuffle,
		ActionWhy, ActionAsk,
		ActionConfirm, ActionCancel,
		ActionQuit,
		ActionCareUp, ActionCareDown,
	}

	for _, action := range expected {
		keys, ok := b[action]
		if !ok {
			t.Errorf("DefaultBindings() missing action %q", action)
			continue
		}
		if len(keys) == 0 {
			t.Errorf("DefaultBindings() action %q has no keys", action)
		}
	}
}

func TestDefaultBindingsSpecificKeys(t *testing.T) {
	b := DefaultBindings()

	tests := []struct {
		action Action
		want   []string
	}{
		{ActionNavigateUp, []string{"up"}},
		{ActionNavigateDown, []string{"down"}},
		{ActionInspect, []string{"enter"}},
		{ActionBack, []string{"q", "esc"}},
		{ActionSearch, []string{"/"}},
		{ActionChat, []string{"tab"}},
		{ActionCustom, []string{"c"}},
		{ActionShuffle, []string{"s"}},
		{ActionWhy, []string{"w"}},
		{ActionAsk, []string{"a"}},
		{ActionConfirm, []string{"enter"}},
		{ActionCancel, []string{"esc"}},
		{ActionQuit, []string{"ctrl+c"}},
		{ActionCareUp, []string{"l", "right"}},
		{ActionCareDown, []string{"h", "left"}},
	}

	for _, tt := range tests {
		got := b[tt.action]
		if !slices.Equal(got, tt.want) {
			t.Errorf("DefaultBindings()[%q] = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestResolveFindsAction(t *testing.T) {
	b := DefaultBindings()

	tests := []struct {
		key  string
		want Action
	}{
		{"up", ActionNavigateUp},
		{"up", ActionNavigateUp},
		{"down", ActionNavigateDown},
		{"down", ActionNavigateDown},
		{"/", ActionSearch},
		{"tab", ActionChat},
		{"c", ActionCustom},
		{"s", ActionShuffle},
		{"w", ActionWhy},
		{"a", ActionAsk},
		{"ctrl+c", ActionQuit},
		{"l", ActionCareUp},
		{"right", ActionCareUp},
		{"h", ActionCareDown},
		{"left", ActionCareDown},
	}

	for _, tt := range tests {
		got := b.Resolve(tt.key)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestResolveUnboundKeyReturnsEmpty(t *testing.T) {
	b := DefaultBindings()
	got := b.Resolve("f12")
	if got != "" {
		t.Errorf("Resolve(\"f12\") = %q, want empty", got)
	}
}

func TestResolveAllMultipleActions(t *testing.T) {
	b := DefaultBindings()

	// "enter" is bound to both inspect and confirm
	actions := b.ResolveAll("enter")
	if len(actions) < 2 {
		t.Fatalf("ResolveAll(\"enter\") returned %d actions, want at least 2", len(actions))
	}
	foundInspect := slices.Contains(actions, ActionInspect)
	foundConfirm := slices.Contains(actions, ActionConfirm)
	if !foundInspect || !foundConfirm {
		t.Errorf("ResolveAll(\"enter\") = %v, want both inspect and confirm", actions)
	}
}

func TestLoadBindingsFromCustomFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.json")

	// Override navigate.up to use "w" and "up" instead of defaults
	custom := map[string][]string{
		"navigate.up": {"w", "up"},
	}
	data, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	b := LoadBindingsFrom(path)

	// Overridden action should have new keys
	got := b[ActionNavigateUp]
	want := []string{"w", "up"}
	if !slices.Equal(got, want) {
		t.Errorf("navigate.up = %v, want %v", got, want)
	}

	// Non-overridden actions should keep defaults
	gotDown := b[ActionNavigateDown]
	wantDown := []string{"down"}
	if !slices.Equal(gotDown, wantDown) {
		t.Errorf("navigate.down = %v, want %v (should keep default)", gotDown, wantDown)
	}
}

func TestLoadBindingsFromReplacesNotAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.json")

	// Override navigate.up to only have one key
	custom := map[string][]string{
		"navigate.up": {"w"},
	}
	data, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	b := LoadBindingsFrom(path)

	got := b[ActionNavigateUp]
	want := []string{"w"}
	if !slices.Equal(got, want) {
		t.Errorf("navigate.up = %v, want %v (replace, not append)", got, want)
	}
}

func TestLoadBindingsFromMissingFileReturnsDefaults(t *testing.T) {
	b := LoadBindingsFrom("/nonexistent/path/keybindings.json")
	defaults := DefaultBindings()

	if len(b) != len(defaults) {
		t.Errorf("LoadBindingsFrom(missing) has %d actions, want %d", len(b), len(defaults))
	}

	for action, wantKeys := range defaults {
		gotKeys := b[action]
		if !slices.Equal(gotKeys, wantKeys) {
			t.Errorf("action %q = %v, want %v", action, gotKeys, wantKeys)
		}
	}
}

func TestLoadBindingsFromInvalidJSONReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.json")
	if err := os.WriteFile(path, []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := LoadBindingsFrom(path)
	defaults := DefaultBindings()

	if len(b) != len(defaults) {
		t.Errorf("LoadBindingsFrom(invalid) has %d actions, want %d", len(b), len(defaults))
	}
}

func TestLoadBindingsFromCustomAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.json")

	custom := map[string][]string{
		"my.custom.action": {"x"},
	}
	data, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	b := LoadBindingsFrom(path)

	got := b[Action("my.custom.action")]
	want := []string{"x"}
	if !slices.Equal(got, want) {
		t.Errorf("my.custom.action = %v, want %v", got, want)
	}

	// Resolve should find it
	action := b.Resolve("x")
	if action != Action("my.custom.action") {
		t.Errorf("Resolve(\"x\") = %q, want %q", action, "my.custom.action")
	}
}
