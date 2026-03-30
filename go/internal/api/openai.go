package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider works with any OpenAI-compatible API.
// Supports: OpenAI, Groq, Mistral, Together, Ollama, DeepInfra, Cerebras, Perplexity, etc.
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Well-known providers with their base URLs.
var KnownProviders = map[string]string{
	"openai":    "https://api.openai.com/v1",
	"groq":      "https://api.groq.com/openai/v1",
	"mistral":   "https://api.mistral.ai/v1",
	"together":  "https://api.together.xyz/v1",
	"deepinfra": "https://api.deepinfra.com/v1/openai",
	"cerebras":  "https://api.cerebras.ai/v1",
	"perplexity": "https://api.perplexity.ai",
	"ollama":    "http://localhost:11434/v1",
	"openrouter": "https://openrouter.ai/api/v1",
}

type oaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaiRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	Stream      bool         `json:"stream"`
}

type oaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// NewOpenAIProvider creates a provider for any OpenAI-compatible API.
// providerName can be a known name ("openai", "groq", etc.) or a full URL.
func NewOpenAIProvider(providerName, apiKey, model string) *OpenAIProvider {
	baseURL := providerName
	if url, ok := KnownProviders[providerName]; ok {
		baseURL = url
	}
	return &OpenAIProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	}
}

// RunCompletion sends a prompt and emits events.
func (p *OpenAIProvider) RunCompletion(ctx context.Context, systemPrompt, userPrompt string, events chan<- Event) {
	req := oaiRequest{
		Model: p.Model,
		Messages: []oaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   8192,
		Temperature: 0.3,
		Stream:      false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		events <- Event{Type: EventError, Error: fmt.Errorf("marshal request: %w", err)}
		return
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		events <- Event{Type: EventError, Error: err}
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		events <- Event{Type: EventError, Error: err}
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		events <- Event{Type: EventError, Error: fmt.Errorf("read response: %w", err)}
		return
	}
	if resp.StatusCode != 200 {
		events <- Event{Type: EventError, Error: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))}
		return
	}

	var result oaiResponse
	if err := json.Unmarshal(data, &result); err != nil {
		events <- Event{Type: EventError, Error: fmt.Errorf("JSON parse error: %w", err)}
		return
	}

	if result.Error != nil {
		events <- Event{Type: EventError, Error: fmt.Errorf("API error: %s", result.Error.Message)}
		return
	}

	if len(result.Choices) > 0 && result.Choices[0].Message.Content != "" {
		events <- Event{Type: EventTextDelta, Text: result.Choices[0].Message.Content}
	}

	events <- Event{Type: EventDone}
}

// GetModel returns the model name.
func (p *OpenAIProvider) GetModel() string {
	return p.Model
}

// ResetSession is a no-op for stateless HTTP API.
func (p *OpenAIProvider) ResetSession() {}
