package agent

// CareLevel controls how much the executor explains and verifies.
type CareLevel string

const (
	CareLevelSkip     CareLevel = "skip"
	CareLevelLow      CareLevel = "low"
	CareLevelMedium   CareLevel = "medium"
	CareLevelHigh     CareLevel = "high"
	CareLevelParanoid CareLevel = "paranoid"
)

const DecomposePrompt = `You are in DEFER MODE. Your ONLY job is to identify decisions.

Do NOT write code. Do NOT explain. Do NOT discuss. Just output decisions.

Rules:
1. Identify every decision the task requires. Group by category.
2. High-level first. Let answers cascade. Bundle related decisions.
3. Every decision MUST have concrete options plus "Choose for me" as the last option.

You MUST output a ` + "```defer-decisions" + ` JSON block:

` + "```defer-decisions" + `
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [
      {"key": "A", "label": "Node.js with Express"},
      {"key": "B", "label": "Python with FastAPI"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Determines the entire backend ecosystem"
  }
]
` + "```" + `

Rules for the JSON:
- "category": short name (e.g. "Stack", "Data", "API", "Auth", "UI")
- "question": clear, specific question
- "options": 2-6 options, each with "key" (uppercase letter) and "label". Last must be "Choose for me"
- "context": one sentence explaining why this matters

You have access to Read, Glob, and Grep tools to explore the project before identifying decisions. Use them to understand the existing codebase.`

const ExecutePromptTemplate = `You are implementing a software project. Domain: %s

%s

IMPORTANT: Never ask the user for permission or confirmation. Never say "should I continue?" or "would you like me to...". Make every decision yourself and implement everything. The user monitors your decisions and will challenge any they disagree with.

You have these tools: Read, Write, Edit, Bash, Glob, Grep. Use them to implement the full project.

When done, say "Implementation complete."`

var CarePrompts = map[CareLevel]string{
	CareLevelSkip:     "Implement everything autonomously. Make all decisions yourself. Move fast. Minimal explanation.",
	CareLevelLow:      "Implement this domain autonomously. Briefly note key decisions you make.",
	CareLevelMedium:   "Implement this domain autonomously. Explain important implementation choices as you make them.",
	CareLevelHigh:     "Implement this domain autonomously. Explain every significant decision with your reasoning.",
	CareLevelParanoid: "Implement this domain autonomously. Explain EVERY decision in detail -- file names, variable names, patterns, config values -- with thorough reasoning for each.",
}

const VerifyPrompt = `Review this domain implementation. Check for errors, missing pieces, or mismatches with the decisions. Be concise. Only flag real problems.

If correct and complete, respond with: VERIFIED OK
If issues exist, respond with: NEEDS FIX followed by a numbered list of issues.`

const ExtractPrompt = `Review this implementation and extract every decision that was made. Include: files created, libraries chosen, patterns used, naming conventions, config values, architecture choices.

For EACH decision, include what was chosen AND what the alternatives were.

Output ONLY a JSON array:
[{"category": "...", "question": "what was the choice about", "options": [{"key": "A", "label": "what was chosen"}, {"key": "B", "label": "alternative 1"}, {"key": "C", "label": "alternative 2"}], "answer": "A", "reasoning": "why this was chosen"}]

The first option (A) should always be what was actually chosen. Other options are what COULD have been chosen instead.`

const ScanPrompt = `You are analyzing an EXISTING codebase to discover all decisions that were already made.

Use Read, Glob, and Grep tools to explore the project. Look at:
- Package manager files (go.mod, package.json, Cargo.toml, etc.)
- Configuration files (tsconfig, eslint, docker, CI/CD)
- Framework and library choices
- Database schemas and migrations
- Project structure and architecture patterns
- Authentication and authorization approach
- API design (REST, GraphQL, tRPC)
- Styling approach (CSS, Tailwind, etc.)
- Testing framework and patterns
- Deployment configuration

For each decision you discover, record it with the ACTUAL choice that was made (not options -- the project already chose).

Output a ` + "```defer-decisions" + ` JSON block:
` + "```defer-decisions" + `
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [{"key": "A", "label": "Go with Gin"}],
    "context": "Discovered from go.mod and main.go"
  }
]
` + "```" + `

Rules:
- "category": group by domain (Stack, Data, Auth, API, UI, Testing, Deploy, etc.)
- "question": what was the choice about
- "options": single option with what was actually chosen (the project already decided)
- "context": where you found this (which file/config)

Be thorough. Scan the entire project.`

const PlanPrompt = `You are a software architect. Given the task and existing decisions, identify ALL implementation decisions that still need to be made.

For EACH decision, provide 3-4 concrete options to choose from.

Output ONLY a JSON array:
[{"category": "...", "question": "what needs to be decided", "options": [{"key": "A", "label": "option 1"}, {"key": "B", "label": "option 2"}, {"key": "C", "label": "option 3"}], "answer": "A", "reasoning": "why you recommend this option"}]

The "answer" field is the KEY (A, B, C) of your recommended option. Always provide real alternatives, not just your recommendation.`
