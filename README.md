# defer

Zero-autonomy AI. Every decision is yours.

Defer decomposes your task into a tree of decisions, lets you set care levels per domain (skip through paranoid), auto-decides the rest, then implements everything while you watch, chat, and challenge in real-time. Every choice -- from framework to variable name -- is tracked, reversible, and exportable.

## Install

**From release (recommended):**
```bash
# macOS / Linux
curl -sSL https://github.com/defer-ai/cli/releases/latest/download/defer_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv defer /usr/local/bin/
```

**From source:**
```bash
cd go && go build -ldflags "-s -w" -o defer .
sudo mv defer /usr/local/bin/
```

## Quick Start

```bash
defer "build a secret sharing tool"    # new project
defer scan                              # onboard existing project
defer                                   # resume last session
```

## How It Works

```
You: "build a secret sharing tool"
         |
    1. DECOMPOSE
         | AI reads your project, extracts high-level decisions
         | Groups by category (Stack, Security, API, UI, ...)
         | Assigns impact score 0-10 per decision
         | Each decision has 3-4 concrete options
         |
    2. PRIORITIZE
         | You set care levels per domain
         | skip/low/medium -> auto-decided (gray in tree)
         | high/paranoid   -> you must confirm (yellow)
         |
    3. DECIDE
         | Navigate the decision tree
         | Inspect, challenge, override any decision
         | Chat with @ID references for context
         |
    4. IMPLEMENT
         | Executor: plan -> implement -> verify -> extract
         | Autonomous execution, no "should I continue?" prompts
         | If you change a decision mid-run, it re-implements
         |
    5. TRACK
         | DECISIONS.md tracks everything
         | .defer/decisions.json for machine-readable state
         | Every implicit decision extracted and logged
```

## The Decision Tree

The tree is the main view. On wide terminals (>100 columns), selecting a decision shows a split-pane with the detail on the right. On narrow terminals, detail opens full-screen.

```
  defer                               3/8 decisions -- 5 pending
  +-------------------------------------------------------------------+
  |                                                                   |
  |  Stack                                                            |
  |  > * STACK-001  Backend framework?         -> Go with Gin         |
  |    * STACK-002  Database?                  -> PostgreSQL          |
  |    * STACK-003  ORM?                       -> sqlc                |
  |                                                                   |
  |  Security                                                         |
  |    + SECU-001   Encryption method?         -> AES-256-GCM        |
  |    o SECU-002   Key management?            (pending)             |
  |                                                                   |
  |  UI                                                               |
  |    * UI-001     Component library?         -> shadcn/ui          |
  |    * UI-002     Styling approach?          -> Tailwind CSS       |
  |                                                                   |
  +-- Stack: executing  Security: planning  UI: done -----------------+
  |  @SECU-002 what are the tradeoffs?                               |
  +-- enter inspect  / search  tab chat  ctrl+c x2 quit -------------+
```

- `o` pending (yellow) -- needs your input
- `+` confirmed (green) -- you decided
- `*` auto-decided (gray) -- agent decided, challengeable
- Impact bars: `|||` high (red), `||` medium (yellow), `|` low (dim)

## TUI Keybindings

### Decision Tree

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate decisions |
| `enter` | Inspect decision (split-pane on wide terminals) |
| `/` | Search/filter decisions by ID, category, or question |
| `tab` | Open conversation panel |
| `ctrl+c` x2 | Quit |

### Decision Detail

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate options |
| `enter` | Confirm selected option |
| `c` | Type a custom answer |
| `s` | Shuffle -- generate new options via AI |
| `w` | Why? -- explain tradeoffs for current option |
| `a` | Ask -- ask a freeform question about this decision |
| `q` / `esc` | Back to tree |

### Conversation Panel

| Key | Action |
|-----|--------|
| `enter` | Send message |
| `@` + type | Autocomplete decision IDs (tab to cycle) |
| `@ID change to X` | Directly change a decision |
| `@ID why?` | Ask about a specific decision |
| `tab` | Back to tree (or cycle autocomplete) |
| `esc` | Back to tree |

