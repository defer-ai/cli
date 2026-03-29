package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatInitializeRequest(t *testing.T) {
	params := initializeParams{
		ProtocolVersion: "2025-03-26",
		Capabilities:    json.RawMessage(`{}`),
		ClientInfo: clientInfo{
			Name:    "defer",
			Version: "0.1.0",
		},
	}

	msg, err := FormatRequest("initialize", params)
	if err != nil {
		t.Fatal(err)
	}

	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		t.Fatalf("cannot parse request: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want 2.0", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Errorf("id = %d, want 1", req.ID)
	}
	if req.Method != "initialize" {
		t.Errorf("method = %q, want initialize", req.Method)
	}

	// Verify params contain expected fields
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		t.Fatal(err)
	}
	paramsStr := string(paramsBytes)
	if !jsonContains(paramsStr, "protocolVersion") {
		t.Error("missing protocolVersion in params")
	}
	if !jsonContains(paramsStr, "2025-03-26") {
		t.Error("missing protocol version value")
	}
	if !jsonContains(paramsStr, "defer") {
		t.Error("missing client name")
	}
}

func TestFormatToolsListRequest(t *testing.T) {
	msg, err := FormatRequest("tools/list", struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		t.Fatalf("cannot parse request: %v", err)
	}

	if req.Method != "tools/list" {
		t.Errorf("method = %q, want tools/list", req.Method)
	}
}

func TestFormatCallToolRequest(t *testing.T) {
	params := callToolParams{
		Name:      "sqlite_query",
		Arguments: json.RawMessage(`{"query": "SELECT * FROM users"}`),
	}

	msg, err := FormatRequest("tools/call", params)
	if err != nil {
		t.Fatal(err)
	}

	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		t.Fatalf("cannot parse request: %v", err)
	}

	if req.Method != "tools/call" {
		t.Errorf("method = %q, want tools/call", req.Method)
	}

	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		t.Fatal(err)
	}
	paramsStr := string(paramsBytes)
	if !jsonContains(paramsStr, "sqlite_query") {
		t.Error("missing tool name in params")
	}
	if !jsonContains(paramsStr, "SELECT * FROM users") {
		t.Error("missing query in arguments")
	}
}

func TestParseToolsListResponse(t *testing.T) {
	response := `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"query","description":"Run SQL query","inputSchema":{"type":"object"}}]}}`

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result toolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(result.Tools))
	}
	if result.Tools[0].Name != "query" {
		t.Errorf("tool name = %q, want query", result.Tools[0].Name)
	}
	if result.Tools[0].Description != "Run SQL query" {
		t.Errorf("tool description = %q", result.Tools[0].Description)
	}
}

func TestParseCallToolResponse(t *testing.T) {
	response := `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"3 rows found"}],"isError":false}}`

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		t.Fatal(err)
	}

	var result callToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Error("expected isError = false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content = %d, want 1", len(result.Content))
	}
	if result.Content[0].Text != "3 rows found" {
		t.Errorf("text = %q, want %q", result.Content[0].Text, "3 rows found")
	}
}

func TestParseErrorResponse(t *testing.T) {
	response := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
}

func TestToolSerialization(t *testing.T) {
	tool := Tool{
		Name:        "read_file",
		Description: "Read a file from disk",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Tool
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Name != "read_file" {
		t.Errorf("name = %q, want read_file", parsed.Name)
	}
	if parsed.Description != "Read a file from disk" {
		t.Errorf("description = %q", parsed.Description)
	}
}

func TestConfigSerialization(t *testing.T) {
	cfg := Config{
		Servers: map[string]ServerConfig{
			"sqlite": {
				Command: "mcp-server-sqlite",
				Args:    []string{"--db", "app.db"},
			},
			"github": {
				Command: "mcp-server-github",
				Args:    []string{},
				Env:     map[string]string{"GITHUB_TOKEN": "ghp_xxx"},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if len(parsed.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(parsed.Servers))
	}

	sqlite := parsed.Servers["sqlite"]
	if sqlite.Command != "mcp-server-sqlite" {
		t.Errorf("sqlite command = %q", sqlite.Command)
	}
	if len(sqlite.Args) != 2 || sqlite.Args[0] != "--db" || sqlite.Args[1] != "app.db" {
		t.Errorf("sqlite args = %v", sqlite.Args)
	}

	github := parsed.Servers["github"]
	if github.Env["GITHUB_TOKEN"] != "ghp_xxx" {
		t.Errorf("github env = %v", github.Env)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil config when no config file exists")
	}
}

func TestLoadConfigFromProjectDir(t *testing.T) {
	dir := t.TempDir()

	// Create .defer/mcp.json
	deferDir := filepath.Join(dir, ".defer")
	if err := os.MkdirAll(deferDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgJSON := `{"servers":{"test":{"command":"echo","args":["hello"]}}}`
	if err := os.WriteFile(filepath.Join(deferDir, "mcp.json"), []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if len(cfg.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(cfg.Servers))
	}
	if cfg.Servers["test"].Command != "echo" {
		t.Errorf("command = %q, want echo", cfg.Servers["test"].Command)
	}
}

// helpers

func jsonContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
