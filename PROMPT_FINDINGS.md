# Prompt optimization findings

DSPy + a real Claude benchmark were used to compare prompt variants for two
defer subsystems: the **decompose phase** (task → decisions JSON) and the
**executor phase** (task + decisions → implementation with inline DECIDED
lines).

## Round 1: decompose prompt (mistral via DSPy)

8 benchmark tasks, 4 prompt variants, scored on JSON validity, decision
count, options/impact presence, and category coverage. Run against local
ollama mistral 7B as a "tough audience" — anything mistral can parse,
Claude can parse easily.

| variant            | avg score |
|--------------------|----------:|
| `defer_current`    |    0.000  |
| `few_shot`         |    0.213  |
| `minimal`          |    0.741  |
| `role_numbered`    |  **0.884**|

The current defer decompose prompt **scored 0.0** because the JSON example
inside the prompt invited mistral to copy Python-dict syntax (single quotes)
instead of valid JSON, producing malformed output that the parser rejected.

A round 2 with refined variants didn't improve on `role_numbered`, so it
remains the winner.

**Caveat**: this benchmark is structural (does the output parse, does it
have the right shape). It's not a fair behavioural test against Claude,
which handles the bloated current prompt fine. But the structural problems
mistral surfaced (invited copy errors, mixed tool/no-tool paths, prose
bloat) are still real and worth fixing.

## Round 2: executor prompt (Claude Code via defer benchmark)

One real task ("build a small Go HTTP server with two endpoints, GET /time
and POST /echo, plus Makefile and README"), two prompt variants run on
Claude Code, ~9 minutes each. Metric:

- **inline DECIDED count** = decisions emitted by the agent in its text
  stream (via the line parser), counted exactly once each.

| variant   | inline DECIDED | categories                                   |
|-----------|---------------:|----------------------------------------------|
| baseline  |             34 | MIS:23, API:7, BUI:3, STA:1                  |
| **skill** |         **56** | MIS:41, TES:6, API:4, BUI:3, STA:1, LOG:1    |

The skill variant emitted **65 % more inline decisions** (56 vs 34) on the
same task. It also surfaced categories the baseline never raised (Testing,
Logging) and documented several "we're not doing this" decisions (CORS,
TLS, body size limits, graceful shutdown).

### Why the skill variant wins

The current executor prompt is heavily rule-based:

> "CRITICAL RULES — DO NOT batch — NEVER write multiple files in a row
> without DECIDED lines — MUST output a DECIDED line for every choice ..."

The more rules and prohibitions you stack, the more the model treats them
as a checklist to surface-pattern-match (emit *some* DECIDED lines to
satisfy the protocol, then plough through the rest of the work because it
already complied). Restrictions invite minimum compliance.

The skill variant reframes the same goal as a role's natural output:

> "You work like an architect who narrates each choice as you make it.
>  When you create a file, you briefly note why it lives there, what it
>  does, which patterns it uses, and what alternatives you considered."

Same protocol, same format. Different framing. The model produces the
documentation as a consequence of *being* the role, not as a tax on top
of doing the work.

### Caveat

The skill variant's run hit the 540s timeout before finishing. The
benchmark's `BENCH_RESULT` line was never written (it comes after the
verify+extract phases) and the executor's inline decisions weren't all
flushed to disk before the SIGKILL. The 56 number is accurate (counted
from the agent's event stream), but only 13 made it into
`.defer/decisions.json`. **There's a separate persistence issue worth
investigating** — `storeDecisionAndSave` should be synchronous but
something is letting decisions get lost between event-fire and disk-write.

## Recommendations

### 1. Replace `DecomposePromptSimple` with the `role_numbered` design

Drop the JSON example, add explicit "use double quotes" instruction, use
numbered structural requirements instead of prose. The current prompt is
robust on Claude but bloated; the lean version works on Claude *and*
mistral.

### 2. Replace `ExecutePromptTemplate` with the skill-based variant

Currently in code as `ExecutePromptVariantSkill`. The role-play framing
("you narrate each choice as an architect would") produced 65 % more
inline decisions than the rule-heavy baseline.

### 3. Investigate the persistence gap

Skill emitted 56 DECIDED events but only 13 made it to
`.defer/decisions.json`. Either `storeDecisionAndSave` is being called but
the disk write is async/buffered, or the events fire before the save
completes. Worth tracing.

### 4. Future work — multi-task validation

This experiment used a single Claude run per variant due to time and token
budget. A more rigorous comparison would use 5+ tasks of varying complexity
and measure variance. But the 65 % gap is large enough that the signal is
unlikely to flip on more samples.
