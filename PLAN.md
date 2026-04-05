# Defer Release Plan: v3.3.0 / v3.4.0 / v3.5.0

---

## v3.3.0 — Distribution

### Summary of findings

**Homebrew**: GoReleaser v2.10+ deprecates `brews` in favor of `homebrew_casks` (pre-built binaries are Casks, not Formulas). The tap repo must be named `homebrew-tap` under the `defer-ai` org. GoReleaser manages the tap entirely — it commits the generated cask file on every release. A separate GitHub PAT with `repo` scope (`HOMEBREW_TAP_TOKEN`) is required because `GITHUB_TOKEN` from Actions can't push cross-repo.

**go install**: The module path is `github.com/defer-ai/cli` but `go.mod` lives in the `go/` subdirectory. `go install github.com/defer-ai/cli@latest` fails because the Go toolchain expects `go.mod` at the repo root. Three options exist: (A) move Go source to repo root, (B) change module to `github.com/defer-ai/cli/go` with `go/v3.3.0` tags, or (C) use a vanity domain with Go 1.25's subdirectory support. Option A is simplest. Option C is cleanest for users but requires DNS setup.

**Windows/ConPTY**: Bubbletea v1.3.x has fixes for the two biggest issues (flickering from spurious resize events, broken arrow keys). Remaining problems — mouse in altscreen, ConPTY swallowing escape sequences — are edge cases that don't affect defer's keyboard-only UI. The main risk is Claude Code subprocess not being available on Windows (it requires Node.js + npm). Bubbletea v2 has a new renderer with dramatically better Windows support, but migration is a major effort (import path change, View API change, key message restructuring).

**New init targets**: Windsurf uses `.windsurfrules` (legacy) or `.windsurf/rules/*.md` (modern, with YAML frontmatter). Zed uses `.rules` (also reads `.cursorrules`, `CLAUDE.md`, `AGENTS.md` as fallbacks). Amp uses `AGENTS.md` (already covered by `codex` target). Cline uses `.clinerules`. Gemini CLI uses `GEMINI.md`. Continue uses `.continue/rules/*.md`. Aider uses `CONVENTIONS.md`. The existing template system in `templates.go` makes adding targets trivial — just add a const, a map entry, and a list entry.

### Architecture decisions

#### A1: Module layout for `go install`

| Option | UX | Migration effort | Tradeoffs |
|--------|-----|-----------------|-----------|
| **A: Move Go to repo root** | `go install github.com/defer-ai/cli@latest` | Medium — move files, update CI, goreleaser `dir:` | Mixes Go source with web/, README at root. Clean import path. |
| **B: Subdirectory module** | `go install github.com/defer-ai/cli/go@latest` | Low — change go.mod module path, prefix tags with `go/` | Ugly install command. Tag prefix confuses goreleaser. |
| **C: Vanity domain** | `go install defer.sh/cli@latest` | Medium — DNS, hosting, go.mod change | Best UX. Requires web hosting for `?go-get=1` meta tag. Depends on defer.sh domain. |

**Recommendation**: A first (unblocks go install immediately), then optionally C later (defer.sh is already owned). A and C are not mutually exclusive — A is the implementation, C is an alias.

#### A2: Homebrew distribution method

| Option | User experience | Maintenance |
|--------|----------------|-------------|
| **Homebrew Cask (via tap)** | `brew tap defer-ai/tap && brew install defer` | Automatic via goreleaser. Needs PAT secret. |
| **Homebrew Core** | `brew install defer` | Requires community submission + review. Must meet popularity threshold. |
| **Both** | Best of both worlds | Tap is immediate; Core submission when popular enough. |

**Recommendation**: Tap first. Core submission is a future goal, not a v3.3.0 task.

#### A3: Windsurf template format

| Option | Compatibility |
|--------|--------------|
| **Legacy `.windsurfrules`** | Works everywhere, simpler |
| **Modern `.windsurf/rules/defer.md`** | Supports `trigger: always_on` frontmatter, coexists with other rules |

**Recommendation**: Modern format. The template system already handles subdirectory creation.

### File-by-file change plan

#### Feature: Homebrew tap

