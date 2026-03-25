```
  ▄██████▄         ▄██████▄
  ██    ██         ██    ██     defer.sh
  ▀██████▀         ▀██████▀     zero-autonomy ai

           ████████             v0.1.0
```

[![npm version](https://img.shields.io/npm/v/@defer/cli.svg)](https://www.npmjs.com/package/@defer/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/defer-ai/cli/blob/main/LICENSE)

AI keeps making choices you didn't ask for. It picks your tech stack, your file structure, your error messages. You don't find out until the code is wrong.

Defer makes the AI ask first, then execute. Slow upfront. Zero rework later.

## Quick Start

```bash
# Requires Claude Code installed and logged in
npx @defer/cli "build a todo app"
```

That's it. Defer wraps Claude Code, decomposes your task into decisions, and asks before writing a single line.

## What Happens

1. You give it a task
2. The AI decomposes it into categorized decisions with selectable options
3. You set how much you care about each domain (skip to paranoid)
4. You answer decisions one by one (or say "choose for me")
5. The AI executes with full context of every choice
6. Every decision the AI makes during execution is tracked as an assumption

```
defer > build a todo app

[? ?]  How much do you care about each area?

  > Stack             ██░░░  medium    3 decisions
    Data              █████  paranoid  2 decisions
    API               ░░░░░  skip      2 decisions
    UI                ██░░░  medium    3 decisions

---

[◉ ◉]  1/8  Stack  STACK-001

  Backend language and framework?

  Determines ecosystem and deployment model

   > A) Node.js (TypeScript)
     B) Python (FastAPI)
     C) Go
     D) Choose for me
```

## Features

**Decision workflow:**
- Categorized decisions with selectable options
- Per-domain care levels: skip, low, medium, high, paranoid
- "Choose for me" delegates to AI with logged reasoning
- Undo last answer, ask about tradeoffs, revisit by ID

**Assumption tracking:**
- Every choice the AI makes during execution is tagged
- `[ASSUMPTION naming: Used camelCase because framework convention]`
- Assumptions logged alongside user decisions in DECISIONS.md
- Nothing is invisible. `/decisions` shows everything.

**Profiles and history:**
- `/profile save my-stack` saves your decisions as a reusable template
- `/profile use my-stack` pre-fills matching decisions on new projects
- `/history` shows previous sessions with cost and duration

**Session management:**
- Decisions persisted to `.defer/decisions.json` + human-readable `DECISIONS.md`
- Quit and resume where you left off
- Cost and token tracking in the status bar

## Commands

```
defer                          Interactive mode
defer "build auth"             Start with a task

/help                          All commands
/model <sonnet|opus|haiku>     Switch model
/decisions                     View all decisions inline
/revisit STACK-001             Change a specific decision
/profile save|use|list         Manage reusable decision templates
/history                       Previous sessions
/export                        Decision record as markdown table
/cost                          Session cost and tokens
/clear                         Clear output
/quit                          Exit (auto-saves to history)
```

**In the decision view:**
```
↑↓   navigate options
enter confirm selection
t     type custom answer
u     undo last answer
w     explain tradeoffs of highlighted option
a     ask a question about the decision
c     change an answered decision
esc   back to stream
```

## How It Works

Defer spawns `claude` as a subprocess using your existing Claude Code authentication. No API key needed, no extra setup.

The system prompt instructs Claude to decompose tasks into structured decisions (as a parseable JSON block), and to tag every autonomous choice during execution as an assumption. The TUI renders the decisions as a selection interface, persists everything to disk, and streams execution output.

## Decision Record

After a session, you have `DECISIONS.md`:

```markdown
## Decisions

| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|
| STACK-001 | Stack | Backend language? | Node.js (TypeScript) | 2026-03-25 |
| DATA-001 | Data | Database? | DELEGATED: PostgreSQL | 2026-03-25 |

## Assumptions

| ID | Category | What was decided | Reasoning |
|----|----------|------------------|-----------|
| NAMI-001 | naming | camelCase for routes | framework convention |
| ERRO-001 | error | 422 for validation | more semantically correct |
```

## Install

```bash
# Run directly
npx @defer/cli

# Or install globally
npm install -g @defer/cli
defer "build something"
```

Requires Claude Code (`npm install -g @anthropic-ai/claude-code && claude login`).

## Development

```bash
git clone https://github.com/defer-ai/cli
cd cli
npm install
npm run build
npm test
```

## Website

[defer.sh](https://defer.sh) - Next.js app in `web/`, deployed via Vercel.

## License

MIT
