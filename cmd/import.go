package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var importCategory string
var importKeepAnswers bool

var importCmd = &cobra.Command{
	Use:   "import <source-path> [@ID...]",
	Short: "Import decisions from another project",
	Long: `Import decisions from another project's .defer/decisions.json into the current project.

Examples:
  defer import /path/to/project                         # import all decisions
  defer import /path/to/project @STA-0001 @DAT-0001    # import specific decisions
  defer import /path/to/project --category Stack        # import by category
  defer import /path/to/project --keep-answers          # preserve original answers`,
	Args: cobra.MinimumNArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVar(&importCategory, "category", "", "Filter source decisions by category")
	importCmd.Flags().BoolVar(&importKeepAnswers, "keep-answers", false, "Preserve answers from source (default: import as pending)")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Collect requested IDs (strip @ prefix).
	var requestedIDs []string
	for _, a := range args[1:] {
		requestedIDs = append(requestedIDs, strings.TrimPrefix(a, "@"))
	}

	// Resolve source path to absolute.
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	// Load source store.
	sourceStore, err := decision.LoadStore(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load source decisions: %w", err)
	}
	if sourceStore == nil {
		return fmt.Errorf("no decisions found at %s", sourcePath)
	}

	// Filter source decisions.
	var toImport []decision.Decision
	for _, d := range sourceStore.Decisions {
		if len(requestedIDs) > 0 {
			if !stringInSlice(d.ID, requestedIDs) {
				continue
			}
		}
		if importCategory != "" {
			if !strings.EqualFold(d.Category, importCategory) {
				continue
			}
		}
		toImport = append(toImport, d)
	}

	if len(toImport) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No matching decisions found in source project.")
		return nil
	}

	// Load or create current project store.
	cwd, _ := os.Getwd()
	store, err := decision.LoadStore(cwd)
	if err != nil {
		return fmt.Errorf("failed to load current project decisions: %w", err)
	}
	if store == nil {
		store, err = decision.CreateStore(cwd, "(imported)")
		if err != nil {
			return fmt.Errorf("failed to create decision store: %w", err)
		}
	}

	// Build old->new ID mapping and re-ID.
	idMap := make(map[string]string)
	now := time.Now().UTC().Format(time.RFC3339)

	for i := range toImport {
		d := &toImport[i]
		oldID := d.ID
		newID := decision.NextID(store.Decisions, d.Category)
		idMap[oldID] = newID
		d.ID = newID
		d.CreatedAt = now
		d.ImportedFrom = &decision.ImportRef{
			Project:    sourcePath,
			OriginalID: oldID,
			ImportedAt: now,
		}
		if !importKeepAnswers {
			d.Answer = nil
			d.AnsweredAt = ""
			d.Source = ""
			d.OriginalSource = ""
			d.RevisionCount = 0
		}
		// Append to store so NextID sees it for subsequent iterations.
		store.Decisions = append(store.Decisions, *d)
	}

	// Remap DependsOn references.
	// We need to update the decisions we just appended (the last len(toImport) entries).
	importStart := len(store.Decisions) - len(toImport)
	existingIDs := make(map[string]bool)
	for _, d := range store.Decisions {
		existingIDs[d.ID] = true
	}

	var warnings []string
	for i := importStart; i < len(store.Decisions); i++ {
		d := &store.Decisions[i]
		if len(d.DependsOn) > 0 {
			remapped := make([]string, 0, len(d.DependsOn))
			for _, dep := range d.DependsOn {
				if newDep, ok := idMap[dep]; ok {
					remapped = append(remapped, newDep)
				} else if existingIDs[dep] {
					remapped = append(remapped, dep)
				} else {
					warnings = append(warnings, fmt.Sprintf("  %s (now %s): dependency %s not found in import set or current project", d.ImportedFrom.OriginalID, d.ID, dep))
					remapped = append(remapped, dep)
				}
			}
			d.DependsOn = remapped
		}
	}

	// Print warnings.
	for _, w := range warnings {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: unresolved dependency:")
		fmt.Fprintln(cmd.ErrOrStderr(), w)
	}

	// Save.
	if err := decision.SaveStore(cwd, store); err != nil {
		return fmt.Errorf("failed to save decisions: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Imported %d decisions from %s\n", len(toImport), sourcePath)
	return nil
}

func stringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
