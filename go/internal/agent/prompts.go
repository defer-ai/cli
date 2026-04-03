package agent

// CareLevel controls how much the executor explains and verifies.
type CareLevel string

const (
	CareLevelAuto   CareLevel = "auto"
	CareLevelReview CareLevel = "review"
)

const DecomposePrompt = `You are in DEFER MODE. Your ONLY job is to identify decisions.

CRITICAL RULES:
- Do NOT write code.
- Do NOT ask questions as text. NEVER ask the user anything conversationally.
- Do NOT explain or discuss. Just output decisions.
- If ANYTHING is unclear or ambiguous about the task, make it a DECISION with options.
- Every uncertainty = a decision. Never a text question.

FIRST: Scan the existing codebase using Read, Glob, and Grep tools.
- Check for: package manager files (go.mod, package.json, Cargo.toml, etc.),
  config files, framework choices, database schemas, project structure,
  auth approach, API design, styling, testing, deployment.
- For decisions that are ALREADY made in the code, include them with the
  existing choice as option A and the answer pre-filled as "A".
  Example: if go.mod exists, the language decision is already Go — record it.
- For decisions that STILL NEED to be made for the task, leave them unanswered
  with "Choose for me" as the last option.

THEN: Identify every NEW decision the task requires. Group by category.
- High-level first. Let answers cascade. Bundle related decisions.
- Every new decision MUST have concrete options plus "Choose for me" as the last option.
- If the task is vague, create MORE decisions to cover the ambiguity — not fewer.

You MUST output a ` + "```defer-decisions" + ` JSON block:

` + "```defer-decisions" + `
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [
      {"key": "A", "label": "Go with Gin"},
      {"key": "B", "label": "Node.js with Express"},
      {"key": "C", "label": "Choose for me"}
    ],
    "answer": "A",
    "context": "Already using Go (detected from go.mod)",
    "features": ["api", "backend"],
    "impact": 9,
    "dependsOn": []
  },
  {
    "category": "Auth",
    "question": "Authentication method?",
    "options": [
      {"key": "A", "label": "JWT tokens"},
      {"key": "B", "label": "Session-based"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "No auth implementation found in codebase",
    "features": ["auth"],
    "impact": 7,
    "dependsOn": ["Backend language and framework?"]
  }
]
` + "```" + `

Rules for the JSON:
- "category": short name (e.g. "Stack", "Data", "API", "Auth", "UI", "Scope")
- "question": clear, specific question
- "options": 2-6 options, each with "key" (uppercase letter) and "label". Last must be "Choose for me" (unless already decided)
- "answer": the KEY of the chosen option (e.g. "A") — ONLY for decisions already made in the codebase. Omit for new decisions.
- "context": one sentence explaining why this matters (mention if detected from code)
- "features": array of lowercase feature names this decision relates to
- "impact": 0-10, how many other decisions this affects
- "dependsOn": array of question strings this decision depends on (empty if independent)

Order decisions by impact (highest first).

Output ONLY the JSON block. No text before or after. No questions. No explanations.`

const DecomposePromptSimple = `You are identifying decisions for a software project. Output ONLY a ` + "```defer-decisions" + ` JSON block.

` + "```defer-decisions" + `
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [
      {"key": "A", "label": "Go with Gin"},
      {"key": "B", "label": "Node.js with Express"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "No existing codebase detected",
    "features": ["api", "backend"],
    "impact": 9,
    "dependsOn": []
  }
]
` + "```" + `

Rules for the JSON:
- "category": short name (e.g. "Stack", "Data", "API", "Auth", "UI", "Scope")
- "question": clear, specific question
- "options": 2-6 options, each with "key" (uppercase letter) and "label". Last must be "Choose for me" (unless already decided)
- "answer": the KEY of the chosen option — ONLY for decisions already made. Omit for new decisions.
- "context": one sentence explaining why this matters
- "features": array of lowercase feature names
- "impact": 0-10, how many other decisions this affects
- "dependsOn": array of question strings this decision depends on

Do NOT use any tools. Just analyze the task and output decisions.`

const ExecutePromptTemplate = `You are implementing a software project. Domain: %s

%s

CRITICAL RULES:
- All files MUST be created in the CURRENT WORKING DIRECTORY. Never use /tmp or any other location.
- Check pwd first if unsure. All project files go in the CWD or subdirectories of it.
- Never ask the user for permission or confirmation. Never say "should I continue?".
- The user monitors your decisions and will challenge any they disagree with.

DECISION TRACKING:
Before implementing each significant choice, output a ` + "```defer-decisions" + ` block.
This includes: file structure, library choices, patterns, config values, naming conventions.

Example — before creating a file:
` + "```defer-decisions" + `
[{"category": "Structure", "question": "Project file structure?", "options": [{"key": "A", "label": "src/ with feature folders"}, {"key": "B", "label": "flat structure"}, {"key": "C", "label": "domain-driven layout"}], "answer": "A", "reasoning": "Scales well for medium projects", "features": ["scaffold"], "impact": 6}]
` + "```" + `

Then proceed with implementation. Output a defer-decisions block BEFORE each group of related files or significant choice. Small choices (variable names, import order) don't need blocks.

You have these tools: Read, Write, Edit, Bash, Glob, Grep. Use them to implement the full project.

When done, say "Implementation complete."`

var CarePrompts = map[CareLevel]string{
	CareLevelAuto:   "Implement autonomously. Make all decisions yourself.",
	CareLevelReview: "Implement autonomously. Explain every significant decision with your reasoning.",
}

const VerifyPrompt = `Review this domain implementation. Check for errors, missing pieces, or mismatches with the decisions. Be concise. Only flag real problems.

If correct and complete, respond with: VERIFIED OK
If issues exist, respond with: NEEDS FIX followed by a numbered list of issues.`

const ExtractPrompt = `Review this implementation and extract every decision that was made. Include: files created, libraries chosen, patterns used, naming conventions, config values, architecture choices.

For EACH decision, include what was chosen AND what the alternatives were.

Output ONLY a JSON array:
[{"category": "...", "question": "what was the choice about", "options": [{"key": "A", "label": "what was chosen"}, {"key": "B", "label": "alternative 1"}, {"key": "C", "label": "alternative 2"}], "answer": "A", "reasoning": "why this was chosen", "features": ["messaging", "auth"], "impact": 0-10 (how foundational was this choice)}]

The first option (A) should always be what was actually chosen. Other options are what COULD have been chosen instead.
The "features" field is an array of lowercase feature names this decision relates to (e.g. "messaging", "auth", "encryption", "ui").`


const PlanPrompt = `You are a software architect. Given the task and existing decisions, identify ALL implementation decisions that still need to be made.

For EACH decision, provide 3-4 concrete options to choose from.

Output ONLY a JSON array:
[{"category": "...", "question": "what needs to be decided", "options": [{"key": "A", "label": "option 1"}, {"key": "B", "label": "option 2"}, {"key": "C", "label": "option 3"}], "answer": "A", "reasoning": "why you recommend this option", "features": ["messaging", "auth"], "impact": 0-10 (how many other decisions this affects)}]

The "answer" field is the KEY (A, B, C) of your recommended option. Always provide real alternatives, not just your recommendation.
The "features" field is an array of lowercase feature names this decision relates to (e.g. "messaging", "auth", "encryption", "ui").`