### Care Level Picker

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate domains |
| `h` / `l` or arrows | Adjust care level |
| `enter` | Confirm all levels |

All keybindings are configurable via `~/.defer/keybindings.json`.

## Care Levels

| Level    | What happens                             |
|----------|------------------------------------------|
| skip     | All auto-decided, hidden from tree       |
| low      | All auto-decided, visible                |
| medium   | First decision per category shown, rest auto |
| high     | All decisions shown, you confirm each    |
| paranoid | All decisions + sub-decisions shown       |

Same number of decisions regardless of care level. The only difference: skip/low/medium get auto-answered, high/paranoid stay pending for you.

## Search

Press `/` in the tree view to filter decisions. Type any part of a decision's ID, category, or question. Press `enter` to lock the filter (navigate within filtered results), or `esc` to clear.

## Provider Support

Auto-detected from environment, or set explicitly:

| Provider         | How to enable                                |
|------------------|----------------------------------------------|
| Claude Code      | Default (free with subscription)             |
| OpenAI           | `export OPENAI_API_KEY=sk-...`               |
| Groq             | `export GROQ_API_KEY=gsk_...`                |
| Mistral          | `--provider mistral --api-key ...`           |
| Together         | `--provider together --api-key ...`          |
| DeepInfra        | `--provider deepinfra --api-key ...`         |
| Cerebras         | `--provider cerebras --api-key ...`          |
| OpenRouter       | `--provider openrouter --api-key ...`        |
| Ollama           | `--provider ollama --model llama3.1`         |
| Any OpenAI-compat| `--provider <url> --api-key <key>`           |

## Commands

```
defer "task"               Start a new project with the given task
defer                      Resume the last session in the current directory
defer scan                 Discover decisions in an existing project
defer init [target]        Scaffold config for: claude-code, cursor, copilot, codex, universal
defer sessions list        List all sessions in directory tree
defer sessions delete      Delete .defer/ in current directory
defer sessions export      Print DECISIONS.md to stdout
defer --debug "task"       Headless mode (no TUI, prints to stdout)
defer --model opus "task"  Use a specific model (sonnet, opus, haiku, or ID)
defer --version            Show version
```

## Configuration

Defer uses a three-level configuration cascade. Later sources override earlier ones.

### Settings Cascade

| Priority | Location | Scope |
|----------|----------|-------|
| 1 (lowest) | `~/.defer/config.json` | Global defaults |
| 2 | `.defer/config.json` | Project overrides |
| 3 (highest) | CLI flags | Session overrides |

### Config File Format

```json
{
  "model": "sonnet",
  "provider": "openai",
  "defaultCare": "medium",
  "domainCare": {
    "Security": "paranoid",
    "UI": "low"
  },
  "hooks": {
    "post-execute": [
      { "command": "npm test" },
      { "url": "https://hooks.slack.com/..." }
    ]
  },
  "skills": {
    "dirs": ["./custom-skills"]
  }
}
```

### Custom Keybindings

Create `~/.defer/keybindings.json` to override default bindings:

```json
{
  "navigate.up": ["k", "up"],
  "navigate.down": ["j", "down"],
  "inspect": ["enter"],
  "back": ["q", "esc"],
  "search": ["/"],
  "chat": ["tab"],
  "custom": ["c"],
  "shuffle": ["s"],
  "why": ["w"],
  "ask": ["a"]
}
```

## Lifecycle Hooks

Hooks run at specific points in the decision/execution lifecycle. Configure in `config.json`:

| Hook | When it fires |
|------|---------------|
| `pre-decision` | Before a decision is auto-decided |
| `post-decision` | After a decision is confirmed |
| `pre-execute` | Before the executor starts implementing |
| `post-execute` | After the executor completes |
| `decision-changed` | When a user revises a decision |

Hook types:
- **Command**: runs a shell command (e.g., `npm test`, `make lint`)
- **Webhook**: POSTs JSON to a URL (e.g., Slack notification)

Environment variables available to hooks: `DEFER_EVENT`, `DEFER_DECISION_ID`, `DEFER_DECISION_ANSWER`, `DEFER_CWD`.

