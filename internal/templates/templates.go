package templates

// Target identifies which AI tool the template is for.
type Target string

const (
	TargetClaudeCode Target = "claude-code"
	TargetCursor     Target = "cursor"
	TargetCopilot    Target = "copilot"
	TargetCodex      Target = "codex"
	TargetWindsurf   Target = "windsurf"
	TargetZed        Target = "zed"
	TargetCline      Target = "cline"
	TargetGemini     Target = "gemini"
	TargetAider      Target = "aider"
	TargetContinue   Target = "continue"
	TargetUniversal  Target = "universal"
)

// Template holds a file template for init.
type Template struct {
	Filename    string
	Description string
	Content     string
}

// integrationSuffix is the common instruction block appended to tool-specific templates.
const integrationSuffix = `
When the user gives you a task:
1. Decompose it into decisions (step 1-2 above)
2. Present them as a markdown table and wait for confirmation
3. Once confirmed, implement autonomously (step 4)
4. Update DECISIONS.md after implementation (step 5-6)
5. Verify your work (step 7)

If DECISIONS.md already exists, read it first — you're resuming, not starting fresh.
Only add new decisions; don't re-decide what's already been decided unless asked.
`

// Templates is the registry of available templates.
var Templates = map[Target]Template{
	TargetClaudeCode: {
		Filename:    "CLAUDE.md",
		Description: "Claude Code (CLAUDE.md)",
		Content: DeferProcess + `
## Claude Code Integration

This file is automatically loaded by Claude Code at the start of every conversation.
The defer process above is your operating mode for this project.
` + integrationSuffix,
	},
	TargetCursor: {
		Filename:    ".cursorrules",
		Description: "Cursor (.cursorrules)",
		Content: DeferProcess + `
## Cursor Integration

This file is automatically loaded by Cursor for every conversation in this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetCopilot: {
		Filename:    ".github/copilot-instructions.md",
		Description: "GitHub Copilot (.github/copilot-instructions.md)",
		Content: DeferProcess + `
## Copilot Integration

This file is loaded by GitHub Copilot as custom instructions for this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetCodex: {
		Filename:    "AGENTS.md",
		Description: "OpenAI Codex / Amp (AGENTS.md)",
		Content: DeferProcess + `
## Codex / Amp Integration

This file is loaded by OpenAI Codex and Amp as agent instructions for this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetWindsurf: {
		Filename:    ".windsurf/rules/defer.md",
		Description: "Windsurf (.windsurf/rules/defer.md)",
		Content: `---
trigger: always_on
description: Defer — zero-autonomy AI decision process
---
` + DeferProcess + `
## Windsurf Integration

This rule is automatically loaded by Windsurf for every conversation in this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetZed: {
		Filename:    ".rules",
		Description: "Zed (.rules)",
		Content: DeferProcess + `
## Zed Integration

This file is automatically loaded by Zed as project rules.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetCline: {
		Filename:    ".clinerules",
		Description: "Cline (.clinerules)",
		Content: DeferProcess + `
## Cline Integration

This file is automatically loaded by Cline for every conversation in this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetGemini: {
		Filename:    "GEMINI.md",
		Description: "Gemini CLI (GEMINI.md)",
		Content: DeferProcess + `
## Gemini CLI Integration

This file is automatically loaded by Gemini CLI for this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetAider: {
		Filename:    "CONVENTIONS.md",
		Description: "Aider (CONVENTIONS.md)",
		Content: DeferProcess + `
## Aider Integration

This file is loaded by Aider as project conventions (add "read: CONVENTIONS.md" to .aider.conf.yml).
Follow the defer process above as your operating mode.
` + integrationSuffix,
	},
	TargetContinue: {
		Filename:    ".continue/rules/defer.md",
		Description: "Continue (.continue/rules/defer.md)",
		Content: DeferProcess + `
## Continue Integration

This rule is automatically loaded by Continue for every conversation in this project.
Follow the defer process above as your operating mode.
` + integrationSuffix,
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
		TargetWindsurf,
		TargetZed,
		TargetCline,
		TargetGemini,
		TargetAider,
		TargetContinue,
		TargetUniversal,
	}
}
