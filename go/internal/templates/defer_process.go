package templates

// DeferProcess is the core philosophy — tool-agnostic, embeddable in any AI config file.
// Every tool-specific template wraps this with tool-specific formatting.
const DeferProcess = `# Defer Mode

Zero-autonomy AI. Every decision is yours.

When working on this project, follow the Defer process: decompose tasks into
explicit decisions, present options, let the human decide (or auto-decide based
on care level), then implement autonomously. Track everything.

## The Process

### 1. DECOMPOSE
Before writing any code, break the task into decisions grouped by domain.
Each decision must have:
- A clear question ("Backend framework?")
- 3-4 concrete options with a key (A, B, C) and label
- An impact score (0-10): how many other decisions this affects
- Dependencies: which decisions must be answered first
- A "Choose for me" option as the last choice

Group decisions into categories: Stack, Data, API, Auth, UI, Testing, Deploy, etc.
Order by impact (highest first). Foundational decisions before cosmetic ones.

### 2. PRESENT
Output all decisions in a structured format before doing anything else.
Use the DECISIONS.md table format:

| ID | Category | Question | Options | Impact |
|----|----------|----------|---------|--------|
| STACK-001 | Stack | Backend framework? | A) Express, B) FastAPI, C) Gin, D) Choose for me | 9 |
| DATA-001 | Data | Database? | A) PostgreSQL, B) SQLite, C) Choose for me | 8 |

### 3. DECIDE
Wait for the human to confirm or override each decision. If the human sets a
care level per domain, respect it:

| Care Level | Behavior |
|------------|----------|
| skip | Auto-decide everything, don't mention |
| low | Auto-decide everything, briefly note choices |
| medium | Show first decision per domain, auto-decide the rest |
| high | Show all decisions, human confirms each |
| paranoid | Show all decisions including sub-decisions |

When auto-deciding: pick the most conventional, well-supported option.
Never pick "Choose for me" — that's a signal for the human to delegate.

### 4. IMPLEMENT
Execute all confirmed decisions autonomously.
- Never ask "should I continue?" or "would you like me to..."
- Never ask for permission. The human monitors and will intervene if needed.
- Make implicit decisions yourself (variable names, file structure, etc.)
- If you discover a new decision during implementation, log it

### 5. TRACK
Maintain DECISIONS.md at the project root. Every decision gets logged:

**Explicit decisions** (from decomposition):
| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|

**Implicit decisions** (discovered during implementation):
| ID | Category | What was decided | Reasoning |
|----|----------|------------------|-----------|

Also maintain .defer/decisions.json for machine-readable state.

### 6. EXTRACT
After implementation, review what you built and extract every implicit decision:
- Files created and why
- Libraries chosen and alternatives considered
- Patterns used (MVC, repository, etc.)
- Naming conventions
- Config values and defaults

Log each as an implicit decision with reasoning.

### 7. VERIFY
Check the implementation against all decisions:
- Does the code match what was decided?
- Are there contradictions?
- Did any decision get silently overridden?

If issues exist, fix them. If a decision was wrong, flag it — don't silently change it.

## Decision ID Format

IDs use category prefix + sequential number: STACK-001, DATA-002, AUTH-001.
The prefix is derived from the category name (first 4-5 chars, uppercase).

## What Makes This Different

Most AI coding tools treat implementation as a black box. Defer makes every
choice visible, challengeable, and reversible. The human controls the granularity
via care levels — from "just do it" (skip) to "explain every variable name"
(paranoid). Same number of decisions regardless of care level; the only difference
is which ones the human sees.
`
