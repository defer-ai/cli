# defer

Zero-autonomy AI. Every decision is yours.

Defer decomposes your task into a tree of decisions, lets you set care levels per domain (skip through paranoid), auto-decides the rest, then implements everything while you watch, chat, and challenge in real-time. Every choice -- from framework to variable name -- is tracked, reversible, and exportable.

## Quick Start

```bash
cd go && go build -o defer .
./defer "build a secret sharing tool"
./defer scan   # onboard an existing project
```

## How It Works

```
You: "build a secret sharing tool"
         |
    Decomposer (Claude Code / OpenAI-compatible)
         | reads project, extracts high-level decisions
         | groups by category (Stack, Security, API, UI, ...)
         | assigns impact 0-10 per decision
         |
    You set care levels per domain
         | skip/low/medium -> auto-decided (gray in tree)
         | high/paranoid   -> you must confirm (yellow)
         |
    Decision Tree (main view)
         | navigate, inspect, challenge any decision
         | conversation panel: chat with @ID references
         | shuffle options, ask "why?", override answers
         |
    Executor (single provider, full tool access)
         | plan -> implement -> verify -> extract implicit decisions
         | if you change a decision mid-run, it re-implements
         |
    Done. DECISIONS.md tracks everything.
```

## The Decision Tree

```
  defer                           3/8 decisions  -- 5 pending
  +-----------------------------------------------------------------+
  |                                                                 |
  |  Stack                                                          |
  |    > * STACK-001  Backend framework?       -> Go with Gin       |
  |      * STACK-002  Database?                -> PostgreSQL        |
  |      * STACK-003  ORM?                     -> sqlc              |
  |                                                                 |
  |  Security                                                       |
  |    + SECU-001   Encryption method?        -> AES-256-GCM       |
  |    o SECU-002   Key management?           (pending)            |
  |                                                                 |
  |  UI                                                             |
  |    * UI-001     Component library?        -> shadcn/ui         |
  |    * UI-002     Styling approach?         -> Tailwind CSS      |
  |                                                                 |
  +-- conversation --------------------------------------------------+
  |  > @SECU-002 what are the tradeoffs?                           |
  |  * AWS KMS gives you managed rotation but adds vendor lock-in  |
  +-- tab chat  enter inspect  ctrl+c x2 quit ----------------------+
```

- `o` pending (yellow) -- needs your input
- `+` confirmed (green) -- you decided
- `*` auto-decided (gray) -- agent decided, challengeable
- Impact bars: `|||` high (red), `||` medium (yellow), `|` low (dim)

## Care Levels

| Level    | What happens                             |
|----------|------------------------------------------|
| skip     | All auto-decided, hidden from tree       |
| low      | All auto-decided, visible                |
| medium   | First decision per category shown, rest auto |
| high     | All decisions shown, you confirm each    |
| paranoid | All decisions + sub-decisions shown       |

Same number of decisions regardless of care level. The only difference: skip/low/medium get auto-answered, high/paranoid stay pending for you.

## Provider Support

Auto-detected from environment, or set explicitly:

| Provider         | How to enable                                |
|------------------|----------------------------------------------|
| Claude Code      | Default (free with subscription)             |
| OpenAI           | `export OPENAI_API_KEY=sk-...`               |
| Groq             | `export GROQ_API_KEY=gsk_...`                |
| Mistral          | `--provider mistral --api-key ...`           |
| Together         | `--provider together --api-key ...`          |
| Ollama           | `--provider ollama --model llama3.1`         |
| Any OpenAI-compat| `--provider <url> --api-key <key>`           |

## Commands

```
defer "task"               Start a new project with the given task
defer                      Resume the last session in the current directory
defer scan                 Discover decisions in an existing project
defer sessions list        List all sessions in directory tree
defer sessions delete      Delete .defer/ in current directory
defer sessions export      Print DECISIONS.md to stdout
defer init [target]        Scaffold config files (claude-code, universal)
defer --debug "task"       Headless mode (no TUI, prints to stdout)
defer --mcp "task"         Enable MCP server connections
defer --model opus "task"  Use a specific model (sonnet, opus, haiku, or ID)
```

## Architecture

```
go/
  main.go                    Entry point
  cmd/
    root.go                  CLI flags, provider resolution, TUI launch
    scan.go                  defer scan subcommand
    sessions.go              sessions list/delete/export
    init.go                  defer init scaffolding
    debug.go                 Headless debug mode (--debug)
  internal/
    api/
      provider.go            Provider interface + auto-detection
      claude_code.go         Claude Code subprocess provider
      openai.go              OpenAI-compatible HTTP provider
      toolexec.go            Tool execution (file I/O, shell commands)
    agent/
      agent.go               Decomposition agent (parses decisions from LLM)
      executor.go            Domain executor (plan/execute/verify/extract)
      manager.go             Coordinator (auto-decide, sync, launch)
      prompts.go             All system prompts
      events.go              Event types for agent -> TUI communication
    decision/
      decision.go            Core types (Decision, DecisionStore)
      store.go               Persistence (.defer/decisions.json)
      id.go                  Category-prefix ID generation (STACK-001)
      markdown.go            DECISIONS.md generation
    tui/
      app.go                 Root Bubbletea model, message routing
      tree.go                Decision tree + detail + chat panel
      priorities.go          Care level picker
      welcome.go             Welcome screen + task input
      mascot.go              Pixel art mascot (half-block characters)
      styles.go              Lip Gloss styles + bordered box renderer
      messages.go            All tea.Msg types
    mcp/
      client.go              MCP client (Model Context Protocol)
      config.go              MCP server configuration
```

## Development

```bash
cd go
go build ./...              # build
go test ./... -count=1      # run tests
go vet ./...                # static analysis
go build -o defer .         # build binary
```

## License

MIT
