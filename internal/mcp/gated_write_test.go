package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// callGatedTool is a small helper that invokes one of the gated-write
// tools via the full JSON-RPC path and returns the parsed call result.
// Keeps each test focused on the tool's behavior rather than the
// JSON-RPC boilerplate.
func callGatedTool(t *testing.T, cwd, name string, args map[string]interface{}) callToolResult {
	t.Helper()
	resps := runServerRequest(t, cwd, makeReq(1, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	}))
	if len(resps) != 1 {
		t.Fatalf("%s: expected 1 response, got %d", name, len(resps))
	}
	if resps[0].Error != nil {
		t.Fatalf("%s: JSON-RPC error: %s", name, resps[0].Error.Message)
	}
	var result callToolResult
	if err := json.Unmarshal(resps[0].Result, &result); err != nil {
		t.Fatalf("%s: cannot parse result: %v", name, err)
	}
	return result
}

func TestRegisterDecisionHappyPath(t *testing.T) {
	cwd := t.TempDir()
	// Start with an empty .defer directory (no store yet) so we also
	// exercise the "bootstrap a fresh store" path.
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	result := callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category":     "Build",
		"question":     "Where should the compiled binary live?",
		"chosen":       "bin/server",
		"alternatives": []string{"./server at repo root", "dist/server"},
		"reasoning":    "bin/ is gitignore-friendly and keeps the root clean",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if body["status"] != "resolved" {
		t.Errorf("status = %v, want resolved", body["status"])
	}
	if body["resolved_answer"] != "bin/server" {
		t.Errorf("resolved_answer = %v, want bin/server", body["resolved_answer"])
	}
	id, _ := body["decision_id"].(string)
	if !strings.HasPrefix(id, "BUI-") {
		t.Errorf("decision_id = %q, want BUI-* prefix (Build category)", id)
	}
}

func TestRegisterDecisionRejectsMissingFields(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	cases := []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{"missing category", map[string]interface{}{"question": "q?", "chosen": "x"}, "category"},
		{"missing question", map[string]interface{}{"category": "Build", "chosen": "x"}, "question"},
		{"missing chosen", map[string]interface{}{"category": "Build", "question": "q?"}, "chosen"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := callGatedTool(t, cwd, "register_decision", tc.args)
			if !result.IsError {
				t.Fatal("expected error, got success")
			}
			if !strings.Contains(result.Content[0].Text, tc.want) {
				t.Errorf("error should mention %q, got: %s", tc.want, result.Content[0].Text)
			}
		})
	}
}

func TestRegisterDecisionStoresAlternatives(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category":     "Stack",
		"question":     "HTTP framework?",
		"chosen":       "net/http stdlib",
		"alternatives": []string{"Gin", "Echo", "Fiber"},
		"reasoning":    "zero deps, stdlib is enough for two endpoints",
	})

	// Read the persisted store and verify the decision round-tripped.
	raw, err := os.ReadFile(filepath.Join(cwd, ".defer", "decisions.json"))
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	var stored map[string]interface{}
	json.Unmarshal(raw, &stored)
	decs, _ := stored["decisions"].([]interface{})
	if len(decs) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decs))
	}
	d := decs[0].(map[string]interface{})
	opts, _ := d["options"].([]interface{})
	if len(opts) != 4 {
		t.Errorf("expected 4 options (chosen + 3 alternatives), got %d", len(opts))
	}
	// First option must be the chosen value.
	first := opts[0].(map[string]interface{})
	if first["label"] != "net/http stdlib" {
		t.Errorf("first option = %v, want the chosen value", first["label"])
	}
	if d["reasoning"] != "zero deps, stdlib is enough for two endpoints" {
		t.Errorf("reasoning not stored: %v", d["reasoning"])
	}
	if d["source"] != "agent" {
		t.Errorf("source = %v, want agent", d["source"])
	}
}

func TestWriteFileHappyPath(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	// Register a decision first so we have an id to pass to write_file.
	regResult := callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category": "Build",
		"question": "binary name?",
		"chosen":   "server",
	})
	var reg map[string]interface{}
	json.Unmarshal([]byte(regResult.Content[0].Text), &reg)
	decisionID := reg["decision_id"].(string)

	result := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": []string{decisionID},
		"path":         "Makefile",
		"content":      "BINARY := server\n.PHONY: build\nbuild:\n\tgo build -o $(BINARY) .\n",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	// File must actually be on disk at the expected path.
	written, err := os.ReadFile(filepath.Join(cwd, "Makefile"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if !strings.Contains(string(written), "BINARY := server") {
		t.Errorf("written content missing expected line: %s", written)
	}
}

func TestWriteFileRejectsEmptyDecisionIDs(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	result := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": []string{},
		"path":         "foo.txt",
		"content":      "hello",
	})
	if !result.IsError {
		t.Fatal("expected error, got success — empty decision_ids should be rejected")
	}
	if !strings.Contains(result.Content[0].Text, "register_decision") {
		t.Errorf("error should redirect the model to register_decision, got: %s", result.Content[0].Text)
	}
	// File must NOT exist.
	if _, err := os.Stat(filepath.Join(cwd, "foo.txt")); !os.IsNotExist(err) {
		t.Error("file was written despite the rejection")
	}
}