## Custom Skills (Prompt Overrides)

The defer process is built from 6 skills (prompts): decompose, plan, execute, extract, verify, scan. You can override any of them per-project.

Create `.defer/skills/decompose.md`:

```markdown
---
name: decompose
description: Custom decomposition for this project
---
Your custom prompt here. This replaces the default decompose prompt.
Focus on security decisions first...
```

Skills are discovered by walking up from the current directory. Project-level skills override parent-level skills, which override built-in defaults.

## Tool Permissions

Care levels also control what the executor is allowed to do without prompting:

| Care Level | Read files | Write files | Run commands |
|------------|-----------|-------------|-------------|
| skip | auto | auto | auto |
| low | auto | auto | auto |
| medium | auto | auto | prompt |
| high | auto | prompt | prompt |
| paranoid | prompt | prompt | prompt |

## Portable Mode (No CLI Required)

You don't need the defer CLI to follow the defer process. Run `defer init` to scaffold a config file for your AI tool:

```bash
defer init claude-code    # Creates CLAUDE.md
defer init cursor         # Creates .cursorrules
defer init copilot        # Creates .github/copilot-instructions.md
defer init codex          # Creates AGENTS.md
defer init universal      # Creates DEFER.md (copy into any tool's config)
```

The generated file contains the full defer philosophy (decompose, present, decide, implement, track, extract, verify). The AI tool reads it and follows the process natively.

## Architecture

```
go/
  main.go                        Entry point + version injection
  cmd/
    root.go                      CLI flags, provider resolution, TUI launch
    scan.go                      defer scan subcommand
    sessions.go                  sessions list/delete/export
    init.go                      defer init scaffolding (5 tool targets)
    debug.go                     Headless debug mode (--debug)
  internal/
    agent/
      agent.go                   Decomposition agent (parses decisions from LLM)
      executor.go                Domain executor (plan/execute/verify/extract)
      manager.go                 Coordinator (auto-decide, sync, launch, cancel)
      prompts.go                 System prompts (decompose, plan, execute, etc.)
      events.go                  Event types for agent -> TUI communication
    api/
      provider.go                Provider interface + auto-detection (9 providers)
      claude_code.go             Claude Code subprocess provider
      openai.go                  OpenAI-compatible HTTP provider
      toolexec.go                Tool execution (file I/O, shell commands)
    config/
      config.go                  Settings cascade (global -> project -> CLI)
    decision/
      decision.go                Core types (Decision, DecisionStore, Options)
      store.go                   Persistence (.defer/decisions.json)
      id.go                      Category-prefix ID generation (STACK-001)
      markdown.go                DECISIONS.md generation
    hooks/
      hooks.go                   Lifecycle hooks (bash commands + webhooks)
    keybindings/
      keybindings.go             Configurable keybindings (~/.defer/keybindings.json)
    mcp/
      client.go                  MCP client (JSON-RPC 2.0 over stdio)
      config.go                  MCP server configuration
    permissions/
      permissions.go             Care-level-aware tool permissions
    skills/
      loader.go                  Skill file loading (.md with YAML frontmatter)
      discovery.go               Dynamic skill directory discovery + file watching
    templates/
      defer_process.go           The defer philosophy (tool-agnostic spec)
      templates.go               Tool-specific config templates (5 targets)
    tui/
      app.go                     Root Bubbletea model, message routing
      tree.go                    Decision tree + split-pane detail + chat + search
      priorities.go              Care level picker
      welcome.go                 Welcome screen + task input (bubbles/textinput)
      mascot.go                  Pixel art mascot with mood animations
      styles.go                  Lip Gloss styles + bordered box renderer
      notifications.go           Priority-based notification system
      messages.go                All tea.Msg types + safeSend helper
```

## Development

```bash
cd go
go build ./...              # build
go test ./... -count=1      # run tests (13 packages)
go test ./... -race         # tests with race detector
go vet ./...                # static analysis
go build -o defer .         # build binary
```

## License

MIT
