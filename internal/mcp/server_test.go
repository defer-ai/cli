package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/defer-ai/cli/internal/decision"
)

func setupTestStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	deferDir := filepath.Join(dir, ".defer")
	os.MkdirAll(deferDir, 0o755)

	answer := "Go"
	store := &decision.DecisionStore{
		Task: "test project",
		Decisions: []decision.Decision{
			{
				ID: "STA-0001", Category: "Stack", Question: "Backend language?",
				Answer: &answer, Source: "auto", Impact: 9,
				Options: []decision.DecisionOption{{Key: "A", Label: "Go"}, {Key: "B", Label: "Rust"}},
			},
			{
				ID: "DAT-0001", Category: "Data", Question: "Database?",
				Impact: 8, DependsOn: []string{"STA-0001"},
				Options: []decision.DecisionOption{{Key: "A", Label: "PostgreSQL"}, {Key: "B", Label: "SQLite"}},
			},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}

	data, _ := json.MarshalIndent(store, "", "  ")
	os.WriteFile(filepath.Join(deferDir, "decisions.json"), data, 0o644)
	return dir
}

func runServerRequest(t *testing.T, cwd string, requests ...string) []jsonRPCResponse {
	t.Helper()
	input := strings.Join(requests, "\n") + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := NewServerWithIO(cwd, "test", reader, &output)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	server.Run(ctx)

	var responses []jsonRPCResponse
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if line == "" {
			continue
		}
		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("parse response: %v\nline: %s", err, line)
		}
		responses = append(responses, resp)
	}
	return responses
}

func makeReq(id int64, method string, params interface{}) string {
	req := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	data, _ := json.Marshal(req)
	return string(data)
}

func TestServerInitialize(t *testing.T) {
	cwd := t.TempDir()
	resps := runServerRequest(t, cwd, makeReq(1, "initialize", map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "1.0"},
	}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if resps[0].Error != nil {
		t.Fatalf("unexpected error: %s", resps[0].Error.Message)
	}

	var result map[string]interface{}
	json.Unmarshal(resps[0].Result, &result)
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("unexpected protocol version: %v", result["protocolVersion"])
	}
}

func TestServerToolsList(t *testing.T) {
	cwd := t.TempDir()
	resps := runServerRequest(t, cwd, makeReq(1, "tools/list", struct{}{}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}

	var result toolsListResult
	json.Unmarshal(resps[0].Result, &result)
	if len(result.Tools) != 8 {
		t.Errorf("expected 8 tools, got %d", len(result.Tools))
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{
		"read_decisions", "list_pending", "confirm_decision",
		"update_decision", "get_session_state", "get_decision_tree",
		"register_decision", "write_file",
	} {
		if !names[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestServerReadDecisions(t *testing.T) {
	cwd := setupTestStore(t)
	resps := runServerRequest(t, cwd, makeReq(1, "tools/call", map[string]interface{}{
		"name":      "read_decisions",
		"arguments": map[string]interface{}{"status": "pending"},
	}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}

	var result callToolResult
	json.Unmarshal(resps[0].Result, &result)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var decs []decision.Decision
	json.Unmarshal([]byte(result.Content[0].Text), &decs)
	if len(decs) != 1 {
		t.Errorf("expected 1 pending decision, got %d", len(decs))
	}
	if len(decs) > 0 && decs[0].ID != "DAT-0001" {
		t.Errorf("expected DAT-0001, got %s", decs[0].ID)
	}
}

func TestServerConfirmDecision(t *testing.T) {
	cwd := setupTestStore(t)
	resps := runServerRequest(t, cwd, makeReq(1, "tools/call", map[string]interface{}{
		"name": "confirm_decision",
		"arguments": map[string]interface{}{
			"id":     "DAT-0001",
			"answer": "PostgreSQL",
		},
	}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}

	var result callToolResult
	json.Unmarshal(resps[0].Result, &result)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	// Verify persisted
	store, _ := decision.LoadStore(cwd)
	for _, d := range store.Decisions {
		if d.ID == "DAT-0001" {
			if d.IsPending() {
				t.Error("DAT-0001 should be answered")
			}
			if d.StrAnswer() != "PostgreSQL" {
				t.Errorf("expected PostgreSQL, got %s", d.StrAnswer())
			}
		}
	}
}

func TestServerGetSessionState(t *testing.T) {
	cwd := setupTestStore(t)
	resps := runServerRequest(t, cwd, makeReq(1, "tools/call", map[string]interface{}{
		"name":      "get_session_state",
		"arguments": map[string]interface{}{},
	}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}

	var result callToolResult
	json.Unmarshal(resps[0].Result, &result)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var state map[string]interface{}
	json.Unmarshal([]byte(result.Content[0].Text), &state)
	if state["task"] != "test project" {
		t.Errorf("unexpected task: %v", state["task"])
	}
	if state["total"].(float64) != 2 {
		t.Errorf("expected 2 total, got %v", state["total"])
	}
}

func TestServerNoSession(t *testing.T) {
	cwd := t.TempDir()
	resps := runServerRequest(t, cwd, makeReq(1, "tools/call", map[string]interface{}{
		"name":      "read_decisions",
		"arguments": map[string]interface{}{},
	}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}

	var result callToolResult
	json.Unmarshal(resps[0].Result, &result)
	if !result.IsError {
		t.Error("expected error for missing session")
	}
	if !strings.Contains(result.Content[0].Text, "No defer session") {
		t.Errorf("unexpected error: %s", result.Content[0].Text)
	}
}

func TestServerMethodNotFound(t *testing.T) {
	cwd := t.TempDir()
	resps := runServerRequest(t, cwd, makeReq(1, "nonexistent/method", struct{}{}))

	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if resps[0].Error == nil {
		t.Fatal("expected error")
	}
	if resps[0].Error.Code != -32601 {
		t.Errorf("expected -32601, got %d", resps[0].Error.Code)
	}
}

func TestServerMultipleRequests(t *testing.T) {
	cwd := setupTestStore(t)
	resps := runServerRequest(t, cwd,
		makeReq(1, "initialize", map[string]interface{}{"protocolVersion": "2025-03-26"}),
		fmt.Sprintf(`{"jsonrpc":"2.0","method":"initialized"}`), // notification, no ID
		makeReq(2, "tools/list", struct{}{}),
		makeReq(3, "tools/call", map[string]interface{}{
			"name":      "get_session_state",
			"arguments": map[string]interface{}{},
		}),
	)

	// Should get 3 responses (notification doesn't get one)
	if len(resps) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(resps))
	}
	if resps[0].ID != 1 {
		t.Errorf("first response ID = %d, want 1", resps[0].ID)
	}
	if resps[1].ID != 2 {
		t.Errorf("second response ID = %d, want 2", resps[1].ID)
	}
}
