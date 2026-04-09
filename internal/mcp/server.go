package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/defer-ai/cli/internal/decision"
)

// Server implements an MCP server over stdio for the defer decision store.
type Server struct {
	cwd     string
	version string
	reader  *bufio.Scanner
	writer  io.Writer
}

// NewServer creates an MCP server that reads from stdin and writes to stdout.
func NewServer(cwd, version string) *Server {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	return &Server{
		cwd:     cwd,
		version: version,
		reader:  scanner,
		writer:  os.Stdout,
	}
}

// NewServerWithIO creates a server with custom I/O (for testing).
func NewServerWithIO(cwd, version string, reader io.Reader, writer io.Writer) *Server {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	return &Server{
		cwd:     cwd,
		version: version,
		reader:  scanner,
		writer:  writer,
	}
}

// Run starts the server loop, reading JSON-RPC requests and dispatching responses.
func (s *Server) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !s.reader.Scan() {
			if err := s.reader.Err(); err != nil {
				return fmt.Errorf("read error: %w", err)
			}
			return nil // EOF
		}

		line := s.reader.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try to parse as a request
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(0, -32700, "Parse error")
			continue
		}

		// Notifications (no ID) — handle silently
		if req.ID == 0 && req.Method == "initialized" {
			continue
		}
		if req.ID == 0 && req.Method == "notifications/initialized" {
			continue
		}

		s.dispatch(req)
	}
}

func (s *Server) dispatch(req jsonRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.writeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req jsonRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
		"serverInfo": map[string]interface{}{
			"name":    "defer",
			"version": s.version,
		},
	}
	s.writeResult(req.ID, result)
}

func (s *Server) handleToolsList(req jsonRPCRequest) {
	result := toolsListResult{Tools: s.toolDefinitions()}
	s.writeResult(req.ID, result)
}

