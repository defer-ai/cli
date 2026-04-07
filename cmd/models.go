package cmd

import (
	"fmt"
	"os"

	"github.com/defer-ai/cli/internal/api"
	"github.com/defer-ai/cli/internal/config"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Inspect available models",
	Long:  `Inspect or refresh the local model registry used by the setup wizard.`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list [provider]",
	Short: "List available models for a provider",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providers := []string{"claude", "openai", "groq", "mistral", "together", "openrouter", "ollama"}
		if len(args) == 1 {
			providers = []string{args[0]}
		}
		for _, p := range providers {
			fmt.Printf("\n%s:\n", p)
			list := api.ModelsForProvider(p)
			if len(list) == 0 {
				fmt.Println("  (none)")
				continue
			}
			for _, m := range list {
				fmt.Printf("  %-32s %s\n", m.Value, m.Description)
			}
		}
		return nil
	},
}

var modelsRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the local model cache from provider APIs",
	Long: `Hits each provider's /v1/models endpoint to rebuild the local model cache
at ~/.defer/models.json.

Providers that need API keys are queried only if a key is configured (via
~/.defer/config.json) or set in the environment (e.g. OPENAI_API_KEY).
OpenRouter is queried via its public endpoint. Ollama is queried locally.

Providers we cannot reach keep their bundled defaults from the binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Collect API keys from config + environment
		keys := map[string]string{}
		cfg, _ := config.LoadGlobalConfig()
		if cfg != nil && cfg.APIKey != "" && cfg.Provider != "" {
			keys[cfg.Provider] = cfg.APIKey
		}
		// Each provider can also be hit via its own env var
		for _, name := range []string{"openai", "groq", "mistral", "together", "openrouter"} {
			env := envKeyName(name)
			if v := os.Getenv(env); v != "" {
				keys[name] = v
			}
		}

		fmt.Fprintln(os.Stderr, "Refreshing model cache...")
		count, err := api.RefreshModels(keys)
		if err != nil {
			return fmt.Errorf("refresh failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Refreshed %d providers. Cache saved to ~/.defer/models.json\n", count)
		return nil
	},
}

func envKeyName(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	case "together":
		return "TOGETHER_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	}
	return ""
}

func init() {
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsRefreshCmd)
	rootCmd.AddCommand(modelsCmd)
}
