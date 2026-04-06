package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/defer-ai/cli/internal/config"
	"github.com/defer-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure your AI provider",
	Long:  `Run the interactive setup wizard to choose or change your AI provider. Saves to ~/.defer/config.json.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runSetupWizard()
		if err != nil {
			return err
		}
		if result.Skipped {
			fmt.Fprintln(os.Stderr, "Setup skipped.")
			return nil
		}
		return saveSetupResult(result)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// runSetupWizard launches the interactive provider picker.
func runSetupWizard() (tui.OnboardingResult, error) {
	m := tui.NewOnboardingModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return tui.OnboardingResult{Skipped: true}, err
	}
	return finalModel.(tui.OnboardingModel).Result(), nil
}

// saveSetupResult persists the wizard choices to ~/.defer/config.json.
func saveSetupResult(result tui.OnboardingResult) error {
	path := config.GlobalConfigPath()

	// Load existing global config (if any) to preserve other fields
	cfg, _ := config.LoadGlobalConfig()
	if cfg == nil {
		cfg = &config.Config{}
	}

	cfg.Provider = result.Provider
	if result.APIKey != "" {
		cfg.APIKey = result.APIKey
	}
	if result.MascotSize != "" {
		cfg.MascotSize = result.MascotSize
	}
	if result.Theme != "" {
		cfg.Theme = result.Theme
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Saved to %s\n", path)
	return nil
}

// needsOnboarding returns true if no global config exists (first run ever).
func needsOnboarding() bool {
	_, err := os.Stat(config.GlobalConfigPath())
	return os.IsNotExist(err)
}