func (s *Server) handleToolsCall(req jsonRPCRequest) {
	raw, _ := json.Marshal(req.Params)
	var params callToolParams
	if err := json.Unmarshal(raw, &params); err != nil {
		s.writeError(req.ID, -32602, "Invalid params")
		return
	}

	var result callToolResult

	switch params.Name {
	case "read_decisions":
		result = s.toolReadDecisions(params.Arguments)
	case "list_pending":
		result = s.toolListPending(params.Arguments)
	case "confirm_decision":
		result = s.toolConfirmDecision(params.Arguments)
	case "update_decision":
		result = s.toolUpdateDecision(params.Arguments)
	case "get_session_state":
		result = s.toolGetSessionState(params.Arguments)
	case "get_decision_tree":
		result = s.toolGetDecisionTree(params.Arguments)
	case "register_decision":
		result = s.toolRegisterDecision(params.Arguments)
	case "write_file":
		result = s.toolWriteFile(params.Arguments)
	default:
		s.writeError(req.ID, -32602, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}

	s.writeResult(req.ID, result)
}

// --- Tool implementations ---

func (s *Server) toolReadDecisions(args json.RawMessage) callToolResult {
	store, err := decision.LoadStore(s.cwd)
	if err != nil || store == nil {
		return errResult("No defer session found. Run 'defer' or 'defer init' first.")
	}

	var params struct {
		Status   string `json:"status"`
		Category string `json:"category"`
		Feature  string `json:"feature"`
		Source   string `json:"source"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}

	filtered := store.Decisions
	if params.Status != "" && params.Status != "all" {
		var out []decision.Decision
		for _, d := range filtered {
			switch params.Status {
			case "pending":
				if d.IsPending() {
					out = append(out, d)
				}
			case "answered":
				if !d.IsPending() {
					out = append(out, d)
				}
			case "delegated":
				if d.Delegated {
					out = append(out, d)
				}
			}
		}
		filtered = out
	}
	if params.Category != "" {
		cat := strings.ToLower(params.Category)
		var out []decision.Decision
		for _, d := range filtered {
			if strings.Contains(strings.ToLower(d.Category), cat) {
				out = append(out, d)
			}
		}
		filtered = out
	}
	if params.Feature != "" {
		feat := strings.ToLower(params.Feature)
		var out []decision.Decision
		for _, d := range filtered {
			for _, f := range d.Features {
				if strings.Contains(strings.ToLower(f), feat) {
					out = append(out, d)
					break
				}
			}
		}
		filtered = out
	}
	if params.Source != "" {
		var out []decision.Decision
		for _, d := range filtered {
			if d.Source == params.Source {
				out = append(out, d)
			}
		}
		filtered = out
	}

	data, _ := json.MarshalIndent(filtered, "", "  ")
	return textResult(string(data))
}

func (s *Server) toolListPending(args json.RawMessage) callToolResult {
	store, err := decision.LoadStore(s.cwd)
	if err != nil || store == nil {
		return errResult("No defer session found.")
	}

	var params struct {
		Category        string `json:"category"`
		IncludeDelegated bool   `json:"include_delegated"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}

	var pending []decision.Decision
	for _, d := range store.Decisions {
		if !d.IsPending() {
			continue
		}
		if d.Delegated && !params.IncludeDelegated {
			continue
		}
		if params.Category != "" && !strings.Contains(strings.ToLower(d.Category), strings.ToLower(params.Category)) {
			continue
		}
		pending = append(pending, d)
	}

	// Sort by impact descending
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Impact > pending[j].Impact
	})

	result := map[string]interface{}{
		"count":     len(pending),
		"decisions": pending,
		"task":      store.Task,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func (s *Server) toolConfirmDecision(args json.RawMessage) callToolResult {
	var params struct {
		ID        string `json:"id"`
		Answer    string `json:"answer"`
		Reasoning string `json:"reasoning"`
		Source    string `json:"source"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}
	if params.ID == "" || params.Answer == "" {
		return errResult("'id' and 'answer' are required")
	}
	if params.Source == "" {
		params.Source = "user"
	}

	var result callToolResult
	err := decision.WithStoreLock(s.cwd, func() error {
		store, err := decision.LoadStore(s.cwd)
		if err != nil || store == nil {
			result = errResult("No defer session found.")
			return nil
		}

		found := false
		var invalidated []string
		for i := range store.Decisions {
			if store.Decisions[i].ID == params.ID {
				found = true
				store.Decisions[i].SetAnswer(params.Answer, params.Source)
				if params.Reasoning != "" {
					store.Decisions[i].Reasoning = params.Reasoning
				}

				// Cascade invalidation
				deps := decision.FindTransitiveDependents(params.ID, store.Decisions)
				for _, dep := range deps {
					for j := range store.Decisions {
						if store.Decisions[j].ID == dep.ID {
							decision.InvalidateDependent(&store.Decisions[j])
							invalidated = append(invalidated, dep.ID)
						}
					}
				}
				break
			}
		}

		if !found {
			result = errResult(fmt.Sprintf("Decision %s not found", params.ID))
			return nil
		}

		if err := decision.SaveStore(s.cwd, store); err != nil {
			result = errResult(fmt.Sprintf("Save failed: %v", err))
			return nil
		}

		out := map[string]interface{}{
			"decision":    store.Decisions,
			"invalidated": invalidated,
		}
		// Find the updated decision for output
		for _, d := range store.Decisions {
			if d.ID == params.ID {
				out["decision"] = d
				break
			}
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		result = textResult(string(data))
		return nil
	})
	if err != nil {
		return errResult(fmt.Sprintf("Lock error: %v", err))
	}
	return result
}

func (s *Server) toolUpdateDecision(args json.RawMessage) callToolResult {
	var params struct {
		ID        string                   `json:"id"`
		Question  string                   `json:"question"`
		Context   string                   `json:"context"`
		Category  string                   `json:"category"`
		Options   []decision.DecisionOption `json:"options"`
		Features  []string                 `json:"features"`
		Impact    *int                     `json:"impact"`
		DependsOn []string                 `json:"dependsOn"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}
	if params.ID == "" {
		return errResult("'id' is required")
	}

	var result callToolResult
	err := decision.WithStoreLock(s.cwd, func() error {
		store, err := decision.LoadStore(s.cwd)
		if err != nil || store == nil {
			result = errResult("No defer session found.")
			return nil
		}

		found := false
		for i := range store.Decisions {
			if store.Decisions[i].ID == params.ID {
				found = true
				if params.Question != "" {
					store.Decisions[i].Question = params.Question
				}
				if params.Context != "" {
					store.Decisions[i].Context = params.Context
				}
				if params.Category != "" {
					store.Decisions[i].Category = params.Category
				}
				if params.Options != nil {
					store.Decisions[i].Options = params.Options
				}
				if params.Features != nil {
					store.Decisions[i].Features = params.Features
				}
				if params.Impact != nil {
					store.Decisions[i].Impact = *params.Impact
				}
				if params.DependsOn != nil {
					store.Decisions[i].DependsOn = params.DependsOn
				}

				if err := decision.SaveStore(s.cwd, store); err != nil {
					result = errResult(fmt.Sprintf("Save failed: %v", err))
					return nil
				}

				data, _ := json.MarshalIndent(store.Decisions[i], "", "  ")
				result = textResult(string(data))
				return nil
			}
		}

		if !found {
			result = errResult(fmt.Sprintf("Decision %s not found", params.ID))
		}
		return nil
	})
	if err != nil {
		return errResult(fmt.Sprintf("Lock error: %v", err))
	}
	return result
}

func (s *Server) toolGetSessionState(args json.RawMessage) callToolResult {
	store, err := decision.LoadStore(s.cwd)
	if err != nil || store == nil {
		return errResult("No defer session found.")
	}

	answered, pending, delegated := 0, 0, 0
	cats := map[string]int{}
	for _, d := range store.Decisions {
		cats[d.Category]++
		if d.IsPending() {
			pending++
		} else {
			answered++
		}
		if d.Delegated {
			delegated++
		}
	}

	total := len(store.Decisions)
	pct := 0.0
	if total > 0 {
		pct = float64(answered) / float64(total) * 100
	}

	features := make([]map[string]interface{}, 0)
	for _, f := range store.Features {
		count := 0
		for _, d := range store.Decisions {
			for _, feat := range d.Features {
				if feat == f.Name || feat == f.ID {
					count++
					break
				}
			}
		}
		features = append(features, map[string]interface{}{
			"id": f.ID, "name": f.Name, "decision_count": count,
		})
	}

	state := map[string]interface{}{
		"task":         store.Task,
		"total":        total,
		"answered":     answered,
		"pending":      pending,
		"delegated":    delegated,
		"progress_pct": pct,
		"categories":   cats,
		"features":     features,
		"created_at":   store.CreatedAt,
		"updated_at":   store.UpdatedAt,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	return textResult(string(data))
}

type treeNode struct {
	Decision  decision.Decision `json:"decision"`
	Status    string            `json:"status"`
	BlockedBy []string          `json:"blocked_by,omitempty"`
	Children  []treeNode        `json:"children"`
}

func (s *Server) toolGetDecisionTree(args json.RawMessage) callToolResult {
	store, err := decision.LoadStore(s.cwd)
	if err != nil || store == nil {
		return errResult("No defer session found.")
	}

	var params struct {
		RootID  string `json:"root_id"`
		GroupBy string `json:"group_by"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}
	if params.GroupBy == "" {
		params.GroupBy = "dependencies"
	}

	decs := store.Decisions
	byID := map[string]*decision.Decision{}
	for i := range decs {
		byID[decs[i].ID] = &decs[i]
	}

	// Build tree by dependencies
	isChild := map[string]bool{}
	for _, d := range decs {
		for _, dep := range d.DependsOn {
			isChild[d.ID] = true
			_ = dep
		}
	}

	var buildNode func(d decision.Decision) treeNode
	buildNode = func(d decision.Decision) treeNode {
		status := "answered"
		if d.IsPending() {
			status = "pending"
		}
		var blocked []string
		for _, dep := range d.DependsOn {
			if dd, ok := byID[dep]; ok && dd.IsPending() {
				blocked = append(blocked, dep)
			}
		}
		children := []treeNode{}
		for _, dep := range decision.FindDependents(d.ID, decs) {
			children = append(children, buildNode(dep))
		}
		return treeNode{
			Decision:  d,
			Status:    status,
			BlockedBy: blocked,
			Children:  children,
		}
	}

	var nodes []treeNode
	if params.RootID != "" {
		if d, ok := byID[params.RootID]; ok {
			nodes = append(nodes, buildNode(*d))
		} else {
			return errResult(fmt.Sprintf("Decision %s not found", params.RootID))
		}
	} else {
		// Find roots (decisions with no dependencies, or not depended-on by others in a dep chain)
		for _, d := range decs {
			if len(d.DependsOn) == 0 {
				nodes = append(nodes, buildNode(d))
			}
		}
		// Orphans: decisions that have deps but whose deps don't exist
		for _, d := range decs {
			if len(d.DependsOn) > 0 && !isChild[d.ID] {
				allMissing := true
				for _, dep := range d.DependsOn {
					if _, ok := byID[dep]; ok {
						allMissing = false
						break
					}
				}
				if allMissing {
					nodes = append(nodes, buildNode(d))
				}
			}
		}
	}

	result := map[string]interface{}{"nodes": nodes}
	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

// --- Tool definitions ---

func (s *Server) toolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "read_decisions",
			Description: "List all architectural decisions in the current session. Supports filtering by status, category, feature, and source.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"status": {"type": "string", "enum": ["all", "pending", "answered", "delegated"], "description": "Filter by decision status. Default: all"},
					"category": {"type": "string", "description": "Filter by category name (case-insensitive substring match)"},
					"feature": {"type": "string", "description": "Filter by feature tag"},
					"source": {"type": "string", "enum": ["user", "auto", "agent", "discovered", "invalidated"], "description": "Filter by decision source"}
				},
				"additionalProperties": false
			}`),
		},
		{
			Name:        "list_pending",
			Description: "Get pending decisions that need human resolution, ordered by impact (highest first).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {"type": "string", "description": "Filter pending decisions to a specific category"},
					"include_delegated": {"type": "boolean", "description": "Include delegated decisions. Default: false"}
				},
				"additionalProperties": false
			}`),
		},
		{
			Name:        "confirm_decision",
			Description: "Set the answer on a decision by ID. Automatically invalidates dependent decisions if the answer changes.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "string", "description": "Decision ID (e.g. STA-0001)"},
					"answer": {"type": "string", "description": "The answer text"},
					"reasoning": {"type": "string", "description": "Optional reasoning for the decision"},
					"source": {"type": "string", "enum": ["user", "agent"], "description": "Who made this decision. Default: user"}
				},
				"required": ["id", "answer"],
				"additionalProperties": false
			}`),
		},
		{
			Name:        "update_decision",
			Description: "Modify properties of an existing decision (question, context, options, category, features, impact, dependencies). Does NOT change the answer.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "string", "description": "Decision ID to update"},
					"question": {"type": "string"},
					"context": {"type": "string"},
					"category": {"type": "string"},
					"options": {"type": "array", "items": {"type": "object", "properties": {"key": {"type": "string"}, "label": {"type": "string"}}, "required": ["key", "label"]}},
					"features": {"type": "array", "items": {"type": "string"}},
					"impact": {"type": "integer", "minimum": 0, "maximum": 10},
					"dependsOn": {"type": "array", "items": {"type": "string"}}
				},
				"required": ["id"],
				"additionalProperties": false
			}`),
		},
		{
			Name:        "get_session_state",
			Description: "Get overall session status including task, decision counts, progress, features, and categories.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"additionalProperties": false
			}`),
		},
		{
			Name:        "get_decision_tree",
			Description: "Get a hierarchical view of decisions organized by dependency relationships.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"root_id": {"type": "string", "description": "Start tree from a specific decision ID"},
					"group_by": {"type": "string", "enum": ["dependencies", "category", "feature"], "description": "How to organize. Default: dependencies"}
				},
				"additionalProperties": false
			}`),
		},
		{
			Name: "register_decision",
			Description: "Register a new decision that the executor is about to materialize. " +
				"Call this BEFORE writing any file — it's the only way to record the choices " +
				"(file layout, package/library, patterns, names, defaults, trade-offs) that the " +
				"team will need to review later. The decision is auto-resolved with the chosen " +
				"value and returned with a decision_id. Pass that id (along with any other " +
				"decision_ids relevant to the file) to write_file when you're ready to write.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category":     {"type": "string", "description": "Short category label: Stack, Data, API, Auth, UI, Build, Testing, Structure, Scope, Misc, etc."},
					"question":     {"type": "string", "description": "The specific choice framed as a concrete question, e.g. 'Where should the compiled binary live?'"},
					"chosen":       {"type": "string", "description": "The answer you're going with, as a short label, e.g. 'bin/server'"},
					"alternatives": {"type": "array", "items": {"type": "string"}, "description": "2-3 alternatives you considered and rejected, e.g. ['./server at root', 'dist/server']"},
					"reasoning":    {"type": "string", "description": "One-line justification for the chosen option"}
				},
				"required": ["category", "question", "chosen"],
				"additionalProperties": false
			}`),
		},
		{
			Name: "write_file",
			Description: "Write a file to the working directory. This is the ONLY way to create or " +
				"modify files in the executor phase — the native Write/Edit tools are not available. " +
				"Every call must supply decision_ids for the decisions that justify this write; " +
				"those must have been registered via register_decision first. An empty decision_ids " +
				"list is rejected. Relative paths are anchored at the working directory. Paths that " +
				"escape the working directory are rejected.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"decision_ids": {"type": "array", "items": {"type": "string"}, "description": "IDs returned by register_decision that cover the choices in this file"},
					"path":         {"type": "string", "description": "Relative or absolute path under the working directory"},
					"content":      {"type": "string", "description": "Full file contents"}
				},
				"required": ["decision_ids", "path", "content"],
				"additionalProperties": false
			}`),
		},
	}
}

// --- Helpers ---

func textResult(text string) callToolResult {
	return callToolResult{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{{Type: "text", Text: text}},
	}
}

func errResult(msg string) callToolResult {
	r := textResult(msg)
	r.IsError = true
	return r
}

func (s *Server) writeResult(id int64, result interface{}) {
	data, _ := json.Marshal(result)
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  data,
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", out)
}

func (s *Server) writeError(id int64, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", out)
}
