package agent

import "os"

// CareLevel controls how much the executor explains and verifies.
type CareLevel string

// ExecutePromptForVariant returns the executor prompt template. The
// DEFER_EXEC_VARIANT env var lets the benchmark command swap in alternate
// prompts for A/B testing without rebuilding. The default
// (ExecutePromptTemplate) is the skill-based version that won the most
// recent comparison; "rules" is kept as a regression baseline.
func ExecutePromptForVariant() string {
	switch os.Getenv("DEFER_EXEC_VARIANT") {
	case "rules":
		return ExecutePromptVariantRules
	case "anchor":
		return ExecutePromptVariantAnchor
	default:
		return ExecutePromptTemplate
	}
}

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

// ExecutePromptTemplate — skill-based reframe (replacing the previous
// rule-heavy version). On a real Claude Code benchmark this version produced
// 65% more inline DECIDED reports (56 vs 34) than the previous rule-based
// prompt on the same task. The behaviour is framed as a role's natural
// output ("you narrate each choice as an architect would") rather than a
// list of prohibitions, which prevents the model from treating the protocol
// as a checkbox to surface-pattern-match.
//
// See PROMPT_FINDINGS.md for the experiment writeup.
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

// ExecutePromptVariantAnchor — anchors the protocol to specific tool events.
// Hypothesis: Claude is tool-driven, so framing the requirement as a tool-result
// reaction makes it structural rather than behavioral.
const ExecutePromptVariantAnchor = `You are implementing %s.

%s

PROTOCOL — non-negotiable, structural rule:

After EVERY Write or Edit tool result, the FIRST text of your next message
must begin with "DECIDED:". You may emit multiple DECIDED lines in a row.
Only after at least one DECIDED line may you call another tool.

DECIDED line format (one per line, pipe-separated):
DECIDED: category | question | chosen answer | alternatives | one-line reason

A "decision" is anything where you picked X over Y. Examples:
- Where to put a file (cmd/ vs root vs internal/)
- Which package to use (cobra vs flag vs os.Args)
- How to name something
- What pattern to follow (interface vs struct, switch vs map)
- Error handling style
- Default values

Use PENDING: instead of DECIDED: only when the domain is marked "review" in the
list below — those decisions go to the user.

Read-only tools (Read, Glob, Grep) do NOT require DECIDED lines.

When done, say "Implementation complete."

Files MUST be created in the CURRENT WORKING DIRECTORY. Never use /tmp.
Follow the listed decisions exactly.`

// ExecutePromptVariantRules — the previous rule-based prompt, kept as a
// regression baseline for A/B testing via DEFER_EXEC_VARIANT=rules. The
// new default ExecutePromptTemplate produced 65% more inline decisions than
// this version on a real Claude Code benchmark.
const ExecutePromptVariantRules = `You are implementing a software project based on confirmed decisions. Domain: %s

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

Rules:
- Do NOT batch. Output each DECIDED line immediately after the choice.
- Do NOT skip "obvious" choices. If you chose X over Y, document it.
- Do NOT re-ask questions already answered in the decisions list below.
- Each DECIDED/PENDING/RESEARCH must be on a SINGLE LINE.
- Read-only operations (Read, Glob, Grep) do not need decision lines.
- Domains marked "review" MUST use PENDING, never DECIDED.

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
