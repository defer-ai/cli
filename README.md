# defer

Zero-autonomy AI. Every decision is yours.

## What is defer

Defer decomposes your task into a tree of decisions, lets you set care levels per domain (auto or review), auto-decides the rest, then implements everything while you watch, chat, and challenge in real-time. Every choice -- from framework to variable name -- is tracked, reversible, and exportable. The agent never asks questions as text; all ambiguity becomes decisions with concrete options.

## Install

**From source (recommended):**

```bash
git clone https://github.com/defer-ai/cli.git && cd cli/go
go build -ldflags "-s -w" -o defer .
sudo mv defer /usr/local/bin/
```

**From release:**

Download the latest binary for your OS/arch from [GitHub Releases](https://github.com/defer-ai/cli/releases).

## Quick Start

```bash
defer "build a secret sharing tool"    # new project
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
         | If anything is unclear, it becomes a decision -- never a question
         |
    2. PRIORITIZE
         | Care levels are set inline when the first decisions arrive
         | auto -> agent decides, you challenge (gray in tree)
         | review -> you confirm each decision (yellow)
         |
    3. DECIDE
         | Navigate the decision tree (tab to switch panel focus)
         | Inspect, challenge, override any decision
         | Chat with @ID references for context
         |
    4. IMPLEMENT
         | Executor: plan -> implement -> verify -> extract
         | Autonomous execution, no "should I continue?" prompts
         | If you change a decision mid-run, it re-implements
         | Changing a high-impact decision invalidates dependents
         |
    5. TRACK
         | DECISIONS.md tracks everything
         | .defer/decisions.json for machine-readable state
         | Every implicit decision extracted and logged
```

## The Conversation

The conversation is the primary interface. When you launch defer, you land in a full-screen chat where the agent streams everything -- decomposition progress, execution output, tool calls, and status updates. You type naturally, reference decisions with `@ID`, and the agent responds inline.

The conversation view is where you:

- Describe your task or ask follow-up questions
- Reference decisions with `@STA-0001` to discuss, challenge, or change them
- Use `@ID change to X` to revise a decision directly
- Use `@ID why?` to ask about a specific decision's tradeoffs
- Watch the agent work (tool calls, file writes, test runs stream in real-time)

Press `tab` to switch focus between panels.

## Side-by-Side Layout

On wide terminals (>80 columns), the decision tree and chat are always visible side by side. `tab` switches focus between the left (tree) and right (chat) panels. On narrow terminals, only the focused panel is shown.

The right panel has two regions: the chat log on top, and the pending resolver at the bottom. The resolver shows pending decisions as a wizard (e.g., 1/2, 2/3) so you can resolve them inline without leaving the chat.

```
┌── Decision Tree ──────┐ ┌── Chat ────────────────┐
│ Stack                 │ │ > build a todo app      │
│   ▪ STA-0001 Lang?   │ │                         │
│   ▪ STA-0002 FW?     │ │ ● Glob(files **/*)      │
│ Auth                  │ │ Found 6 decisions       │
│   ○ AUT-0001 Method?  │ ├─────────────────────────┤
│                       │ │ Pending 1/2             │
│                       │ │ ○ Auth method?           │
│                       │ │ > A) JWT  B) Session     │
└───────────────────────┘ └─────────────────────────┘
```

Status indicators:

- `o` pending (yellow) -- needs your input
- `+` confirmed (green) -- you decided
- `*` auto-decided (gray) -- agent decided, challengeable

Impact bars: `|||` high (red), `||` medium (yellow), `|` low (dim)

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `tab` | Switch focus between tree panel (left) and chat panel (right) |
| `ctrl+q` | Quit |
| `ctrl+c` | Shows warning (press `ctrl+q` to actually quit) |

### Tree Panel (left)

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate decisions |
| `enter` | Inspect decision |
| `/` | Filter decisions by ID, category, or question |
| `f` or `ctrl+f` | Find and jump to a decision, category, or feature |
| `g` | Toggle grouping between category and feature |

### Decision Detail

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate options |
| `enter` | Confirm selected option |
| `c` | Type a custom answer |
| `s` | Shuffle -- generate new options via AI |
| `w` | Why? -- explain tradeoffs for current option |
| `a` | Ask a freeform question about this decision |
| `f` | Edit feature tags (comma-separated) |
| `q` / `esc` | Back to tree |

### Chat Panel (right)

| Key | Action |
|-----|--------|
| `enter` | Send message |
| `@` + type | Autocomplete decision IDs |
| `tab` | Cycle autocomplete (when completions visible), otherwise switch panel |
| `ctrl+o` | Toggle expand/collapse on last agent topic |
| `j` / `k` | Navigate resolver options (when input is empty) |
| `n` / `p` | Cycle through pending decisions in resolver |

All keybindings are configurable via `~/.defer/keybindings.json`.

## Pending Resolver

The pending resolver is the bottom section of the right (chat) panel. When the agent discovers decisions that require your input, they appear here as a wizard -- one at a time, showing progress (e.g., "Pending 1/3").

Each pending decision shows its question and concrete options. Use `j`/`k` (when the chat input is empty) to navigate options and `enter` to confirm. Use `n`/`p` to cycle to the next or previous pending decision. Care levels are also set inline here when the first decisions arrive -- no separate picker screen.

## Care Levels

| Level | What happens |
|--------|------------------------------------------|
| auto | Agent decides everything. Decisions visible in tree, challengeable anytime. |
| review | You confirm each decision before execution proceeds. |

Same decisions either way. The difference: auto answers them for you (you challenge after), review leaves them pending (you confirm first).

## Features

Decisions can be tagged with features (e.g., "messaging", "auth", "encryption"). Features provide a second axis for organizing decisions beyond categories.

**Feature tagging.** Press `f` in the decision detail view to edit a decision's feature tags. Enter comma-separated values (e.g., `auth, onboarding`). Feature IDs use the `#` prefix and follow the same 3-letter prefix rules as categories: `#MSG`, `#AUT`, `#ENC`.

**Grouping.** Press `g` in the tree view to toggle between grouping by category (default) and grouping by feature.

**Search.** Press `/` to filter decisions by any part of their ID, category, or question text. Press `enter` to lock the filter and navigate within results. Press `esc` to clear.

**Jump.** Press `f` or `ctrl+f` in the tree view to open a jump search. Type to find decisions, categories, or features by name, then press `enter` to jump directly to the match.

## ID Scheme

Defer uses prefixed IDs to reference decisions and features throughout the system.

**Decision IDs** use the `@` prefix followed by a 3-letter category code and a 4-digit sequence number:

```
@STA-0001    (Stack)
@SEC-0001    (Security)
@DAT-0001    (Data)
```

**Feature IDs** use the `#` prefix followed by a 3-letter code:

```
#MSG    (messaging)
#AUT    (auth)
#ENC    (encryption)
```

**Prefix rules:**

- Single-word category: first 3 letters, uppercased. "Stack" becomes `STA`, "Data" becomes `DAT`.
- Multi-word category: first letter of each word, take first 3. "User Interface" becomes `UI` + padding.
- Words shorter than 3 characters are padded by repeating the last character (e.g., "UI" becomes `UII`, "DB" becomes `DBB`).
- Empty or blank names produce `UNK`.

IDs are stored with their prefix -- `@STA-0001` in JSON, in chat, and in DECISIONS.md. You reference them the same way everywhere.

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
  "defaultCare": "auto",
  "domainCare": {
    "Security": "review",
    "UI": "auto"
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
  "focus.switch": ["tab"],
  "resolver.next": ["n"],
  "resolver.prev": ["p"],
  "custom": ["c"],
  "shuffle": ["s"],
  "why": ["w"],
  "ask": ["a"],
  "quit": ["ctrl+q"],
  "care.up": ["l", "right"],
  "care.down": ["h", "left"]
}
```

User bindings replace defaults per-action (not append). Unknown actions are accepted for extensibility.

## Lifecycle Hooks

Hooks run at specific points in the decision/execution lifecycle. Configure them in `config.json` under the `hooks` key.

### Events

| Hook | When it fires |
|------|---------------|
| `pre-decision` | Before a decision is auto-decided |
| `post-decision` | After a decision is confirmed |
| `pre-execute` | Before the executor starts implementing |
| `post-execute` | After the executor completes |
| `decision-changed` | When a user revises a decision |

### Hook Types

- **Command**: runs a shell command with a 10-second timeout (e.g., `npm test`, `make lint`)
- **Webhook**: POSTs JSON to a URL with a 5-second timeout (e.g., Slack notification)

### Environment Variables

All hooks receive these environment variables: `DEFER_EVENT`, `DEFER_DECISION_ID`, `DEFER_DECISION_ANSWER`, `DEFER_CWD`.

## Custom Skills (Prompt Overrides)

The defer process is built from 5 skills:

| Skill | Purpose |
|-------|---------|
| `decompose` | Break task into decisions |
| `plan` | Identify remaining implementation decisions |
| `execute` | Implement a domain given decisions |
| `extract` | Extract implicit decisions from implementation |
| `verify` | Review domain implementation for correctness |

You can override any skill per-project by creating a `.md` file with YAML frontmatter in `.defer/skills/`:

```markdown
---
name: decompose
description: Custom decomposition for this project
---
Your custom prompt here. This replaces the default decompose prompt.
Focus on security decisions first...
```

Skills are discovered by walking up from the current directory. Project-level skills override parent-level skills, which override built-in defaults. Additional skill directories can be configured via `config.json`:

```json
{
  "skills": {
    "dirs": ["./custom-skills"]
  }
}
```

## Portable Mode (No CLI Required)

You don't need the defer CLI to follow the defer process. Run `defer init` to scaffold a config file for your AI tool:

```bash
defer init claude-code    # Creates CLAUDE.md
defer init cursor         # Creates .cursorrules
defer init copilot        # Creates .github/copilot-instructions.md
defer init codex          # Creates AGENTS.md
defer init universal      # Creates DEFER.md (copy into any tool's config)
```

The generated file contains the full defer philosophy (decompose, present, decide, implement, track, extract, verify). The AI tool reads it and follows the process natively -- no CLI needed.

## Provider Support

Auto-detected from environment, or set explicitly with flags:

| Provider | How to enable |
|------------------|----------------------------------------------|
| Claude Code | Default (free with subscription) |
| OpenAI | `export OPENAI_API_KEY=sk-...` |
| Groq | `export GROQ_API_KEY=gsk_...` |
| Mistral | `--provider mistral` / `MISTRAL_API_KEY` |
| Together | `--provider together` / `TOGETHER_API_KEY` |
| DeepInfra | `--provider deepinfra` / `DEEPINFRA_API_KEY` |
| Cerebras | `--provider cerebras` / `CEREBRAS_API_KEY` |
| Perplexity | `--provider perplexity` / `PERPLEXITY_API_KEY` |
| OpenRouter | `--provider openrouter` / `OPENROUTER_API_KEY` |
| Ollama | `--provider ollama --model llama3.1` |
| Any OpenAI-compat | `--provider <url> --api-key <key>` |

## Commands

```
defer "task"                Start a new project with the given task
defer                       Resume the last session in the current directory
defer init [target]         Scaffold config for: claude-code, cursor, copilot, codex, universal
defer sessions list         List all sessions in directory tree
defer sessions delete       Delete .defer/ in current directory
defer sessions export       Print DECISIONS.md to stdout
defer --debug "task"        Headless mode (no TUI, prints to stdout)
defer --model opus "task"   Use a specific model (sonnet, opus, haiku, or provider-specific ID)
defer --provider <p> "task" Override the AI provider
defer --api-key <k> "task"  Override the API key
defer --no-mascot "task"    Hide the mascot header
defer --version             Show version
```

## Permission Model

Care levels control what the executor is allowed to do without prompting:

| Care Level | Read files | Write files | Run commands |
|------------|-----------|-------------|-------------|
| auto | auto | auto | auto |
| review | auto | prompt | prompt |

Tool classification: `Glob`, `Grep`, `Read` are read actions. `Write`, `Edit` are write actions. `Bash` is an execute action. Unknown tools default to execute (most restrictive).

## Architecture

The codebase lives under `go/` and is organized into these packages:

| Package | Purpose |
|---------|---------|
| `cmd` | CLI entry point, cobra commands (root, sessions, init, debug) |
| `internal/agent` | Decomposition agent, domain executor, manager coordinator, system prompts, events |
| `internal/api` | Provider interface, auto-detection, Claude Code subprocess, OpenAI-compatible HTTP, tool execution |
| `internal/config` | Settings cascade (global -> project -> CLI flags) |
| `internal/decision` | Core types (Decision, DecisionStore, Feature), persistence, ID generation, DECISIONS.md, dependency tracking |
| `internal/hooks` | Lifecycle hooks (shell commands + webhooks) |
| `internal/keybindings` | Configurable keybindings with defaults |
| `internal/mcp` | MCP client (JSON-RPC 2.0 over stdio) |
| `internal/permissions` | Care-level-aware tool permissions |
| `internal/skills` | Skill file loading (.md with YAML frontmatter), directory discovery |
| `internal/templates` | Defer philosophy spec, tool-specific config templates (5 targets) |
| `internal/tui` | Bubbletea TUI: app, tree, priorities, welcome, mascot (box-drawn, 4 lines tall), styles, notifications, messages |
| `internal/update` | Version update checking |

Session state is persisted in `.defer/`:

- `decisions.json` -- all decisions and features
- `priorities.json` -- care levels per domain
- `session_id` -- Claude Code session continuity

## Development

```bash
cd go
go build ./...              # build all packages
go test ./... -count=1      # run tests
go test ./... -race         # tests with race detector
go vet ./...                # static analysis
go build -o defer .         # build binary
```

## License

MIT
