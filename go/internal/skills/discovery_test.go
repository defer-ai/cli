package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverSkillDirsFindsNestedDirs(t *testing.T) {
	// Create:
	//   tmpdir/
	//     .defer/skills/         (shallow)
	//     sub/
	//       .defer/skills/       (deep)
	root := t.TempDir()

	shallow := filepath.Join(root, ".defer", "skills")
	deep := filepath.Join(root, "sub", ".defer", "skills")
	if err := os.MkdirAll(shallow, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	dirs := DiscoverSkillDirs(filepath.Join(root, "sub"))

	if len(dirs) != 2 {
		t.Fatalf("DiscoverSkillDirs returned %d dirs, want 2: %v", len(dirs), dirs)
	}

	// Shallowest should come first
	if dirs[0] != shallow {
		t.Errorf("dirs[0] = %q, want %q (shallow first)", dirs[0], shallow)
	}
	if dirs[1] != deep {
		t.Errorf("dirs[1] = %q, want %q (deep last)", dirs[1], deep)
	}
}

func TestDiscoverSkillDirsEmpty(t *testing.T) {
	root := t.TempDir()
	dirs := DiscoverSkillDirs(root)
	if len(dirs) != 0 {
		t.Errorf("DiscoverSkillDirs on empty tree returned %d dirs, want 0", len(dirs))
	}
}

func TestDiscoverSkillDirsSingleLevel(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".defer", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dirs := DiscoverSkillDirs(root)
	if len(dirs) != 1 {
		t.Fatalf("DiscoverSkillDirs returned %d dirs, want 1", len(dirs))
	}
	if dirs[0] != skillDir {
		t.Errorf("dirs[0] = %q, want %q", dirs[0], skillDir)
	}
}

func TestDiscoverSkillDirsThreeLevels(t *testing.T) {
	// Create:
	//   tmpdir/
	//     .defer/skills/
	//     a/
	//       .defer/skills/
	//       b/
	//         .defer/skills/
	root := t.TempDir()

	level0 := filepath.Join(root, ".defer", "skills")
	level1 := filepath.Join(root, "a", ".defer", "skills")
	level2 := filepath.Join(root, "a", "b", ".defer", "skills")
	for _, d := range []string{level0, level1, level2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	dirs := DiscoverSkillDirs(filepath.Join(root, "a", "b"))
	if len(dirs) != 3 {
		t.Fatalf("DiscoverSkillDirs returned %d dirs, want 3: %v", len(dirs), dirs)
	}
	if dirs[0] != level0 {
		t.Errorf("dirs[0] = %q, want %q", dirs[0], level0)
	}
	if dirs[1] != level1 {
		t.Errorf("dirs[1] = %q, want %q", dirs[1], level1)
	}
	if dirs[2] != level2 {
		t.Errorf("dirs[2] = %q, want %q", dirs[2], level2)
	}
}

func TestMergeSkillsDefaultsOnly(t *testing.T) {
	defaults := map[string]Skill{
		"decompose": {Name: "decompose", Prompt: "default prompt"},
		"plan":      {Name: "plan", Prompt: "plan prompt"},
	}

	merged := MergeSkills(defaults, nil)

	if len(merged) != 2 {
		t.Fatalf("MergeSkills returned %d skills, want 2", len(merged))
	}
	if merged["decompose"].Prompt != "default prompt" {
		t.Errorf("decompose prompt = %q, want %q", merged["decompose"].Prompt, "default prompt")
	}
}

func TestMergeSkillsOverrideByName(t *testing.T) {
	defaults := map[string]Skill{
		"decompose": {Name: "decompose", Prompt: "default prompt"},
		"plan":      {Name: "plan", Prompt: "plan prompt"},
	}

	loaded := []Skill{
		{Name: "decompose", Prompt: "custom prompt"},
		{Name: "my-skill", Prompt: "new skill prompt"},
	}

	merged := MergeSkills(defaults, loaded)

	if len(merged) != 3 {
		t.Fatalf("MergeSkills returned %d skills, want 3", len(merged))
	}
	if merged["decompose"].Prompt != "custom prompt" {
		t.Errorf("decompose prompt = %q, want %q (should be overridden)", merged["decompose"].Prompt, "custom prompt")
	}
	if merged["plan"].Prompt != "plan prompt" {
		t.Errorf("plan prompt = %q, want %q (should be unchanged)", merged["plan"].Prompt, "plan prompt")
	}
	if merged["my-skill"].Prompt != "new skill prompt" {
		t.Errorf("my-skill prompt = %q, want %q", merged["my-skill"].Prompt, "new skill prompt")
	}
}

func TestMergeSkillsEmptyDefaults(t *testing.T) {
	loaded := []Skill{
		{Name: "custom", Prompt: "custom prompt"},
	}

	merged := MergeSkills(nil, loaded)

	if len(merged) != 1 {
		t.Fatalf("MergeSkills returned %d skills, want 1", len(merged))
	}
	if merged["custom"].Prompt != "custom prompt" {
		t.Errorf("custom prompt = %q, want %q", merged["custom"].Prompt, "custom prompt")
	}
}

func TestMergeSkillsBothEmpty(t *testing.T) {
	merged := MergeSkills(nil, nil)
	if len(merged) != 0 {
		t.Errorf("MergeSkills(nil, nil) returned %d skills, want 0", len(merged))
	}
}

func TestWatchSkillDirsDetectsNewFile(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".defer", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Use a fast poll interval for testing
	notify, stop := WatchSkillDirsInterval([]string{skillDir}, 50*time.Millisecond)

	// Write a new file after the watcher starts
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(skillDir, "new.md"), []byte("new skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for notification
	select {
	case _, ok := <-notify:
		if !ok {
			t.Error("notify channel closed unexpectedly")
		}
		// success: we received a change notification
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for change notification")
	}

	close(stop)
}

func TestWatchSkillDirsDetectsModification(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".defer", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	existingFile := filepath.Join(skillDir, "existing.md")
	if err := os.WriteFile(existingFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	notify, stop := WatchSkillDirsInterval([]string{skillDir}, 50*time.Millisecond)

	// Modify the file after the watcher starts.
	// Ensure mod time changes by waiting a bit.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(existingFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-notify:
		if !ok {
			t.Error("notify channel closed unexpectedly")
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for modification notification")
	}

	close(stop)
}

func TestWatchSkillDirsStopsCleanly(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".defer", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	notify, stop := WatchSkillDirsInterval([]string{skillDir}, 50*time.Millisecond)

	close(stop)

	// The notify channel should eventually close
	select {
	case _, ok := <-notify:
		if ok {
			// Got a spurious notification, that's fine; drain and wait for close
			select {
			case <-notify:
			case <-time.After(1 * time.Second):
				t.Error("notify channel did not close after stop")
			}
		}
		// ok == false means channel closed, which is expected
	case <-time.After(1 * time.Second):
		t.Error("notify channel did not close after stop")
	}
}

func TestChangedDetectsDifferences(t *testing.T) {
	now := time.Now()

	prev := map[string]time.Time{
		"/a/b.md": now,
		"/a/c.md": now,
	}

	// Same - no change
	curr := map[string]time.Time{
		"/a/b.md": now,
		"/a/c.md": now,
	}
	if changed(prev, curr) {
		t.Error("changed() should return false for identical snapshots")
	}

	// Different mod time
	curr2 := map[string]time.Time{
		"/a/b.md": now.Add(time.Second),
		"/a/c.md": now,
	}
	if !changed(prev, curr2) {
		t.Error("changed() should return true for different mod times")
	}

	// New file
	curr3 := map[string]time.Time{
		"/a/b.md": now,
		"/a/c.md": now,
		"/a/d.md": now,
	}
	if !changed(prev, curr3) {
		t.Error("changed() should return true for new files")
	}

	// Removed file
	curr4 := map[string]time.Time{
		"/a/b.md": now,
	}
	if !changed(prev, curr4) {
		t.Error("changed() should return true for removed files")
	}
}
