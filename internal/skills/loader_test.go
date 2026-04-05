package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillFileParseFrontmatterAndContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "decompose.md")
	content := `---
name: decompose
description: Break task into decisions
when-to-use: When starting a new task
---
You are in DEFER MODE. Your ONLY job is to identify decisions.

Do NOT write code.`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skill, err := LoadSkillFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Name != "decompose" {
		t.Errorf("Name = %q, want %q", skill.Name, "decompose")
	}
	if skill.Description != "Break task into decisions" {
		t.Errorf("Description = %q, want %q", skill.Description, "Break task into decisions")
	}
	if skill.Metadata["when-to-use"] != "When starting a new task" {
		t.Errorf("Metadata[when-to-use] = %q, want %q", skill.Metadata["when-to-use"], "When starting a new task")
	}
	if !strings.HasPrefix(skill.Prompt, "You are in DEFER MODE") {
		t.Errorf("Prompt should start with 'You are in DEFER MODE', got: %q", skill.Prompt[:min(50, len(skill.Prompt))])
	}
	if !strings.Contains(skill.Prompt, "Do NOT write code.") {
		t.Error("Prompt should contain 'Do NOT write code.'")
	}
	if skill.Path != path {
		t.Errorf("Path = %q, want %q", skill.Path, path)
	}
}

func TestLoadSkillFileNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.md")
	content := `Just a plain prompt without frontmatter.

Do things.`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skill, err := LoadSkillFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Name != "custom" {
		t.Errorf("Name = %q, want %q (derived from filename)", skill.Name, "custom")
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
	if !strings.Contains(skill.Prompt, "Just a plain prompt") {
		t.Error("Prompt should contain the full file content when no frontmatter")
	}
}

func TestLoadSkillFileNameFromFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-skill.md")
	content := `---
description: A skill without explicit name
---
The prompt.`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skill, err := LoadSkillFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Name != "my-skill" {
		t.Errorf("Name = %q, want %q (derived from filename)", skill.Name, "my-skill")
	}
}

func TestDefaultSkillsReturnsAll6(t *testing.T) {
	defaults := DefaultSkills()

	expected := []string{"decompose", "execute", "extract", "verify"}
	if len(defaults) != len(expected) {
		t.Errorf("DefaultSkills() returned %d skills, want %d", len(defaults), len(expected))
	}

	for _, name := range expected {
		skill, ok := defaults[name]
		if !ok {
			t.Errorf("DefaultSkills() missing %q", name)
			continue
		}
		if skill.Name != name {
			t.Errorf("skill %q has Name = %q", name, skill.Name)
		}
		if skill.Prompt == "" {
			t.Errorf("skill %q has empty Prompt", name)
		}
		if skill.Description == "" {
			t.Errorf("skill %q has empty Description", name)
		}
		if len(skill.Metadata) == 0 {
			t.Errorf("skill %q has empty Metadata", name)
		}
	}
}

func TestDefaultSkillsPromptsMatchAgent(t *testing.T) {
	defaults := DefaultSkills()

	// Verify the prompts are not empty and contain expected content
	if !strings.Contains(defaults["decompose"].Prompt, "DEFER MODE") {
		t.Error("decompose prompt should contain 'DEFER MODE'")
	}
	if !strings.Contains(defaults["decompose"].Prompt, "Scan the existing codebase") {
		t.Error("decompose prompt should contain codebase scanning instructions")
	}
	if !strings.Contains(defaults["verify"].Prompt, "VERIFIED OK") {
		t.Error("verify prompt should contain 'VERIFIED OK'")
	}
	if !strings.Contains(defaults["extract"].Prompt, "extract every decision") {
		t.Error("extract prompt should contain 'extract every decision'")
	}
	if !strings.Contains(defaults["execute"].Prompt, "implementing a software project") {
		t.Error("execute prompt should contain 'implementing a software project'")
	}
}

func TestLoadSkillsFromTempDirectory(t *testing.T) {
	// Create a directory structure:
	//   tmpdir/
	//     .defer/skills/
	//       decompose.md
	//       custom.md
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".defer", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	decomposeContent := `---
name: decompose
description: Custom decompose
---
Custom decompose prompt.`

	customContent := `---
name: my-custom
description: A custom skill
---
Do custom things.`

	if err := os.WriteFile(filepath.Join(skillsDir, "decompose.md"), []byte(decomposeContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "custom.md"), []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadSkills(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(skills) != 2 {
		t.Fatalf("LoadSkills returned %d skills, want 2", len(skills))
	}

	// Check both skills loaded
	found := make(map[string]Skill)
	for _, s := range skills {
		found[s.Name] = s
	}

	if _, ok := found["decompose"]; !ok {
		t.Error("missing 'decompose' skill")
	} else if found["decompose"].Prompt != "Custom decompose prompt." {
		t.Errorf("decompose prompt = %q, want %q", found["decompose"].Prompt, "Custom decompose prompt.")
	}

	if _, ok := found["my-custom"]; !ok {
		t.Error("missing 'my-custom' skill")
	} else if found["my-custom"].Prompt != "Do custom things." {
		t.Errorf("my-custom prompt = %q, want %q", found["my-custom"].Prompt, "Do custom things.")
	}
}

func TestLoadSkillsDeeperOverridesShallower(t *testing.T) {
	// Create:
	//   tmpdir/
	//     .defer/skills/decompose.md  (shallow, original)
	//     sub/
	//       .defer/skills/decompose.md  (deep, override)
	dir := t.TempDir()

	shallowDir := filepath.Join(dir, ".defer", "skills")
	if err := os.MkdirAll(shallowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shallowContent := `---
name: decompose
description: Shallow decompose
---
Shallow prompt.`
	if err := os.WriteFile(filepath.Join(shallowDir, "decompose.md"), []byte(shallowContent), 0o644); err != nil {
		t.Fatal(err)
	}

	deepDir := filepath.Join(dir, "sub", ".defer", "skills")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	deepContent := `---
name: decompose
description: Deep decompose
---
Deep prompt.`
	if err := os.WriteFile(filepath.Join(deepDir, "decompose.md"), []byte(deepContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load from sub/ directory -- deeper should override shallower
	skills, err := LoadSkills(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]Skill)
	for _, s := range skills {
		found[s.Name] = s
	}

	decompose, ok := found["decompose"]
	if !ok {
		t.Fatal("missing 'decompose' skill")
	}
	if decompose.Prompt != "Deep prompt." {
		t.Errorf("decompose prompt = %q, want %q (deeper should override shallower)", decompose.Prompt, "Deep prompt.")
	}
	if decompose.Description != "Deep decompose" {
		t.Errorf("decompose description = %q, want %q", decompose.Description, "Deep decompose")
	}
}

func TestLoadSkillsEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	skills, err := LoadSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("LoadSkills on empty dir returned %d skills, want 0", len(skills))
	}
}

func TestLoadSkillsSkipsNonMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".defer", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a .txt file (should be ignored) and a .md file
	if err := os.WriteFile(filepath.Join(skillsDir, "ignored.txt"), []byte("not a skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "real.md"), []byte("---\nname: real\n---\nReal prompt."), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Errorf("LoadSkills returned %d skills, want 1 (should skip .txt)", len(skills))
	}
	if len(skills) > 0 && skills[0].Name != "real" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "real")
	}
}

func TestLoadSkillFileNonexistent(t *testing.T) {
	_, err := LoadSkillFile("/nonexistent/path/skill.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
