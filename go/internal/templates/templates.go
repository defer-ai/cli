package templates

// Target identifies which AI tool the template is for.
type Target string

const (
	TargetClaudeCode Target = "claude-code"
	TargetCursor     Target = "cursor"
	TargetCopilot    Target = "copilot"
	TargetCodex      Target = "codex"
	TargetUniversal  Target = "universal"
)

// Template holds a file template for init.
type Template struct {
	Filename    string
	Description string
	Content     string
}

// Templates is the registry of available templates.
var Templates = map[Target]Template{
	TargetClaudeCode: {
		Filename:    "CLAUDE.md",
		Description: "Claude Code (CLAUDE.md)",
		Content: DeferProcess + `
## Claude Code Integration

This file is automatically loaded by Claude Code at the start of every conversation.
The defer process above is your operating mode for this project.

When the user gives you a task:
1. Decompose it into decisions (step 1-2 above)
2. Present them as a markdown table and wait for confirmation
3. Once confirmed, implement autonomously (step 4)
4. Update DECISIONS.md after implementation (step 5-6)
5. Verify your work (step 7)

If DECISIONS.md already exists, read it first — you're resuming, not starting fresh.
Only add new decisions; don't re-decide what's already been decided unless asked.
`,
	},
	TargetCursor: {
		Filename:    ".cursorrules",
		Description: "Cursor (.cursorrules)",
		Content: DeferProcess + `
## Cursor Integration

This file is automatically loaded by Cursor for every conversation in this project.
Follow the defer process above as your operating mode.

When the user gives you a task:
1. Decompose it into decisions (step 1-2 above)
2. Present them as a markdown table and wait for confirmation
3. Once confirmed, implement autonomously (step 4)
4. Update DECISIONS.md after implementation (step 5-6)
5. Verify your work (step 7)

If DECISIONS.md already exists, read it first — you're resuming, not starting fresh.
Only add new decisions; don't re-decide what's already been decided unless asked.
`,
	},
	TargetCopilot: {
		Filename:    ".github/copilot-instructions.md",
		Description: "GitHub Copilot (.github/copilot-instructions.md)",
		Content: DeferProcess + `
## Copilot Integration

This file is loaded by GitHub Copilot as custom instructions for this project.
Follow the defer process above as your operating mode.

When the user gives you a task:
1. Decompose it into decisions (step 1-2 above)
2. Present them as a markdown table and wait for confirmation
3. Once confirmed, implement autonomously (step 4)
4. Update DECISIONS.md after implementation (step 5-6)
5. Verify your work (step 7)

If DECISIONS.md already exists, read it first — you're resuming, not starting fresh.
Only add new decisions; don't re-decide what's already been decided unless asked.
`,
	},
	TargetCodex: {
		Filename:    "AGENTS.md",
		Description: "OpenAI Codex (AGENTS.md)",
		Content: DeferProcess + `
## Codex Integration

This file is loaded by OpenAI Codex as agent instructions for this project.
Follow the defer process above as your operating mode.

When the user gives you a task:
1. Decompose it into decisions (step 1-2 above)
2. Present them as a markdown table and wait for confirmation
3. Once confirmed, implement autonomously (step 4)
4. Update DECISIONS.md after implementation (step 5-6)
5. Verify your work (step 7)

If DECISIONS.md already exists, read it first — you're resuming, not starting fresh.
Only add new decisions; don't re-decide what's already been decided unless asked.
`,
	},
	TargetUniversal: {
		Filename:    "DEFER.md",
		Description: "Universal (DEFER.md — copy into any tool's config)",
		Content:     DeferProcess,
	},
}

// TargetList returns all available targets in display order.
func TargetList() []Target {
	return []Target{
		TargetClaudeCode,
		TargetCursor,
		TargetCopilot,
		TargetCodex,
		TargetUniversal,
	}
}
