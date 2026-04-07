package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ModelChoice describes a model option for a provider, used by the setup wizard.
type ModelChoice struct {
	Name        string `json:"name"`        // display name (e.g. "GPT-4o")
	Value       string `json:"value"`       // exact model ID to pass to the provider
	Description string `json:"description"` // short tag (e.g. "general-purpose")
}

// BundledModels is the curated default list shipped with the binary. It's
// what the wizard shows when there's no cached refresh and no live query
// available. The list ends implicitly with a "Custom..." entry that the
// wizard appends so the user can always type a model name we don't know
// about. Order matters — the first entry is the default selection.
var BundledModels = map[string][]ModelChoice{
	"claude": {
		{Name: "Sonnet", Value: "sonnet", Description: "balanced — default"},
		{Name: "Opus", Value: "opus", Description: "deepest reasoning, slower"},
		{Name: "Haiku", Value: "haiku", Description: "fastest, cheapest"},
	},
	"openai": {
		{Name: "GPT-4o", Value: "gpt-4o", Description: "general-purpose"},
		{Name: "GPT-4o mini", Value: "gpt-4o-mini", Description: "fast, cheap"},
		{Name: "o1", Value: "o1", Description: "reasoning"},
		{Name: "o1-mini", Value: "o1-mini", Description: "fast reasoning"},
	},
	"groq": {
		{Name: "Llama 3.3 70B", Value: "llama-3.3-70b-versatile", Description: "flagship"},
		{Name: "Llama 3.1 8B", Value: "llama-3.1-8b-instant", Description: "fast"},
		{Name: "Mixtral 8x7B", Value: "mixtral-8x7b-32768", Description: "long context"},
	},
	"mistral": {
		{Name: "Mistral Large", Value: "mistral-large-latest", Description: "flagship"},
		{Name: "Mistral Small", Value: "mistral-small-latest", Description: "fast, cheap"},
		{Name: "Codestral", Value: "codestral-latest", Description: "code-specialized"},
	},
	"together": {
		{Name: "Llama 3.3 70B", Value: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Description: "flagship open-source"},
		{Name: "Qwen 2.5 72B", Value: "Qwen/Qwen2.5-72B-Instruct-Turbo", Description: "alternative flagship"},
		{Name: "Llama 3.2 3B", Value: "meta-llama/Llama-3.2-3B-Instruct-Turbo", Description: "fast, cheap"},
	},
	"openrouter": {
		{Name: "Claude Sonnet (via OR)", Value: "anthropic/claude-sonnet-4", Description: "Anthropic's Sonnet"},
		{Name: "GPT-4o (via OR)", Value: "openai/gpt-4o", Description: "OpenAI flagship"},
		{Name: "Llama 3.3 70B", Value: "meta-llama/llama-3.3-70b-instruct", Description: "open-source flagship"},
	},
	"ollama": {
		// Populated dynamically by OllamaInstalledModels(); these are
		// fallbacks if the local query fails.
		{Name: "Llama 3.1", Value: "llama3.1", Description: "common default"},
		{Name: "Mistral", Value: "mistral", Description: "alternative"},
		{Name: "Qwen 2.5", Value: "qwen2.5", Description: "code-friendly"},
	},
}

// modelsCachePath returns the path to the local model cache file.
func modelsCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".defer", "models.json")
}

// modelsCache is the on-disk format. Per-provider lists keyed by provider name.
type modelsCache struct {
	UpdatedAt string                     `json:"updatedAt"`
	Providers map[string][]ModelChoice  `json:"providers"`
}

// LoadCachedModels reads the cached model registry. Returns nil if no cache
// exists or it's unreadable.
func LoadCachedModels() *modelsCache {
	path := modelsCachePath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c modelsCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

// ModelsForProvider returns the model list for a given provider, in
// preference order:
//  1. For Ollama: dynamic query of localhost:11434 (always live)
//  2. Cached refresh (if present)
//  3. BundledModels default
func ModelsForProvider(provider string) []ModelChoice {
	if provider == "ollama" {
		if installed := OllamaInstalledModels(); len(installed) > 0 {
			return installed
		}
		// fall through to bundled if Ollama isn't running
	}

	if c := LoadCachedModels(); c != nil {
		if list, ok := c.Providers[provider]; ok && len(list) > 0 {
			return list
		}
	}

	return BundledModels[provider]
}

// OllamaInstalledModels queries the local Ollama daemon for installed models.
// Returns nil if Ollama isn't reachable.
func OllamaInstalledModels() []ModelChoice {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var raw struct {
		Models []struct {
			Name    string `json:"name"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}
	var out []ModelChoice
	for _, m := range raw.Models {
		desc := strings.TrimSpace(m.Details.ParameterSize)
		out = append(out, ModelChoice{
			Name:        m.Name,
			Value:       m.Name,
			Description: desc,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// RefreshModels rebuilds the local model cache by querying each provider's
// /v1/models endpoint where possible. Providers that need API keys are
// skipped if no key is provided. Always queries OpenRouter (public) and
// Ollama (local). Returns the count of providers refreshed.
func RefreshModels(keys map[string]string) (int, error) {
	cache := &modelsCache{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Providers: map[string][]ModelChoice{},
	}

	// Start with bundled defaults so providers we can't refresh stay populated.
	for k, v := range BundledModels {
		cache.Providers[k] = append([]ModelChoice(nil), v...)
	}

	count := 0

	// Ollama (local)
	if installed := OllamaInstalledModels(); len(installed) > 0 {
		cache.Providers["ollama"] = installed
		count++
	}

	// OpenRouter (public, no key)
	if list := refreshOpenAIStyle("openrouter", "https://openrouter.ai/api/v1/models", ""); len(list) > 0 {
		cache.Providers["openrouter"] = list
		count++
	}

	// Providers that need API keys
	keyedProviders := []struct {
		name string
		url  string
	}{
		{"openai", "https://api.openai.com/v1/models"},
		{"groq", "https://api.groq.com/openai/v1/models"},
		{"mistral", "https://api.mistral.ai/v1/models"},
		{"together", "https://api.together.xyz/v1/models"},
	}
	for _, p := range keyedProviders {
		key := keys[p.name]
		if key == "" {
			key = os.Getenv(strings.ToUpper(p.name) + "_API_KEY")
		}
		if key == "" {
			continue
		}
		if list := refreshOpenAIStyle(p.name, p.url, key); len(list) > 0 {
			cache.Providers[p.name] = list
			count++
		}
	}

	// Save cache
	path := modelsCachePath()
	if path == "" {
		return count, fmt.Errorf("no home directory available")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return count, err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return count, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return count, err
	}
	return count, nil
}

// refreshOpenAIStyle calls a provider's /v1/models endpoint (which all
// OpenAI-compat providers expose) and returns the model list.
func refreshOpenAIStyle(provider, url, apiKey string) []ModelChoice {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var raw struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}
	var out []ModelChoice
	for _, m := range raw.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		out = append(out, ModelChoice{
			Name:        name,
			Value:       m.ID,
			Description: trimDescription(m.Description),
		})
	}
	// Stable order
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out
}

func trimDescription(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		s = s[:57] + "..."
	}
	return s
}
