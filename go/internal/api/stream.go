package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/defer-ai/cli/internal/decision"
)

// EventType classifies agent loop events.
type EventType int

const (
	EventTextDelta     EventType = iota // Claude produced text
	EventToolCallStart                  // Claude wants to call a tool
	EventToolCallDone                   // Tool execution finished
	EventDecisionFound                  // An implicit decision was logged
	EventDone                           // Agent loop finished
	EventError                          // Something went wrong
)

// Event is emitted by the agent loop.
type Event struct {
	Type       EventType
	Text       string              // for TextDelta
	ToolCall   *ToolCall           // for ToolCallStart
	ToolResult *ToolResult         // for ToolCallDone
	Decision   *decision.Decision  // for DecisionFound
	Error      error               // for Error
}

// RunConfig configures an agent loop run.
type RunConfig struct {
	Client       *Client
	SystemPrompt string
	Messages     []anthropic.MessageParam
	ToolSet      ToolSet
	CWD          string
	MaxTurns     int                           // 0 = default 50
	Domain       string                        // domain name for decision tagging
	AllDecisions []decision.Decision           // for dedup and ID generation
	ApprovalFunc func(tc ToolCall) bool        // if set, called for major actions; return false to reject
}

// RunAgentLoop runs the tool-use loop until Claude stops calling tools.
// Events are sent to the channel as they occur. The channel is NOT closed by this function.
func RunAgentLoop(ctx context.Context, cfg RunConfig, events chan<- Event) {
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 50
	}

	messages := make([]anthropic.MessageParam, len(cfg.Messages))
	copy(messages, cfg.Messages)

	tools := GetTools(cfg.ToolSet)

	for turn := 0; turn < maxTurns; turn++ {
		select {
		case <-ctx.Done():
			events <- Event{Type: EventError, Error: ctx.Err()}
			return
		default:
		}

		// Build request
		params := anthropic.MessageNewParams{
			Model:     cfg.Client.ModelID(),
			MaxTokens: 8192,
			System: []anthropic.TextBlockParam{
				{Text: cfg.SystemPrompt},
			},
			Messages: messages,
		}
		if len(tools) > 0 {
			params.Tools = tools
		}

		// Call API with timeout
		apiCtx, apiCancel := context.WithTimeout(ctx, 5*time.Minute)
		resp, err := cfg.Client.Inner.Messages.New(apiCtx, params)
		apiCancel()

		if err != nil {
			events <- Event{Type: EventError, Error: fmt.Errorf("API error: %w", err)}
			return
		}

		// Process response content blocks
		var toolResults []anthropic.ContentBlockParamUnion
		hasToolUse := false

		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				events <- Event{Type: EventTextDelta, Text: block.Text}

			case "tool_use":
				hasToolUse = true
				tc := ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: json.RawMessage(block.Input),
				}

				events <- Event{Type: EventToolCallStart, ToolCall: &tc}

				// Log as a decision
				if tc.IsMajorAction() {
					today := time.Now().Format("2006-01-02")
					cat := cfg.Domain
					if cat == "" {
						cat = "Misc"
					}
					desc := tc.HumanDescription()
					d := decision.Decision{
						ID:        decision.NextID(cfg.AllDecisions, cat),
						Category:  cat,
						Question:  desc,
						Answer:    &desc,
						Implicit:  true,
						Source:    "agent",
						Date:      today,
					}
					cfg.AllDecisions = append(cfg.AllDecisions, d)
					events <- Event{Type: EventDecisionFound, Decision: &d}
				}

				// Approval check for paranoid mode
				// Approval check for paranoid mode
				if cfg.ApprovalFunc != nil && tc.IsMajorAction() {
					if !cfg.ApprovalFunc(tc) {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(tc.ID, "User rejected this action.", true))
						continue
					}
				}

				// Execute the tool
				result := ExecuteTool(ctx, tc, cfg.CWD)
				events <- Event{Type: EventToolCallDone, ToolResult: &result}

				toolResults = append(toolResults, anthropic.NewToolResultBlock(result.ToolUseID, result.Content, result.IsError))
			}
		}

		// Append assistant response to messages
		var assistantContent []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantContent = append(assistantContent, anthropic.NewTextBlock(block.Text))
			case "tool_use":
				assistantContent = append(assistantContent, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
			}
		}
		messages = append(messages, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: assistantContent,
		})

		// If no tool use, we're done
		if !hasToolUse {
			events <- Event{Type: EventDone}
			return
		}

		// Send tool results back as user message
		messages = append(messages, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleUser,
			Content: toolResults,
		})
	}

	events <- Event{Type: EventError, Error: fmt.Errorf("max turns (%d) exceeded", maxTurns)}
}

// SimpleCompletion runs a single message with no tools (for verification/extraction).
func SimpleCompletion(ctx context.Context, client *Client, systemPrompt, userMessage string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	resp, err := client.Inner.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     client.ModelID(),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: userMessage}},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text, nil
}
