<p align="center"><img width="255" height="255" alt="defer" src="https://github.com/user-attachments/assets/bca626d3-979d-49e5-9f6e-a4c6fd45ffe1" /></p>
<p align="center">Zero-autonomy AI. Every decision is yours.</p>

## What is Defer

Defer decomposes your task into a tree of decisions, lets you set care levels per domain (auto or review), auto-decides the rest, then implements everything while you watch, chat, and challenge in real-time. Every choice -- from framework to variable name -- is tracked, reversible, and exportable. The agent never asks questions as text; all ambiguity becomes decisions with concrete options.

<img width="1378" height="1002" alt="image" src="https://github.com/user-attachments/assets/2a6005cd-4815-4da6-9c4d-8d02e178f399" />

## Install

**Homebrew:**

```bash
brew tap defer-ai/tap
brew install defer
```

**go install:**

```bash
go install github.com/defer-ai/cli@latest
```

**From source:**

```bash
git clone https://github.com/defer-ai/cli.git && cd cli
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
         | Executor: implement -> verify -> extract
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

Press `tab` to cycle focus (tree -> chat -> resolver), `shift+tab` to reverse.

## Three-Panel Layout

On wide terminals (>80 columns), the decision tree and chat are always visible side by side. There are three focus targets: tree (left), chat (right top), and resolver (right bottom). `tab` cycles focus forward (tree -> chat -> resolver -> tree), `shift+tab` cycles in reverse. On narrow terminals, only the focused panel is shown.

The right panel has two regions: the chat log on top, and the pending resolver at the bottom. The resolver shows pending decisions as a wizard (e.g., 1/2, 2/3) so you can resolve them inline without leaving the chat.

```
┌── Tree ───────────────┐ ┌── Chat ────────────────┐
│ Stack                 │ │ > build a todo app      │
│   ▪ STA-0001 Lang?   │ │                         │
│   ▪ STA-0002 FW?     │ │ ● Glob(files **/*)      │
│ Auth                  │ │ Found 6 decisions       │
│   ○ AUT-0001 Method?  │ ├── Resolver ─────────────┤
│                       │ │ Pending 1/2             │
│                       │ │ ○ Auth method?           │
│                       │ │ > A) JWT  B) Session     │
└───────────────────────┘ └─────────────────────────┘
  tab cycles: Tree → Chat → Resolver → Tree
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
| `tab` | Cycle focus forward: tree -> chat -> resolver -> tree |
| `shift+tab` | Cycle focus in reverse: tree -> resolver -> chat -> tree |
| `ctrl+q` | Quit |
| `ctrl+c` | Shows warning (press `ctrl+q` to actually quit) |

### Tree Panel (left)

| Key | Action |
|-----|--------|
| `↑` / `↓` or arrows | Navigate decisions |
| `enter` | Inspect decision |
| `/` | Filter decisions by ID, category, or question |
| `f` or `ctrl+f` | Find and jump to a decision, category, or feature |
| `s` | Cycle sort: category, impact, status, a-z |

### Decision Detail

| Key | Action |
|-----|--------|
| `↑` / `↓` or arrows | Navigate options |
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
| `tab` | Cycle autocomplete (when completions visible), otherwise switch focus |
| `ctrl+o` | Toggle expand/collapse on last agent topic |
| `↑` / `↓` | Navigate resolver options (when input is empty) |
| `←` / `→` | Cycle through pending decisions in resolver |

All keybindings are configurable via `~/.defer/keybindings.json`.

## Pending Resolver

The pending resolver is the bottom section of the right (chat) panel. When the agent discovers decisions that require your input, they appear here as a wizard -- one at a time, showing progress (e.g., "Pending 1/3").

Each pending decision shows its question and concrete options. Use `↑`/`↓` (when the chat input is empty) to navigate options and `enter` to confirm. Use `←`/`→` to cycle to the next or previous pending decision. Care levels are also set inline here when the first decisions arrive -- no separate picker screen.

## Care Levels

| Level | What happens |
|--------|------------------------------------------|
| auto | Agent decides everything. Decisions visible in tree, challengeable anytime. |
| review | You confirm each decision before execution proceeds. |

Same decisions either way. The difference: auto answers them for you (you challenge after), review leaves them pending (you confirm first).

## Features

Decisions can be tagged with features (e.g., "messaging", "auth", "encryption"). Features provide a second axis for organizing decisions beyond categories.

**Feature tagging.** Press `f` in the decision detail view to edit a decision's feature tags. Enter comma-separated values (e.g., `auth, onboarding`). Feature IDs use the `#` prefix and follow the same 3-letter prefix rules as categories: `#MSG`, `#AUT`, `#ENC`.

**Sorting.** Press `s` in the decision list to cycle sort order: category, impact (high first), status (pending first), alphabetical.

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
  "navigate.up": ["up"],
  "navigate.down": ["down"],
  "inspect": ["enter"],
  "back": ["q", "esc"],
  "search": ["/"],
  "focus.switch": ["tab"],
  "resolver.next": ["right"],
  "resolver.prev": ["left"],
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

The defer process is built from 4 skills:

| Skill | Purpose |
|-------|---------|
| `decompose` | Break task into decisions |
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

## Decision Presets

Start projects faster with pre-defined decision templates. Presets provide a common set of decisions for typical project types.

**Built-in presets:**

| Preset | Description |
|--------|-------------|
| `rest-api` | Backend framework, language, database, ORM, API style, auth, hosting |
| `cli-tool` | Language, TUI framework, output format, distribution, config |
| `web-app` | Frontend framework, language, CSS, state management, API, hosting, CI/CD |

```bash
defer init claude-code --preset rest-api    # scaffold config + seed 7 decisions
defer init cursor --preset web-app          # works with any target
```

**Custom presets.** Create YAML files in `.defer/templates/` or `~/.defer/templates/`:

```yaml
name: my-preset
description: Custom decisions for my team's stack
decisions:
  - category: Stack
    question: "Backend framework?"
    options:
      - key: A
        label: Express
      - key: B
        label: FastAPI
    impact: 9
    context: "We're a Python shop"
```

Project presets override global presets, which override built-in presets (by name).

## Portable Mode (No CLI Required)

You don't need the defer CLI to follow the defer process. Run `defer init` to scaffold a config file for your AI tool:

```bash
defer init claude-code    # Creates CLAUDE.md
defer init cursor         # Creates .cursorrules
defer init copilot        # Creates .github/copilot-instructions.md
defer init codex          # Creates AGENTS.md (also works for Amp)
defer init windsurf       # Creates .windsurf/rules/defer.md
defer init zed            # Creates .rules
defer init cline          # Creates .clinerules
defer init gemini         # Creates GEMINI.md
defer init aider          # Creates CONVENTIONS.md
defer init continue       # Creates .continue/rules/defer.md
defer init universal      # Creates DEFER.md (copy into any tool's config)
```

The generated file contains the full defer philosophy (decompose, present, decide, implement, track, extract, verify). The AI tool reads it and follows the process natively -- no CLI needed.

## Git Integration

Track which decisions influenced each commit with git trailers.

```bash
defer trailers              # print Decision-Ref trailers for all decided decisions
defer trailers --ids STA-0001,DAT-0001   # specific decisions only

defer commit -m "Add auth"  # git commit with trailers auto-appended
defer commit -m "Add auth" --dry-run     # preview the full message
```

Example commit message:
```
Add auth

Decision-Ref: @STA-0001
Decision-Ref: @AUT-0001
Decision-Ref: @DAT-0001
```

## Decision Analytics

```bash
defer stats
```

Shows: total/pending/decided counts, auto/review ratio, override rate, per-category breakdown, impact distribution, dependency chain depth, and most-revised decisions.

## Cross-Project Import

Reuse decisions from other projects:

```bash
defer import ../api-project @STA-0001 @DAT-0001    # import specific decisions
defer import ../api-project --category Stack        # import by category
defer import ../api-project --keep-answers          # preserve original answers
```

Imported decisions are re-IDed to avoid conflicts, dependencies are remapped, and provenance is tracked (`importedFrom` field in the decision store).

## Review

Post a decision diff as a GitHub PR comment:

```bash
defer review --pr 42              # post changes since last commit
defer review --pr 42 --diff-only  # just print the diff
defer review --pr 42 --base ../old-project   # compare against baseline
```

Requires `gh` CLI (recommended) or `GITHUB_TOKEN` environment variable.

## MCP Server Mode

Run defer as an MCP (Model Context Protocol) server so AI tools can read and modify decisions programmatically:

```bash
defer serve --mcp
```

**6 tools exposed:**

| Tool | Description |
|------|-------------|
| `read_decisions` | List/filter decisions by status, category, feature, source |
| `list_pending` | Get pending decisions sorted by impact |
| `confirm_decision` | Set answer on a decision (cascades invalidation) |
| `update_decision` | Modify decision properties (question, options, impact) |
| `get_session_state` | Summary: task, counts, progress, features, categories |
| `get_decision_tree` | Hierarchical view grouped by dependencies |

**Example MCP config for Claude Code** (`~/.claude/mcp.json`):

```json
{
  "servers": {
    "defer": {
      "command": "defer",
      "args": ["serve", "--mcp"]
    }
  }
}
```

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
defer init [target]         Scaffold config for AI tools (see Portable Mode)
defer init --preset <name>  Initialize with preset decisions (rest-api, cli-tool, web-app)
defer stats                 Show decision analytics for current session
defer trailers              Print git Decision-Ref trailers
defer commit -m "msg"       Git commit with decision trailers appended
defer import <path> [@IDs]  Import decisions from another project
defer review --pr <n>   Post decision diff as GitHub PR comment
defer serve --mcp           Run as MCP server (stdio transport)
defer sessions list         List all sessions in directory tree
defer sessions delete       Delete .defer/ in current directory
defer sessions export       Print DECISIONS.md to stdout
defer update                Check for and install updates
defer --debug "task"        Headless mode (no TUI, prints to stdout)
defer --model opus "task"   Use a specific model
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

| Package | Purpose |
|---------|---------|
| `cmd` | CLI entry point, cobra commands |
| `internal/agent` | Decomposition agent, domain executor, manager coordinator, system prompts, events |
| `internal/api` | Provider interface, auto-detection, Claude Code subprocess, OpenAI-compatible HTTP, tool execution |
| `internal/config` | Settings cascade (global -> project -> CLI flags) |
| `internal/decision` | Core types, persistence, ID generation, DECISIONS.md, dependency tracking, stats, trailers, diffs |
| `internal/hooks` | Lifecycle hooks (shell commands + webhooks) |
| `internal/keybindings` | Configurable keybindings with defaults |
| `internal/mcp` | MCP client + server (JSON-RPC 2.0 over stdio) |
| `internal/permissions` | Care-level-aware tool permissions |
| `internal/skills` | Skill file loading (.md with YAML frontmatter), directory discovery |
| `internal/templates` | Defer philosophy spec, tool-specific config templates, decision presets |
| `internal/tui` | Bubbletea TUI: app, tree, priorities, mascot, styles, notifications, messages |
| `internal/update` | Version update checking |

Session state is persisted in `.defer/`:

- `decisions.json` -- all decisions and features
- `priorities.json` -- care levels per domain
- `session_id` -- Claude Code session continuity

## Development

```bash
go build ./...              # build all packages
go test ./... -count=1      # run tests
go test ./... -race         # tests with race detector
go vet ./...                # static analysis
go build -o defer .         # build binary
```

## License

MIT
