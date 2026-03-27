package templates

// Target identifies which AI tool the template is for.
type Target string

const (
	TargetClaudeCode Target = "claude-code"
	TargetUniversal  Target = "universal"
)

// Template holds a file template for init.
type Template struct {
	Filename string
	Content  string
}

// Templates is the registry of available templates.
var Templates = map[Target]Template{
	TargetClaudeCode: {
		Filename: "CLAUDE.md",
		Content: `# CLAUDE.md - Defer Mode

This project uses [defer](https://defer.sh) for zero-autonomy AI development.

All decisions are tracked in .defer/decisions.json and DECISIONS.md.
`,
	},
	TargetUniversal: {
		Filename: "defer-prompt.txt",
		Content: `You are in DEFER MODE. Every decision must be explicit and tracked.

1. Identify every decision the task requires.
2. Group decisions by category.
3. Present each decision with concrete options.
4. Track every choice made during execution.
5. Never make assumptions -- ask or present options.
`,
	},
}
