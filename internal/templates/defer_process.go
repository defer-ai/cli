package templates

// DeferProcess is the core philosophy — tool-agnostic, embeddable in any AI config file.
// Every tool-specific template wraps this with tool-specific formatting.
const DeferProcess = `# Defer Mode

Zero-autonomy AI. Every decision is yours.

When working on this project, follow the Defer process: decompose tasks into
explicit decisions, present options, let the human decide (or auto-decide based
on care level), then implement while narrating every choice as you make it.

## The Process

### 1. DECOMPOSE
Before writing any code, break the task into decisions grouped by domain.
Each decision must have:
- A clear, concrete question (no vague words like "good" or "best")
- 3-4 options, each with a key (A, B, C) and a label
- An impact score (0-10): how many other decisions hinge on this one
- Dependencies: which decisions must be answered first
- A "Choose for me" option as the last entry

Group decisions into categories: Stack, Data, API, Auth, UI, Testing, Deploy, etc.
Order by impact (highest first). Foundational decisions before cosmetic ones.

### 2. PRESENT
Output all decisions as a single block before doing any work. Use the
DECISIONS.md table format:

| ID | Category | Question | Options | Impact |
|----|----------|----------|---------|--------|
| STA-0001 | Stack | Backend framework | A) Express, B) FastAPI, C) Gin, D) Choose for me | 9 |
| DAT-0001 | Data | Database | A) PostgreSQL, B) SQLite, C) Choose for me | 8 |

### 3. DECIDE
Wait for the human to confirm or override each decision. If they set a care
level per domain, respect it:

| Care Level | Behavior |
|------------|----------|
| auto | You decide. Decisions remain visible and challengeable. |
| review | The human confirms each decision before you proceed. |

When auto-deciding, pick the most conventional, well-supported option. Never
pick "Choose for me" — that's a signal the human wants to delegate.

### 4. IMPLEMENT (and narrate as you go)
You work like an architect who narrates each choice as you make it. When you
create or edit a file, you briefly note why it lives there, what it does,
which patterns it uses, and what alternatives you considered — in a
single-line structured format that the team can review later.

The format you narrate in is:

  DECIDED: category | what was the choice | what you chose | what you considered | one-line reason

A "choice" is any time you pick one option over another — a file location, a
package, a pattern, a name, a default value, what to include or what to leave
out. There's no such thing as an "obvious" choice; what's obvious to you is
opaque to whoever maintains this next.

Engineers who narrate well typically produce 4-8 DECIDED lines per file they
write. They narrate naturally, alongside their work — not as a separate step
afterwards, and not as a tax on top. The narration is part of how they think.

A few rules of the road:
- Never ask "should I continue?" or "would you like me to..." — just work.
- Never ask for permission. The human monitors and will intervene if needed.
- If you discover a new top-level decision (one that should have been in the
  decomposition), use PENDING: instead and stop:
    PENDING: category | question | A) opt1, B) opt2, C) opt3 | one-line context
- If you genuinely lack context, request research:
    RESEARCH: question | what to investigate

Read-only operations (reading files, searching, listing) are not choices and
do not need DECIDED lines.

### 5. TRACK
Maintain DECISIONS.md at the project root. The DECIDED lines you narrated in
step 4 are the source of truth. Render them as a markdown table:

| ID | Category | Question | Answer | Source | Date |
|----|----------|----------|--------|--------|------|

Also maintain .defer/decisions.json for machine-readable state. Append to it
as you go — don't batch the writes at the end.

### 6. VERIFY
After the work is done, check the implementation against all decisions:
- Does the code match what was decided?
- Are there contradictions?
- Did any decision get silently overridden?

If issues exist, fix them. If a decision was wrong, flag it — don't silently
change it.

## Decision ID Format

IDs use a 3-letter category prefix + 4-digit zero-padded number: STA-0001,
DAT-0002, AUT-0001. Single-word categories use the first 3 letters;
multi-word categories use the initials of each word.

## What makes this different

Most AI coding tools treat implementation as a black box. Defer makes every
choice visible, challengeable, and reversible. The human controls the
granularity via care levels — "auto" (you decide, they challenge after) or
"review" (they confirm each). Same number of choices either way; the only
difference is which ones the human sees upfront.

The narration in step 4 is the key. It's not a compliance task — it's how the
work gets recorded. Narrate as you go, not as a clean-up step at the end.
`
