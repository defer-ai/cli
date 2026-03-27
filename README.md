# defer

Zero-autonomy AI. Every decision is yours.

AI keeps making choices you didn't ask for. It picks your tech stack, your file structure, your error messages. You don't find out until the code is wrong.

Defer decomposes your project into decisions, lets you set how much you care about each domain, auto-decides the rest, and implements everything while you watch and challenge in real-time.

## Quick Start

```bash
# Build from source (requires Go 1.22+)
cd go && go build -o defer . && ./defer "build a todo app"

# Or with Claude Code installed (no API key needed)
defer "build a secret sharing tool"

# Or with Anthropic API key (enables tool interception)
export ANTHROPIC_API_KEY=sk-ant-...
defer "build a dream journal"

# Debug mode (no TUI, prints everything to stdout)
defer --debug "build a todo app"
```

## How It Works

```
You: "build a secret sharing tool"
         │
    Decomposer (Sonnet)
         │ extracts 8 high-level decisions
         │ across Stack, Security, API, UI, etc.
         │
    Decision Swarm (Haiku × N, parallel)
         │ each domain gets 8-12 sub-decisions
         │ ~50 total decisions in seconds
         │
    You set care levels per domain
         │ skip → auto-decided (gray in tree)
         │ paranoid → you must confirm (yellow)
         │
    Decision Tree (main view)
         │ watch decisions appear in real-time
         │ navigate, inspect, challenge any decision
         │ change your mind at any point
         │
    Executor (Sonnet, full tool access)
         │ implements everything based on decisions
         │ if you change a decision, it re-implements
         │
    Done. DECISIONS.md tracks everything.
```

## The Decision Tree

The tree is the only view. Decisions stream in as agents work.

```
  defer
  29/29 decisions  ○ 3 pending

  Stack
    ▪ STACK-001  Backend framework       → Next.js
    ▪ STACK-002  Database                → PostgreSQL
    ▪ STACK-003  ORM                     → Prisma

  Security
    ✓ SECU-001  Encryption method       → AES-256-GCM
    ○ SECU-002  Key management          → (pending)

  UI
    ▪ UI-001    Component library       → shadcn/ui
    ▪ UI-002    Styling approach        → Tailwind CSS
```

- `○` pending (yellow) -- needs your input
- `✓` confirmed (green) -- you decided
- `▪` auto-decided (gray) -- agent decided, challengeable

### Controls

```
Tree:
  ↑↓       navigate decisions
  enter    inspect / pick option
  tab      live agent feed
  ctrl+c×2 quit

Detail:
  ↑↓       navigate options
  enter    confirm selection
  c        custom answer (free text)
  s        shuffle (AI generates new options)
  w        why (explain tradeoffs)
  a        ask (question about this decision)
  q        back to tree
```

## Care Levels

After decomposition, you set how much you care about each domain:

```
  Stack     ░░░░░ skip      → all auto-decided
  Security  █████ paranoid  → you confirm everything
  API       ██░░░ medium    → auto-decided
  UI        ████░ high      → you confirm everything
```

Same number of decisions regardless of care level. The only difference: skip/low/medium get auto-answered, high/paranoid stay pending for you.

## Architecture

```
go/
  cmd/
    root.go          # CLI entry, launches Bubbletea or debug mode
    init.go          # defer init subcommand
    debug.go         # headless debug mode (--debug flag)
  internal/
    api/
      client.go      # Anthropic SDK wrapper
      claude_code.go # Claude Code subprocess provider
      tools.go       # Tool schemas (Read, Write, Edit, Bash, Glob, Grep)
      toolexec.go    # Tool execution (file I/O, shell commands)
      stream.go      # Agent loop with tool interception
    agent/
      agent.go       # Decomposition agent
      swarm.go       # Haiku subagent swarm (parallel domain expansion)
      executor.go    # Implementation executor
      manager.go     # Coordinator
      prompts.go     # All system prompts
    decision/
      decision.go    # Core types
      store.go       # Persistence (.defer/decisions.json)
      id.go          # Timestamp-based ID generation (no collisions)
      markdown.go    # DECISIONS.md generation
    tui/
      app.go         # Root Bubbletea model (Elm architecture)
      tree.go        # Decision tree + detail view + feed
      priorities.go  # Domain priority picker
      welcome.go     # Welcome screen
      mascot.go      # Pixel art mascot (half-block characters)
      styles.go      # Lip Gloss styles
      messages.go    # All tea.Msg types
```

### Two Provider Modes

| | Direct API | Claude Code Subprocess |
|---|---|---|
| Auth | `ANTHROPIC_API_KEY` | `claude login` (existing subscription) |
| Tool interception | Every Write/Edit/Bash logged as decision | Tool calls intercepted from stream-json |
| Cost | Pay per token | Free with subscription |
| Tool permissions | Controlled per care level | Claude Code handles internally |

## Decision Record

After a session:

```markdown
# DECISIONS.md

> Task: build a secret sharing tool

## Decisions

| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|
| STACK-001 | Stack | Backend framework? | Next.js | 2026-03-26 |
| SECU-001 | Security | Encryption? | AES-256-GCM | 2026-03-26 |

## AI Choices

| ID | Category | What was decided | Reasoning |
|----|----------|------------------|-----------|
| STACK-002 | Stack | TypeScript strict mode | Type safety for crypto operations |
```

## Development

```bash
cd go
go build ./...        # build
go test ./...         # run tests (~100 tests)
go test -race ./...   # race condition check
go build -o defer .   # build binary
```

## Website

[defer.sh](https://defer.sh) -- Next.js app in `web/`.

## License

MIT
