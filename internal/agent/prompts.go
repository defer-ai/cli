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

const ExecutePromptTemplate = `You are implementing a software project based on confirmed decisions. Domain: %s

%s

CRITICAL RULES:
- All files MUST be created in the CURRENT WORKING DIRECTORY. Never use /tmp.
- Follow the decisions EXACTLY as specified below. Do not deviate.

DECISION PROTOCOL — THIS IS MANDATORY:
You MUST output a DECIDED line for every choice you make during implementation.
A "choice" is anything where you picked one option over another — no matter how small.

Format (each on its own line, pipe-separated):

DECIDED: category | question | answer | alternatives | reasoning
PENDING: category | question | A) opt1, B) opt2, C) opt3 | context
RESEARCH: question | what to investigate

When to use each:
- DECIDED: You made the choice yourself. Output IMMEDIATELY, before the next tool call.
- PENDING: The domain is marked "review" — the user must decide. Stop and wait.
- RESEARCH: You need more context. The system will investigate.

DEPTH REQUIRED — these are ALL decisions, not just the big ones:
- Creating a file → DECIDED about the file's purpose and location
- Choosing a function signature → DECIDED about the return type, parameter design
- Picking a data structure → DECIDED about slice vs map, struct fields
- Naming something → DECIDED about the naming convention
- Error handling → DECIDED about return error vs panic vs log
- Adding a dependency → DECIDED about which package and why
- Picking a pattern → DECIDED about MVC vs flat, middleware vs wrapper
- Config values → DECIDED about defaults, formats, locations
- Skipping something → DECIDED about what was excluded and why

WRONG (too shallow — only 2 decisions for an entire file):
  Write main.go
  DECIDED: Structure | Entry point? | main.go | cmd/main.go | Simple project

RIGHT (every choice in that file documented):
  Write main.go
  DECIDED: Structure | Entry point location? | main.go | cmd/main.go | Single-file project, no need for cmd/
  DECIDED: Structure | Package name? | main | app | Entry point must be package main
  DECIDED: Dependencies | Argument parsing? | os.Args | cobra, flag | Zero dependencies for a simple CLI
  DECIDED: Error handling | How to report errors? | fmt.Fprintf(os.Stderr, ...) + os.Exit(1) | log.Fatal, panic | Clean stderr output
  DECIDED: Structure | Function organization? | One function per command (cmdAdd, cmdList, cmdDone) | Single switch block | Testable, readable

Rules:
- Do NOT batch. Output each DECIDED line immediately after the choice.
- Do NOT skip "obvious" choices. If you chose X over Y, document it.
- Do NOT re-ask questions already answered in the decisions list below.
- Each DECIDED/PENDING/RESEARCH must be on a SINGLE LINE.
- Read-only operations (Read, Glob, Grep) do not need decision lines.
- Domains marked "review" MUST use PENDING, never DECIDED.

WORKFLOW — follow this cycle for EVERY file:
1. BEFORE writing: output DECIDED lines for the choices you're about to make
   (file location, purpose, pattern, naming, dependencies, what's included/excluded)
2. Write the file
3. AFTER writing: output DECIDED lines for any choices that emerged during writing
   (error handling approach, specific APIs used, config defaults, struct design)
4. Move to the next file — repeat from step 1

NEVER write multiple files in a row without DECIDED lines between them.

Implement step by step. When done, say "Implementation complete."`

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
