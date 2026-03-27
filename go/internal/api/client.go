package api

import (
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client wraps the Anthropic SDK client.
type Client struct {
	Inner anthropic.Client
	Model string
}

// NewClient creates a client from an API key and model name.
func NewClient(model string) *Client {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	c := anthropic.NewClient(opts...)
	return &Client{
		Inner: c,
		Model: model,
	}
}

// IsConfigured returns true if an API key is available.
func IsConfigured() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

// ModelID returns the full model identifier.
func (c *Client) ModelID() anthropic.Model {
	switch c.Model {
	case "opus":
		return anthropic.ModelClaudeOpus4_6
	case "haiku":
		return anthropic.ModelClaudeHaiku4_5_20251001
	default:
		return anthropic.ModelClaudeSonnet4_6
	}
}
