package api

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// ToolSet defines which tools are available.
type ToolSet int

const (
	ReadOnlyTools ToolSet = iota // decomposition: Read, Glob, Grep
	AllTools                     // execution: all 6 tools
	NoTools                      // extraction/verification: text only
)

// GetTools returns tool definitions for the given set.
func GetTools(set ToolSet) []anthropic.ToolUnionParam {
	switch set {
	case ReadOnlyTools:
		return []anthropic.ToolUnionParam{readTool(), globTool(), grepTool()}
	case AllTools:
		return []anthropic.ToolUnionParam{readTool(), writeTool(), editTool(), bashTool(), globTool(), grepTool()}
	case NoTools:
		return nil
	default:
		return nil
	}
}

func readTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Read",
			Description: anthropic.String("Read the contents of a file. Returns the file text. Use offset/limit for large files."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute or relative path to the file",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Line number to start reading from (0-based). Optional.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of lines to read. Optional.",
					},
				},
				Required: []string{"file_path"},
			},
		},
	}
}

func writeTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Write",
			Description: anthropic.String("Create or overwrite a file with the given content. Creates parent directories if needed."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The full content to write to the file",
					},
				},
				Required: []string{"file_path", "content"},
			},
		},
	}
}

func editTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Edit",
			Description: anthropic.String("Replace a specific string in a file. The old_text must match exactly (including whitespace)."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"old_text": map[string]interface{}{
						"type":        "string",
						"description": "The exact text to find and replace",
					},
					"new_text": map[string]interface{}{
						"type":        "string",
						"description": "The replacement text",
					},
				},
				Required: []string{"file_path", "old_text", "new_text"},
			},
		},
	}
}

func bashTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Bash",
			Description: anthropic.String("Execute a shell command and return stdout+stderr. Commands run in the project directory."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The shell command to execute",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds. Default 120.",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

func globTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Glob",
			Description: anthropic.String("Find files matching a glob pattern (e.g. '**/*.ts'). Returns a list of matching file paths."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Glob pattern to match files against",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory to search in. Defaults to project root.",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

func grepTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "Grep",
			Description: anthropic.String("Search file contents for a regex pattern. Returns matching lines with file paths and line numbers."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Regex pattern to search for",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File or directory to search in. Defaults to project root.",
					},
					"glob": map[string]interface{}{
						"type":        "string",
						"description": "Glob filter for files (e.g. '*.ts'). Optional.",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}
