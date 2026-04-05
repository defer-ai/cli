package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/defer-ai/cli/internal/templates"
	"github.com/spf13/cobra"
)

var presetFlag string

var initCmd = &cobra.Command{
	Use:   "init [target]",
	Short: "Scaffold Defer config files for your AI tool",
	Long: func() string {
		var b strings.Builder
		b.WriteString("Scaffold Defer config files for your AI tool.\n\n")
		b.WriteString("Available targets:\n")
		for _, t := range templates.TargetList() {
			tmpl := templates.Templates[t]
			b.WriteString(fmt.Sprintf("  %-14s %s → %s\n", t, tmpl.Description, tmpl.Filename))
		}
		b.WriteString("\nExample:\n  defer init claude-code\n  defer init cursor\n  defer init copilot\n  defer init codex\n  defer init universal\n")
		b.WriteString("\nPresets (--preset <name>):\n")
		for _, p := range templates.DefaultPresets() {
			b.WriteString(fmt.Sprintf("  %-14s %s\n", p.Name, p.Description))
		}
		return b.String()
	}(),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := templates.TargetClaudeCode
		if len(args) > 0 {
			target = templates.Target(args[0])
		}

		tmpl, ok := templates.Templates[target]
		if !ok {
			var targets []string
			for _, t := range templates.TargetList() {
				targets = append(targets, string(t))
			}
			return fmt.Errorf("unknown target: %s (available: %s)", target, strings.Join(targets, ", "))
		}

		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, tmpl.Filename)

		// Create parent directory if needed (e.g. .github/ for copilot)
		dir := filepath.Dir(path)
		if dir != cwd {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}

		if err := os.WriteFile(path, []byte(tmpl.Content), 0o644); err != nil {
			return err
		}
		fmt.Printf("Created %s\n", tmpl.Filename)

		// Handle --preset flag
		if presetFlag != "" {
			return initWithPreset(cwd, presetFlag)
		}

		if !decision.StoreExists(cwd) {
			if _, err := decision.CreateStore(cwd, "(not started)"); err != nil {
				return err
			}
			fmt.Println("Created .defer/decisions.json")
		}

		fmt.Println("\nDefer mode is active. The AI will decompose tasks into decisions before coding.")
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&presetFlag, "preset", "", "Initialize with a preset (rest-api, cli-tool, web-app, or custom)")
}

// initWithPreset loads a preset by name and creates a decision store populated
// with its decisions.
func initWithPreset(cwd, name string) error {
	presets := templates.DiscoverPresets(cwd)

	var matched *templates.Preset
	for i := range presets {
		if presets[i].Name == name {
			matched = &presets[i]
			break
		}
	}

	if matched == nil {
		var available []string
		for _, p := range presets {
			available = append(available, p.Name)
		}
		return fmt.Errorf("unknown preset: %s (available: %s)", name, strings.Join(available, ", "))
	}

	// Convert PresetDecisions to decision.Decision structs
	now := time.Now().UTC().Format(time.RFC3339)
	var decisions []decision.Decision
	for _, pd := range matched.Decisions {
		var opts []decision.DecisionOption
		for _, o := range pd.Options {
			opts = append(opts, decision.DecisionOption{
				Key:   o.Key,
				Label: o.Label,
			})
		}
		id := decision.NextID(decisions, pd.Category)
		d := decision.Decision{
			ID:        id,
			Category:  pd.Category,
			Question:  pd.Question,
			Options:   opts,
			Context:   pd.Context,
			Impact:    pd.Impact,
			DependsOn: pd.DependsOn,
			Date:      now,
			CreatedAt: now,
		}
		decisions = append(decisions, d)
	}

	store := &decision.DecisionStore{
		Task:      fmt.Sprintf("preset:%s", matched.Name),
		Decisions: decisions,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := decision.SaveStore(cwd, store); err != nil {
		return err
	}

	fmt.Printf("Initialized with %d decisions from preset: %s\n", len(decisions), matched.Name)
	return nil
}