| File | Change |
|------|--------|
| `.goreleaser.yaml` | Add `brews:` section (not `homebrew_casks` — Bubbletea v1 needs the formula `install` block for binary placement). Target `defer-ai/homebrew-tap` repo. |
| `.github/workflows/release.yaml` | Add `HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}` to goreleaser env. |
| **New repo**: `defer-ai/homebrew-tap` | Empty repo. GoReleaser auto-creates `Formula/defer.rb`. |
| README.md | Add `brew tap defer-ai/tap && brew install defer` install method. |

#### Feature: `go install` support

| File | Change |
|------|--------|
| `go/go.mod` | Move to repo root as `go.mod`. Module path stays `github.com/defer-ai/cli`. |
| `go/go.sum` | Move to repo root as `go.sum`. |
| `go/main.go` | Move to repo root as `main.go`. |
| `go/cmd/` | Move to `cmd/`. |
| `go/internal/` | Move to `internal/`. |
| `.goreleaser.yaml` | Remove `dir: go`. Update `before.hooks` to drop `cd go &&`. |
| `.github/workflows/ci.yaml` | Remove `working-directory: go`. Update `go-version-file` and `cache-dependency-path`. |
| `.github/workflows/release.yaml` | Update `go-version-file` and `cache-dependency-path`. |
| README.md | Add `go install github.com/defer-ai/cli@latest`. |

#### Feature: New `defer init` targets

| File | Change |
|------|--------|
| `internal/templates/templates.go` | Add 5 new targets: `windsurf`, `zed`, `cline`, `gemini`, `aider`. Add consts, map entries, update `TargetList()`. |
| README.md | Document new targets in the `defer init` section. |

Windsurf template uses YAML frontmatter:
```markdown
---
trigger: always_on
description: Defer — zero-autonomy AI decision process
---
{DeferProcess}
```

Other targets wrap `DeferProcess` with tool-specific integration instructions (same pattern as existing targets).

#### Feature: Windows smoke test

| File | Change |
|------|--------|
| `.github/workflows/ci.yaml` | Add `windows-latest` to the matrix. Build + test only (no TUI integration test). |

### Implementation order

1. Move Go source to repo root (everything else depends on correct paths)
2. Update CI and goreleaser configs
3. Add Homebrew tap configuration
4. Add new init targets
5. Add Windows CI matrix entry
6. Update README

### Risk flags

- **Moving Go to root is a large diff** that touches every import path in tests. Use `go mod edit` and `find/sed` carefully. Verify all tests pass before committing.
- **Homebrew PAT secret** must be created manually in GitHub org settings. If the `defer-ai` org has branch protection or required reviews on `homebrew-tap`, goreleaser's push will fail.
- **Windows CI may reveal test failures** in packages that use Unix-specific paths (e.g., `filepath.Join` with hardcoded `/`). The `decision` and `config` packages use `filepath.Join` correctly, but audit `skills/discovery.go` which walks directories.

### Estimated complexity

| Feature | Size | Notes |
|---------|------|-------|
| Move Go to root | M | Large diff, mechanical, low risk |
| Homebrew tap | S | Config-only + repo creation |
| New init targets | S | 5 map entries + template text |
| Windows CI | S | Matrix addition + possible path fixes |

---

## v3.4.0 — Decisions as durable artifacts

### Summary of findings

**Decision schema**: The current `Decision` struct has no revision tracking. `Source` reflects only the latest state, so override rate (auto -> user) is invisible. Adding `OriginalSource`, `RevisionCount`, `CreatedAt`, and `AnsweredAt` fields (all `omitempty`) is backward-compatible and enables analytics without breaking existing stores.

