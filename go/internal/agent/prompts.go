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

Output ONLY a JSON array:
[{"category": "...", "question": "what was the choice about", "answer": "what was chosen", "reasoning": "why"}]`

const PlanPrompt = `You are a software architect. Given the task and existing decisions, identify ALL implementation decisions that still need to be made.

Output ONLY a JSON array:
[{"category": "...", "question": "what needs to be decided", "answer": "your recommended choice", "reasoning": "why this matters"}]`
