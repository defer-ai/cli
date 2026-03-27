package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecRead(t *testing.T) {
	dir := t.TempDir()
	content := "line 1\nline 2\nline 3\n"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644)

	input, _ := json.Marshal(map[string]interface{}{"file_path": "test.txt"})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Read", Input: input}, dir)

	if result.IsError {
		t.Fatalf("error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line 1") {
		t.Errorf("missing line 1 in output: %s", result.Content)
	}
}

func TestExecReadNotFound(t *testing.T) {
	dir := t.TempDir()
	input, _ := json.Marshal(map[string]interface{}{"file_path": "nonexistent.txt"})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Read", Input: input}, dir)

	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
}

func TestExecWrite(t *testing.T) {
	dir := t.TempDir()
	input, _ := json.Marshal(map[string]interface{}{
		"file_path": "subdir/new.txt",
		"content":   "hello world",
	})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Write", Input: input}, dir)

	if result.IsError {
		t.Fatalf("error: %s", result.Content)
	}

	data, err := os.ReadFile(filepath.Join(dir, "subdir", "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("file content = %q", string(data))
	}
}

func TestExecEdit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("hello world"), 0644)

	input, _ := json.Marshal(map[string]interface{}{
		"file_path": "edit.txt",
		"old_text":  "world",
		"new_text":  "Go",
	})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Edit", Input: input}, dir)

	if result.IsError {
		t.Fatalf("error: %s", result.Content)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "edit.txt"))
	if string(data) != "hello Go" {
		t.Errorf("file content = %q", string(data))
	}
}

func TestExecEditNotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("hello"), 0644)

	input, _ := json.Marshal(map[string]interface{}{
		"file_path": "edit.txt",
		"old_text":  "nonexistent",
		"new_text":  "new",
	})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Edit", Input: input}, dir)

	if !result.IsError {
		t.Fatal("expected error for missing old_text")
	}
}

func TestExecBash(t *testing.T) {
	dir := t.TempDir()
	input, _ := json.Marshal(map[string]interface{}{"command": "echo hello"})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Bash", Input: input}, dir)

	if result.IsError {
		t.Fatalf("error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("output = %q", result.Content)
	}
}

func TestExecGlob(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0644)

	input, _ := json.Marshal(map[string]interface{}{"pattern": "*.go"})
	result := ExecuteTool(context.Background(), ToolCall{ID: "1", Name: "Glob", Input: input}, dir)

	if result.IsError {
		t.Fatalf("error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "a.go") || !strings.Contains(result.Content, "b.go") {
		t.Errorf("missing .go files in: %s", result.Content)
	}
	if strings.Contains(result.Content, "c.txt") {
		t.Error("should not contain .txt files")
	}
}

func TestToolCallDescription(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Write", `{"file_path": "src/auth.ts"}`, "Create file src/auth.ts"},
		{"Edit", `{"file_path": "config.ts"}`, "Edit config.ts"},
		{"Bash", `{"command": "npm install prisma"}`, "Run: npm install prisma"},
		{"Read", `{"file_path": "go.mod"}`, "Read go.mod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ToolCall{Name: tt.name, Input: json.RawMessage(tt.input)}
			got := tc.HumanDescription()
			if got != tt.want {
				t.Errorf("HumanDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsMajorAction(t *testing.T) {
	major := []string{"Write", "Edit", "Bash"}
	minor := []string{"Read", "Glob", "Grep"}

	for _, name := range major {
		tc := ToolCall{Name: name}
		if !tc.IsMajorAction() {
			t.Errorf("%s should be major", name)
		}
	}
	for _, name := range minor {
		tc := ToolCall{Name: name}
		if tc.IsMajorAction() {
			t.Errorf("%s should not be major", name)
		}
	}
}