**Templates/presets**: ADR tools use simple markdown-template-per-type patterns. For defer, YAML files in `.defer/templates/` containing pre-defined decision lists (category, question, options, impact, dependencies) are the natural fit. Discovery mirrors the existing config cascade: built-in -> `~/.defer/templates/` -> `.defer/templates/`. The simplest approach is pure YAML files (no frontmatter needed since these aren't skills).

**Cross-project import**: The main challenges are ID conflicts (two projects will have `STA-0001`) and dependency resolution. Re-IDing on import using `decision.NextID()` with an old->new ID mapping solves both. An `ImportedFrom` field preserves provenance. The feature should validate that all `dependsOn` references are resolvable (either in the import set or in existing decisions).

**Git commit integration**: Three viable approaches — `defer trailers` (prints Decision-Ref trailers), `defer commit` (wraps git with trailers appended), and a `prepare-commit-msg` hook installer. The trailers-first approach doesn't require git as a hard dependency. Git trailers (`Decision-Ref: @STA-0001`) are natively understood by `git log --format='%(trailers)'`.

**Stats**: Most metrics are computable from the existing store today (auto/review ratio, impact distribution, decisions per category, pending count, dependency depth). Override rate and time-to-decide require the new schema fields. The output should be a compact terminal report — no charting libraries needed, just aligned text.

### Architecture decisions

#### B1: Template/preset file format

| Option | Authoring UX | Parsing complexity |
|--------|-------------|-------------------|
| **YAML list** | Natural for structured data. Human-readable. | One dependency (`gopkg.in/yaml.v3`). |
| **JSON** | Matches `decisions.json` format | Verbose, harder to author by hand. |
| **Markdown tables** | Matches DECISIONS.md visual format | Fragile to parse. Loses nested fields (options, dependencies). |

**Recommendation**: YAML. It maps cleanly to `[]Decision`, is easy to hand-author, and avoids the parsing fragility of markdown tables. The YAML dependency is lightweight and commonly used in Go CLIs.

#### B2: Cross-project import UX

| Option | Interaction model |
|--------|------------------|
| **CLI arguments** | `defer import ../project-a @STA-0001 @DAT-0001` — explicit, scriptable |
| **Interactive picker** | `defer import ../project-a` — shows decision list, user selects with space/enter |
| **Both** | Arguments for scripting, interactive if no IDs specified |

**Recommendation**: Both. IDs as arguments for CI/scripting. Interactive picker (using Bubbletea list component) when no IDs given.

#### B3: Git integration approach

| Option | Git dependency | Transparency |
|--------|---------------|-------------|
| **`defer trailers` only** | None (prints text) | User controls how trailers enter commits. |
| **`defer commit` wrapper** | Requires `git` binary | Convenient but parallel workflow to `git commit`. |
| **Hook installer** | Requires `.git/hooks/` | Automatic but requires hook setup step. |
| **All three** | Graduated | Maximum flexibility. |

**Recommendation**: Ship `defer trailers` and `defer commit` in v3.4.0. Hook installer is a follow-up (it's a convenience, not a capability).

#### B4: Stats schema additions

The following new fields are needed on `Decision`:

```go
OriginalSource string     `json:"originalSource,omitempty"` // set once when answer first assigned
RevisionCount  int        `json:"revisionCount,omitempty"`  // incremented on each answer change
CreatedAt      string     `json:"createdAt,omitempty"`      // RFC3339 when decision was created
AnsweredAt     string     `json:"answeredAt,omitempty"`     // RFC3339 when answer was last set
ImportedFrom   *ImportRef `json:"importedFrom,omitempty"`   // provenance for imported decisions
```

All `omitempty` for backward compatibility. Existing stores load without error; new fields populate going forward. The `OriginalSource` and `RevisionCount` must be updated everywhere an answer is set: `Agent.AutoDecide()`, TUI confirm flow (`handleResolverKey`), `Executor.storeDecision()`, and `Executor.UpdateDecision()`.

### File-by-file change plan

#### Feature: Decision presets

| File | Change |
|------|--------|
| **New**: `internal/templates/presets.go` | `Preset` struct, `PresetDecision` struct, `DiscoverPresets(cwd)`, `LoadPresetFile(path)`, built-in presets (rest-api, spa, cli-tool, library). |
| **New**: `internal/templates/presets_test.go` | Tests for discovery, loading, merge precedence. |
| `cmd/init.go` | Add `--preset` flag. When set, load preset and seed `.defer/decisions.json` with pre-defined decisions. |
| README.md | Document preset usage. |

#### Feature: `defer import`

| File | Change |
|------|--------|
| **New**: `cmd/import.go` | `importCmd` cobra command. Loads source store, selects decisions (by args or picker), validates dependencies, re-IDs, merges into current store. |
| **New**: `cmd/import_test.go` | Tests for ID remapping, dependency validation, duplicate detection. |
| `internal/decision/decision.go` | Add `ImportedFrom *ImportRef` field and `ImportRef` struct. |
| `internal/decision/id.go` | No changes needed — `NextID()` already handles the re-IDing correctly. |
| `internal/decision/markdown.go` | Show import provenance in DECISIONS.md when `ImportedFrom` is set. |

#### Feature: `defer stats`

| File | Change |
|------|--------|
| **New**: `cmd/stats.go` | `statsCmd` cobra command. Loads store, computes metrics, prints formatted report. |
| **New**: `cmd/stats_test.go` | Tests for metric computation. |
| `internal/decision/decision.go` | Add `OriginalSource`, `RevisionCount`, `CreatedAt`, `AnsweredAt` fields. |
| `internal/decision/store.go` | Set `CreatedAt` in `storeDecision()` paths (first time only). |
| **New**: `internal/decision/stats.go` | Pure functions: `OverrideRate()`, `AutoReviewRatio()`, `ImpactDistribution()`, `MaxDependencyDepth()`, `MostRevised()`, `DecisionsPerCategory()`. |
| **New**: `internal/decision/stats_test.go` | Tests for each metric function. |
| `internal/agent/executor.go` | Set `OriginalSource`/`RevisionCount`/`AnsweredAt` in `storeDecision()` and `UpdateDecision()`. |
| `internal/agent/agent.go` | Set `OriginalSource`/`CreatedAt` in `AutoDecide()`. |
| `internal/tui/tree.go` | Set `RevisionCount`/`AnsweredAt` in confirm handlers. |

#### Feature: `defer trailers` and `defer commit`

| File | Change |
|------|--------|
| **New**: `internal/decision/trailers.go` | `Trailers(store)` returns formatted git trailer lines. `TrailersForIDs(store, ids)` for specific decisions. |
| **New**: `internal/decision/trailers_test.go` | Tests for trailer formatting. |
| **New**: `cmd/trailers.go` | `trailersCmd` — prints trailers to stdout. Flags: `--since` (date filter), `--ids` (specific IDs). |
| **New**: `cmd/commit.go` | `commitCmd` — wraps `git commit` with trailers appended. Flags: `-m` (message), `--dry-run` (show what would be committed). |

### Implementation order

1. Schema additions to `decision.go` (everything else depends on the new fields)
2. Wire `OriginalSource`/`RevisionCount`/`CreatedAt`/`AnsweredAt` into all answer-setting code paths
3. `defer stats` (validates that schema additions work end-to-end)
4. `defer trailers` + `defer commit` (independent of stats, but benefits from schema)
5. Decision presets (independent feature)
6. `defer import` (depends on schema additions for `ImportedFrom`)

### Risk flags

- **Schema migration**: Existing `.defer/decisions.json` files won't have the new fields. All new fields must be `omitempty` and code must handle zero values gracefully. The `CreatedAt` field will be empty for old decisions — stats should skip those when computing time-to-decide.
- **Answer-setting code paths are scattered**: `Agent.AutoDecide()`, `Executor.storeDecision()`, `Executor.UpdateDecision()`, `Executor.scanInlineDecisions()`, `Executor.parseImplicitChoices()`, TUI confirm in `handleResolverKey()`, and TUI detail confirm in `handleDetailKey()`. Missing any one of these means inaccurate stats. A test that creates a decision via each path and verifies `OriginalSource` is set would catch this.
- **`defer commit` shelling out to git**: Must handle git not being installed (clear error message), not being in a git repo, and git commit failing (e.g., nothing staged). The `--dry-run` flag helps users verify before committing.
- **YAML dependency**: Adding `gopkg.in/yaml.v3` is the first non-Charm/Cobra/stdlib dependency. Should be acceptable — it's the de facto YAML library for Go.

### Estimated complexity

| Feature | Size | Notes |
|---------|------|-------|
| Schema additions + wiring | M | Many code paths to touch, needs thorough testing |
| `defer stats` | S | Pure computation + formatted output |
| `defer trailers` + `defer commit` | S | Thin CLI wrappers |
| Decision presets | M | New file format, discovery, built-in content |
| `defer import` | M | ID remapping, dependency validation, interactive picker |

---

## v3.5.0 — Team workflows and MCP server mode

### Summary of findings

**MCP server**: The existing `mcp/client.go` already defines all the JSON-RPC types needed for a server (`jsonRPCRequest`, `jsonRPCResponse`, `jsonRPCError`, `Tool`, `callToolParams`, `callToolResult`). A server is the mirror image: read requests from stdin, dispatch to handlers, write responses to stdout. Stdio transport is the right choice — it matches how Claude Code, Cursor, and other tools discover MCP servers. The official Go SDK exists (`github.com/modelcontextprotocol/go-sdk`) but adds a dependency for ~150 lines of reusable code. Building on the existing types is lighter.

**Tool schema**: Six tools cover the full decision lifecycle: `read_decisions` (list/filter), `list_pending` (pending only, sorted by impact), `confirm_decision` (set answer, cascade invalidation), `update_decision` (modify properties), `get_session_state` (summary metrics), `get_decision_tree` (hierarchical view). Input schemas use JSON Schema. All tools read from / write to `.defer/decisions.json`.

**GitHub PR integration**: The GitHub REST API supports posting review comments via `POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews`. For simplicity, shelling out to `gh pr comment --body "..."` avoids a library dependency and gets free auth via `gh auth login`. The body would be a formatted DECISIONS.md diff showing what changed.

**Shared sessions**: Two processes sharing `.defer/decisions.json` risk lost updates (both read, both write, last write wins) and torn reads (read during mid-write). The fix is: (1) advisory file lock via `gofrs/flock` on `.defer/decisions.lock`, (2) atomic writes (write to `.tmp`, rename), (3) MCP server reloads store on every tool call (never caches). File watching via `fsnotify` is nice-to-have for push notifications but not essential — reload-on-call is sufficient.

### Architecture decisions

#### C1: MCP server implementation

| Option | Dependency | Effort | Control |
|--------|-----------|--------|---------|
| **Reuse existing types, hand-rolled server** | None (internal package) | ~150 LOC | Full |
| **Official Go SDK** (`modelcontextprotocol/go-sdk`) | New dependency | ~50 LOC | Limited by SDK API |
| **Community SDK** (`mark3labs/mcp-go`) | New dependency | ~50 LOC | Different abstraction |

**Recommendation**: Hand-rolled server reusing `mcp/` types. The JSON-RPC loop is trivial, and the existing types are already there. Avoids a dependency and keeps the server code alongside the client code in one package.

#### C2: GitHub PR integration approach

| Option | Dependency | Auth |
|--------|-----------|------|
| **`gh` CLI** | Requires `gh` installed | `gh auth login` (already common) |
| **`go-github` library** | New Go dependency | `GITHUB_TOKEN` env var |
| **Direct HTTP** | None | `GITHUB_TOKEN` env var |

**Recommendation**: `gh` CLI first (simplest, most common auth). Fall back to direct HTTP with `GITHUB_TOKEN` if `gh` is not installed. Avoid the `go-github` library dependency for a single API call.

#### C3: File locking strategy

| Option | Cross-platform | Complexity |
|--------|---------------|-----------|
| **`gofrs/flock`** (advisory lock on `.defer/decisions.lock`) | Yes (flock/LockFileEx) | Low — wrap read-modify-write |
| **Custom lock file** (create/check `.defer/decisions.lock` PID file) | Yes | Medium — must handle stale locks |
| **No locking, last-write-wins** | N/A | None — but data loss risk |

**Recommendation**: `gofrs/flock`. It's battle-tested (Docker, Terraform), cross-platform, and the integration is ~30 lines wrapping `LoadStore`/`SaveStore`.

#### C4: MCP server tool set

Six tools, organized by capability:

| Tool | Read/Write | Description |
|------|-----------|-------------|
| `read_decisions` | Read | List decisions with optional filters (status, category, feature, source) |
| `list_pending` | Read | Get pending decisions sorted by impact |
| `confirm_decision` | Write | Set answer on a decision, cascade invalidation |
| `update_decision` | Write | Modify decision properties (question, options, category, impact) |
| `get_session_state` | Read | Summary: task, counts, progress, features, categories |
| `get_decision_tree` | Read | Hierarchical view grouped by dependencies, category, or feature |

Each write tool acquires the file lock, reloads from disk, mutates, saves, and releases. Each read tool acquires a shared lock (or just reads — atomic writes prevent torn reads).

### File-by-file change plan

#### Feature: MCP server mode (`defer serve --mcp`)

| File | Change |
|------|--------|
| **New**: `internal/mcp/server.go` | `Server` struct with `Run(ctx)` method. Reads JSON-RPC from stdin, dispatches to tool handlers, writes responses to stdout. Uses existing `jsonRPCRequest`/`jsonRPCResponse` types. |
| **New**: `internal/mcp/server_test.go` | Tests: initialize handshake, tools/list, each tool call, error cases. Use `io.Pipe` to simulate stdin/stdout. |
| **New**: `internal/mcp/tools.go` | Tool handler functions: `handleReadDecisions()`, `handleListPending()`, `handleConfirmDecision()`, `handleUpdateDecision()`, `handleGetSessionState()`, `handleGetDecisionTree()`. Each loads store from disk, operates, saves if write. |
| **New**: `internal/mcp/tools_test.go` | Unit tests for each tool handler with fixture stores. |
| **New**: `cmd/serve.go` | `serveCmd` with `--mcp` flag. Creates `mcp.Server` with cwd, runs `server.Run(ctx)`. |
| `cmd/root.go` | Register `serveCmd`. |
| README.md | Document MCP server setup for Claude Code, Cursor, etc. |

MCP server tool registration (in `server.go`):
```go
func (s *Server) tools() []Tool {
    return []Tool{
        {Name: "read_decisions", Description: "...", InputSchema: readDecisionsSchema},
        {Name: "list_pending", Description: "...", InputSchema: listPendingSchema},
        {Name: "confirm_decision", Description: "...", InputSchema: confirmDecisionSchema},
        {Name: "update_decision", Description: "...", InputSchema: updateDecisionSchema},
        {Name: "get_session_state", Description: "...", InputSchema: getSessionStateSchema},
        {Name: "get_decision_tree", Description: "...", InputSchema: getDecisionTreeSchema},
    }
}
```

#### Feature: File locking + atomic writes

| File | Change |
|------|--------|
| `go.mod` (root after v3.3.0 move) | Add `github.com/gofrs/flock` dependency. |
| `internal/decision/store.go` | Add `WithStoreLock(cwd, fn)` function. Change `SaveStore` to use atomic write (write `.tmp`, rename). Wrap write paths in lock acquisition. |
| **New**: `internal/decision/store_lock_test.go` | Test concurrent access: two goroutines writing simultaneously, verify no data loss. |

#### Feature: GitHub PR decisions comment (`defer review`)

| File | Change |
|------|--------|
| **New**: `cmd/pr_comment.go` | `prCommentCmd` — generates DECISIONS.md diff, posts as PR comment via `gh pr comment` or direct HTTP. Flags: `--pr` (PR number/URL), `--diff-only` (just print diff, don't post). |
| **New**: `internal/decision/diff.go` | `DiffMarkdown(old, new *DecisionStore)` — generates a human-readable diff of decision changes (new decisions, changed answers, invalidations). |
| **New**: `internal/decision/diff_test.go` | Tests for diff generation. |
| README.md | Document `defer review` usage. |

#### Feature: Shared session safety

| File | Change |
|------|--------|
| `internal/decision/store.go` | `SaveStore` uses atomic write. All read-modify-write cycles wrapped in `WithStoreLock`. |
| `internal/mcp/tools.go` | Every write tool calls `WithStoreLock`. Read tools use standard `LoadStore` (atomic writes prevent torn reads). |
| `internal/agent/executor.go` | `storeDecisionAndSave()` wraps in `WithStoreLock`. |
| `internal/agent/manager.go` | `persistDecisions()` wraps in `WithStoreLock`. |

### Implementation order

1. File locking + atomic writes (foundation for everything else)
2. MCP server core (initialize, tools/list, tools/call dispatch loop)
3. MCP tool handlers (read_decisions, list_pending first, then write tools)
4. Decision diff generation
5. `defer review`
6. Wire locking into existing write paths (executor, manager)
7. Integration testing with Claude Code as MCP client

### Risk flags

- **MCP protocol compliance**: The initialize handshake has specific ordering requirements (client sends `initialize`, server responds, client sends `initialized` notification). Missing any step causes clients to hang. Test against Claude Code early.
- **File locking on NFS/network drives**: `flock` doesn't work reliably on NFS. If someone has `.defer/` on a network mount, locking will silently fail. Document this limitation.
- **`gh` CLI availability**: Not everyone has `gh` installed. The `defer review` command must fail gracefully with a clear message ("install gh CLI or set GITHUB_TOKEN"). Falling back to direct HTTP is more work but removes the hard dependency.
- **MCP server and TUI competing for stdin**: If `defer serve --mcp` is launched as a subprocess by an AI tool, it reads stdin for JSON-RPC. But the TUI also reads stdin for keyboard input. These must be mutually exclusive — `serve` is a headless mode, never launched alongside the TUI. The cobra command should enforce this.
- **Concurrent executor writes**: The executor runs in a goroutine and calls `storeDecisionAndSave()` frequently. Adding file locking here means the executor blocks on lock acquisition. Since it's the only writer in a normal session (TUI reads only), contention is rare. But in shared sessions (TUI + MCP server), the lock must be fair.
- **JSON schema for tool inputs**: Each MCP tool needs a `json.RawMessage` containing a valid JSON Schema. These schemas must be maintained as Go constants or generated. Hand-maintained schemas are error-prone but keep the code simple.

### Estimated complexity

| Feature | Size | Notes |
|---------|------|-------|
| MCP server core | M | ~150 LOC server loop + 6 tool handlers |
| File locking + atomic writes | S | ~40 LOC, one dependency |
| MCP tool handlers | L | 6 tools with schemas, validation, tests |
| `defer review` | M | Diff generation + gh/HTTP integration |
| Shared session wiring | S | Wrap existing write paths in lock |

---

## Cross-release dependency graph

```
v3.3.0 (Distribution)
├── Move Go to repo root ← EVERYTHING depends on this
├── Homebrew tap
├── New init targets
└── Windows CI

v3.4.0 (Durable artifacts)
├── Schema additions (OriginalSource, RevisionCount, CreatedAt, AnsweredAt, ImportedFrom)
├── defer stats ← depends on schema additions
├── defer trailers + defer commit
├── Decision presets (YAML dependency)
└── defer import ← depends on schema additions

v3.5.0 (Team workflows)
├── File locking + atomic writes ← depends on v3.3.0 repo layout
├── MCP server core ← depends on file locking
├── MCP tool handlers ← depends on MCP server core
├── defer review ← depends on diff generation
└── Shared session wiring ← depends on file locking
```

## New dependencies summary

| Release | Dependency | Purpose | Size |
|---------|-----------|---------|------|
| v3.4.0 | `gopkg.in/yaml.v3` | Parse preset YAML files | Standard, widely used |
| v3.5.0 | `github.com/gofrs/flock` | Cross-platform file locking | Small, battle-tested |

## New CLI commands summary

| Release | Command | Description |
|---------|---------|-------------|
| v3.4.0 | `defer stats` | Print decision analytics |
| v3.4.0 | `defer trailers` | Print git Decision-Ref trailers |
| v3.4.0 | `defer commit -m "msg"` | Git commit with trailers appended |
| v3.4.0 | `defer import <path> [@IDs]` | Import decisions from another project |
| v3.5.0 | `defer serve --mcp` | Run as MCP server (stdio transport) |
| v3.5.0 | `defer review --pr <num>` | Post DECISIONS.md diff as PR comment |

## New file count per release

| Release | New files | Modified files |
|---------|-----------|---------------|
| v3.3.0 | 0 | ~20 (repo restructure) + 1 (templates.go) |
| v3.4.0 | ~12 | ~8 |
| v3.5.0 | ~10 | ~6 |
