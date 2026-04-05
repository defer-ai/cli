package api

import (
	"context"
	"os"
)

// Provider is the interface any AI provider must implement.
type Provider interface {
	RunCompletion(ctx context.Context, systemPrompt, userPrompt string, events chan<- Event)
	GetModel() string
	ResetSession()
}

// ResolveProvider creates the right provider based on flags and environment.
// Priority: --provider flag > env API keys > Claude Code subprocess
func ResolveProvider(providerName, apiKey, model string) (Provider, error) {
	// Explicit provider flag
	if providerName != "" && providerName != "claude" {
		if apiKey == "" {
			apiKey = envKey(providerName)
		}
		return NewOpenAIProvider(providerName, apiKey, mapModel(model, providerName)), nil
	}

	// Check for API keys in environment
	if k := envKey("openai"); k != "" && providerName == "" {
		return NewOpenAIProvider("openai", k, mapModel(model, "openai")), nil
	}
	if k := envKey("groq"); k != "" && providerName == "" {
		return NewOpenAIProvider("groq", k, mapModel(model, "groq")), nil
	}

	// Fall back to Claude Code subprocess
	if IsClaudeInstalled() {
		return NewClaudeCodeProvider(model), nil
	}

	return nil, &NoProviderError{}
}

type NoProviderError struct{}

func (e *NoProviderError) Error() string {
	return "no AI provider available. Set OPENAI_API_KEY, GROQ_API_KEY, or install Claude Code"
}

func envKey(provider string) string {
	switch provider {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "groq":
		return os.Getenv("GROQ_API_KEY")
	case "mistral":
		return os.Getenv("MISTRAL_API_KEY")
	case "together":
		return os.Getenv("TOGETHER_API_KEY")
	case "deepinfra":
		return os.Getenv("DEEPINFRA_API_KEY")
	case "cerebras":
		return os.Getenv("CEREBRAS_API_KEY")
	case "perplexity":
		return os.Getenv("PERPLEXITY_API_KEY")
	case "openrouter":
		return os.Getenv("OPENROUTER_API_KEY")
	default:
		return ""
	}
}

// mapModel converts defer's model names to provider-specific model IDs
func mapModel(model, provider string) string {
	switch provider {
	case "openai":
		switch model {
		case "sonnet", "opus":
			return "gpt-4o"
		case "haiku":
			return "gpt-4o-mini"
		default:
			return model
		}
	case "groq":
		switch model {
		case "sonnet", "opus":
			return "llama-3.3-70b-versatile"
		case "haiku":
			return "llama-3.1-8b-instant"
		default:
			return model
		}
	default:
		return model
	}
}
