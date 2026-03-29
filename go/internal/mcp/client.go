package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client connects to an MCP server over stdio using JSON-RPC 2.0.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	tools  []Tool
	mu     sync.Mutex
	nextID int64
}

// Tool represents an MCP tool exposed by a server.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// initializeParams contains the params for the initialize request.
type initializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities"`
	ClientInfo      clientInfo      `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// toolsListResult is the result of tools/list.
type toolsListResult struct {
	Tools []Tool `json:"tools"`
}

// callToolParams is the params for tools/call.
type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// callToolResult is the result of tools/call.
type callToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

// NewClient creates an MCP client that communicates with a subprocess over stdio.
func NewClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("start command %q: %w", command, err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
	}, nil
}

// Initialize sends the initialize request to the MCP server.
func (c *Client) Initialize() error {
	params := initializeParams{
		ProtocolVersion: "2025-03-26",
		Capabilities:    json.RawMessage(`{}`),
		ClientInfo: clientInfo{
			Name:    "defer",
			Version: "0.1.0",
		},
	}

	_, err := c.call("initialize", params)
	return err
}

// ListTools retrieves the available tools from the MCP server.
func (c *Client) ListTools() ([]Tool, error) {
	result, err := c.call("tools/list", struct{}{})
	if err != nil {
		return nil, err
	}

	var tlr toolsListResult
	if err := json.Unmarshal(result, &tlr); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}

	c.tools = tlr.Tools
	return tlr.Tools, nil
}

// CallTool invokes a tool on the MCP server and returns the text result.
func (c *Client) CallTool(name string, input json.RawMessage) (string, error) {
	params := callToolParams{
		Name:      name,
		Arguments: input,
	}

	result, err := c.call("tools/call", params)
	if err != nil {
		return "", err
	}

	var ctr callToolResult
	if err := json.Unmarshal(result, &ctr); err != nil {
		return "", fmt.Errorf("parse tools/call result: %w", err)
	}

	var texts []string
	for _, c := range ctr.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}

	text := ""
	if len(texts) > 0 {
		text = texts[0]
		for _, t := range texts[1:] {
			text += "\n" + t
		}
	}

	if ctr.IsError {
		return text, fmt.Errorf("tool error: %s", text)
	}

	return text, nil
}

// Close terminates the MCP server subprocess.
func (c *Client) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

// Tools returns the cached tool list from the last ListTools call.
func (c *Client) Tools() []Tool {
	return c.tools
}

// call sends a JSON-RPC request and reads the response.
func (c *Client) call(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := atomic.AddInt64(&c.nextID, 1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Write request followed by newline
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response line
	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("server closed connection")
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// FormatRequest creates a JSON-RPC request string for testing/debugging.
func FormatRequest(method string, params interface{}) (string, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