func TestWriteFileRejectsUnknownDecisionID(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	// Register a real decision so we have a valid store.
	callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category": "Stack", "question": "lang?", "chosen": "Go",
	})

	// Try to write with a bogus id.
	result := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": []string{"BOGUS-9999"},
		"path":         "main.go",
		"content":      "package main",
	})
	if !result.IsError {
		t.Fatal("expected error, got success")
	}
	if !strings.Contains(result.Content[0].Text, "BOGUS-9999") {
		t.Errorf("error should name the bogus id, got: %s", result.Content[0].Text)
	}
}

func TestWriteFileRejectsPathEscape(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	regResult := callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category": "Scope", "question": "where?", "chosen": "/etc/passwd",
	})
	var reg map[string]interface{}
	json.Unmarshal([]byte(regResult.Content[0].Text), &reg)
	decisionID := reg["decision_id"].(string)

	result := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": []string{decisionID},
		"path":         "../../../etc/passwd",
		"content":      "malicious",
	})
	if !result.IsError {
		t.Fatal("path escape should be rejected")
	}
	if !strings.Contains(result.Content[0].Text, "escapes") {
		t.Errorf("error should mention escape, got: %s", result.Content[0].Text)
	}
}

func TestWriteFileCreatesParentDirs(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	regResult := callGatedTool(t, cwd, "register_decision", map[string]interface{}{
		"category": "Structure", "question": "layout?", "chosen": "nested",
	})
	var reg map[string]interface{}
	json.Unmarshal([]byte(regResult.Content[0].Text), &reg)
	decisionID := reg["decision_id"].(string)

	result := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": []string{decisionID},
		"path":         "pkg/server/handlers.go",
		"content":      "package server\n",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if _, err := os.Stat(filepath.Join(cwd, "pkg", "server", "handlers.go")); err != nil {
		t.Errorf("nested file not created: %v", err)
	}
}

// TestWriteFileEndToEndRegisterThenWrite walks the full expected flow
// the executor will use in production: call register_decision for each
// choice, collect the ids, then call write_file once with all of them.
// This is the ergonomic happy path we want Claude to fall into.
func TestWriteFileEndToEndRegisterThenWrite(t *testing.T) {
	cwd := t.TempDir()
	os.MkdirAll(filepath.Join(cwd, ".defer"), 0o755)

	ids := make([]string, 0, 3)
	for _, dec := range []map[string]interface{}{
		{"category": "Stack", "question": "language?", "chosen": "Python",
			"alternatives": []string{"Go", "Node.js"}, "reasoning": "Flask familiarity"},
		{"category": "API", "question": "framework?", "chosen": "Flask 3.0.0",
			"alternatives": []string{"FastAPI", "Django"}, "reasoning": "tiny scope"},
		{"category": "Structure", "question": "layout?", "chosen": "single app.py",
			"alternatives": []string{"blueprints", "multi-module"}, "reasoning": "two routes, no split needed"},
	} {
		r := callGatedTool(t, cwd, "register_decision", dec)
		if r.IsError {
			t.Fatalf("register_decision error: %s", r.Content[0].Text)
		}
		var body map[string]interface{}
		json.Unmarshal([]byte(r.Content[0].Text), &body)
		ids = append(ids, body["decision_id"].(string))
	}

	writeResult := callGatedTool(t, cwd, "write_file", map[string]interface{}{
		"decision_ids": ids,
		"path":         "app.py",
		"content":      "from flask import Flask\napp = Flask(__name__)\n",
	})
	if writeResult.IsError {
		t.Fatalf("write_file error: %s", writeResult.Content[0].Text)
	}

	// Verify the file exists with the right content and the store has
	// all three decisions from register_decision calls.
	body, _ := os.ReadFile(filepath.Join(cwd, "app.py"))
	if !strings.Contains(string(body), "from flask import Flask") {
		t.Error("app.py missing expected content")
	}
	storeRaw, _ := os.ReadFile(filepath.Join(cwd, ".defer", "decisions.json"))
	var store map[string]interface{}
	json.Unmarshal(storeRaw, &store)
	decs := store["decisions"].([]interface{})
	if len(decs) != 3 {
		t.Errorf("expected 3 decisions in store, got %d", len(decs))
	}
}
