package cmd

import (
	"os"
	"testing"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

// setupSourceProject creates a source project with decisions and returns its path.
func setupSourceProject(t *testing.T, decisions []decision.Decision) string {
	t.Helper()
	dir := t.TempDir()
	store := &decision.DecisionStore{
		Task:      "source project",
		Decisions: decisions,
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	if err := decision.SaveStore(dir, store); err != nil {
		t.Fatal(err)
	}
	return dir
}

// setupTargetProject creates a target project with decisions and returns its path.
func setupTargetProject(t *testing.T, decisions []decision.Decision) string {
	t.Helper()
	dir := t.TempDir()
	store, err := decision.CreateStore(dir, "target project")
	if err != nil {
		t.Fatal(err)
	}
	if decisions != nil {
		store.Decisions = decisions
		if err := decision.SaveStore(dir, store); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// execImport runs the import command in the given working directory with the given args.
// It temporarily chdir's and resets package-level flags.
func execImport(t *testing.T, cwd string, args ...string) error {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	// Reset package-level flags to defaults.
	importCategory = ""
	importKeepAnswers = false

	// Call runImport directly with a fresh cobra.Command for output.
	cmd := &cobra.Command{}
	cmd.SetArgs(args)

	// Parse flags from args: extract --category and --keep-answers.
	var positional []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--category" && i+1 < len(args):
			importCategory = args[i+1]
			i++ // skip value
		case args[i] == "--keep-answers":
			importKeepAnswers = true
		default:
			positional = append(positional, args[i])
		}
	}

	return runImport(cmd, positional)
}

func TestImportIDRemapping(t *testing.T) {
	answer := "Go"
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: &answer, Source: "user"},
	})
	// Target already has STA-0001, so imported one should become STA-0002.
	targetDir := setupTargetProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Existing?"},
	})

	if err := execImport(t, targetDir, sourceDir, "@STA-0001"); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(store.Decisions))
	}

	imported := store.Decisions[1]
	if imported.ID != "STA-0002" {
		t.Errorf("expected imported ID STA-0002, got %s", imported.ID)
	}
	if imported.ImportedFrom == nil {
		t.Fatal("expected ImportedFrom to be set")
	}
	if imported.ImportedFrom.OriginalID != "STA-0001" {
		t.Errorf("expected OriginalID STA-0001, got %s", imported.ImportedFrom.OriginalID)
	}
	if imported.ImportedFrom.Project != sourceDir {
		t.Errorf("expected Project %s, got %s", sourceDir, imported.ImportedFrom.Project)
	}
	// Default: answer should be cleared.
	if imported.Answer != nil {
		t.Errorf("expected Answer to be nil (pending), got %v", *imported.Answer)
	}
}

func TestImportDependencyRemapping(t *testing.T) {
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?"},
		{ID: "STA-0002", Category: "Stack", Question: "Framework?", DependsOn: []string{"STA-0001"}},
	})
	// Target already has STA-0001, so source STA-0001 -> STA-0002, source STA-0002 -> STA-0003.
	targetDir := setupTargetProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Existing?"},
	})

	if err := execImport(t, targetDir, sourceDir); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(store.Decisions))
	}

	// Source STA-0001 -> STA-0002, Source STA-0002 -> STA-0003
	imported1 := store.Decisions[1]
	imported2 := store.Decisions[2]

	if imported1.ID != "STA-0002" {
		t.Errorf("expected first import STA-0002, got %s", imported1.ID)
	}
	if imported2.ID != "STA-0003" {
		t.Errorf("expected second import STA-0003, got %s", imported2.ID)
	}

	// The dependency should be remapped from STA-0001 to STA-0002.
	if len(imported2.DependsOn) != 1 || imported2.DependsOn[0] != "STA-0002" {
		t.Errorf("expected DependsOn [STA-0002], got %v", imported2.DependsOn)
	}
}

func TestImportCategoryFilter(t *testing.T) {
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?"},
		{ID: "DAT-0001", Category: "Data", Question: "Database?"},
		{ID: "STA-0002", Category: "Stack", Question: "Framework?"},
	})
	targetDir := setupTargetProject(t, nil)

	if err := execImport(t, targetDir, sourceDir, "--category", "Stack"); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 2 {
		t.Fatalf("expected 2 decisions (Stack only), got %d", len(store.Decisions))
	}

	for _, d := range store.Decisions {
		if d.Category != "Stack" {
			t.Errorf("expected category Stack, got %s", d.Category)
		}
	}
}

func TestImportKeepAnswers(t *testing.T) {
	answer := "Go"
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?", Answer: &answer, Source: "user", Reasoning: "fast"},
	})
	targetDir := setupTargetProject(t, nil)

	if err := execImport(t, targetDir, sourceDir, "--keep-answers"); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(store.Decisions))
	}

	imported := store.Decisions[0]
	if imported.Answer == nil || *imported.Answer != "Go" {
		t.Errorf("expected Answer 'Go', got %v", imported.Answer)
	}
}

func TestImportNonExistentPathErrors(t *testing.T) {
	targetDir := setupTargetProject(t, nil)
	err := execImport(t, targetDir, "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent source path")
	}
}

func TestImportUnresolvedDependenciesWarns(t *testing.T) {
	// Source has a decision that depends on an ID not being imported.
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0002", Category: "Stack", Question: "Framework?", DependsOn: []string{"STA-0001"}},
	})
	// Target does not have STA-0001 either.
	targetDir := setupTargetProject(t, nil)

	// Should not error, but the dependency will be unresolved.
	if err := execImport(t, targetDir, sourceDir); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(store.Decisions))
	}

	// The unresolved dependency should remain as-is.
	imported := store.Decisions[0]
	if len(imported.DependsOn) != 1 || imported.DependsOn[0] != "STA-0001" {
		t.Errorf("expected unresolved DependsOn [STA-0001], got %v", imported.DependsOn)
	}
}

func TestImportAllFromSource(t *testing.T) {
	sourceDir := setupSourceProject(t, []decision.Decision{
		{ID: "STA-0001", Category: "Stack", Question: "Language?"},
		{ID: "DAT-0001", Category: "Data", Question: "Database?"},
	})
	targetDir := setupTargetProject(t, nil)

	if err := execImport(t, targetDir, sourceDir); err != nil {
		t.Fatal(err)
	}

	store, err := decision.LoadStore(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(store.Decisions))
	}
}
