package agent

// CareLevel controls how much the executor explains and verifies.
type CareLevel string

const (
	CareLevelAuto   CareLevel = "auto"
	CareLevelReview CareLevel = "review"
)

// DecomposePrompt is the tool-using decomposition prompt. The agent is
// given Read/Glob/Grep and is expected to scan the existing codebase
// before outputting a defer-decisions JSON block.
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

// DecomposePromptSimple is the text-only fallback used when the tool-using
// path fails. Refined via the DSPy prompt comparison — the previous version
// included a JSON example that invited weaker models to copy syntax errors
// (single quotes, malformed nesting). This version uses numbered structural
// requirements and an explicit double-quote instruction with no example.
const DecomposePromptSimple = `You are an architect. Your job is to plan a software project by listing
every decision that must be made before any code is written.

Output a JSON array. Nothing else. No explanation. No prose. Use double
quotes only — never single quotes.

Each item in the array represents one decision and must have:
1. category   — short label (Stack, Data, API, Auth, UI, Deploy, Testing, Scope)
2. question   — concrete, specific question (no vague words like "good" or "best")
3. options    — array of 3-4 objects, each {"key": "A", "label": "..."}.
                Last option must always be {"key": "X", "label": "Choose for me"}.
4. impact     — integer 0-10. How many other decisions hinge on this one.
5. dependsOn  — array of question strings this decision depends on (may be empty).

Aim for 12-25 decisions covering: language, framework, data layer, API style,
auth, error handling, testing, deployment, naming, dependencies.

Order from highest impact to lowest. Do NOT use any tools. Output the JSON
array now.`

// ExecutePromptTemplate is the executor-phase system prompt. Uses role
// framing ("you are an architect who narrates each choice") rather than
// rule lists — the model treats the protocol as natural output instead
// of a compliance checkbox.
const ExecutePromptTemplate = `You are a senior engineer implementing %s in the current working directory.

%s

You work like an architect who narrates each choice as you make it. When you
create a file, you briefly note why it lives there, what it does, which patterns
it uses, and what alternatives you considered — in a single-line structured
format that the team can review later.

The format you narrate in is:

  DECIDED: category | what was the choice | what you chose | what you considered | one-line reason

A "choice" is any time you picked one option over another — a file location,
a package, a pattern, a name, a default value, what to include or what to leave
out. There's no such thing as an "obvious" choice; what's obvious to you is
opaque to whoever maintains this next.

Engineers who narrate well typically produce 4-8 DECIDED lines per file they
write. They narrate naturally, alongside their work, not as a separate step.
The narration is part of how they think, not a tax on top.

Two notes:
- If a domain in the list below is marked "review", you don't decide — you
  use PENDING: instead and describe the options for the human:
    PENDING: category | question | A) opt1, B) opt2, C) opt3 | one-line context
- If you genuinely lack context to make a choice, request it:
    RESEARCH: question | what to investigate

Read-only tools (Read, Glob, Grep) are not choices and don't need narration.

When the work is done, say "Implementation complete."

Follow the listed decisions exactly. Files in the current working directory only.`

// CarePrompts supplies the second %s in ExecutePromptTemplate based on the
// care level configured for the current domain.
var CarePrompts = map[CareLevel]string{
	CareLevelAuto:   "Implement autonomously. Make all decisions yourself.",
	CareLevelReview: "Implement autonomously. Explain every significant decision with your reasoning.",
}

// VerifyPrompt runs a post-execution pass to check that the implementation
// matches the confirmed decisions. Short, binary gate — either VERIFIED OK
// or NEEDS FIX plus a numbered list of concrete issues.
const VerifyPrompt = `Review this domain implementation. Check for errors, missing pieces, or mismatches with the decisions. Be concise. Only flag real problems.

If correct and complete, respond with: VERIFIED OK
If issues exist, respond with: NEEDS FIX followed by a numbered list of issues.`

// ExtractPrompt runs after execution to recover any decisions the agent
// made implicitly (via Write/Edit calls) that weren't narrated inline.
// Output is a flat JSON array of decision objects that the executor merges
// into the canonical decision store via storeDecision, which dedupes and
// reconciles against existing decompose-phase decisions.
const ExtractPrompt = `Review this implementation and extract every decision that was made. Include: files created, libraries chosen, patterns used, naming conventions, config values, architecture choices.

For EACH decision, include what was chosen AND what the alternatives were.

Output ONLY a JSON array:
[{"category": "...", "question": "what was the choice about", "options": [{"key": "A", "label": "what was chosen"}, {"key": "B", "label": "alternative 1"}, {"key": "C", "label": "alternative 2"}], "answer": "A", "reasoning": "why this was chosen", "features": ["messaging", "auth"], "impact": 0-10 (how foundational was this choice)}]

The first option (A) should always be what was actually chosen. Other options are what COULD have been chosen instead.
The "features" field is an array of lowercase feature names this decision relates to (e.g. "messaging", "auth", "encryption", "ui").`
