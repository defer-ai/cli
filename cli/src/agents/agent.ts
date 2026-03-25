import { writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import type { LLMProvider, Message, StreamEvent } from "../providers/types.js";
import type { Decision, DecisionOption, DecisionStore } from "../decisions.js";
import {
  loadStore,
  saveStore,
  createStore,
  nextDecisionId,
} from "../decisions.js";

export type AgentStatus =
  | "idle"
  | "thinking"
  | "asking"
  | "executing"
  | "done"
  | "error";
export type AgentPhase = "decomposing" | "confirming" | "executing";

export interface ParsedOption {
  label: string;
  value: string;
}

export interface AgentState {
  id: string;
  task: string;
  status: AgentStatus;
  phase: AgentPhase;
  decisions: Decision[];
  messages: Message[];
  currentOutput: string;
  parsedOptions: ParsedOption[];
  /** Index of the currently active pending decision */
  pendingIndex: number;
  error?: string;
}

const DEFER_SYSTEM_PROMPT = `You are in DEFER MODE, a zero-autonomy protocol.

Before acting on any task:
1. Identify every decision the task requires. Group by category.
2. Ask high-level first. Let answers cascade. Bundle related decisions.
3. For each decision, offer concrete options plus "Choose for me."
4. After answers, confirm the decision record before executing.
5. If new decisions emerge during execution, stop and ask.

CRITICAL: After listing your questions in human-readable form, also output a JSON block that the CLI can parse. Wrap it in \`\`\`defer-decisions tags:

\`\`\`defer-decisions
[
  {
    "category": "Technology Stack",
    "question": "Backend language & framework?",
    "options": [
      {"key": "A", "label": "Node.js (TypeScript)"},
      {"key": "B", "label": "Python (FastAPI)"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Affects ecosystem and deployment model"
  }
]
\`\`\`

Always include this JSON block alongside your human-readable questions. Each object needs: category, question, options (array of {key, label}), context (one sentence).`;

export class Agent {
  state: AgentState;
  private provider: LLMProvider;
  private onUpdate: (state: AgentState) => void;
  private cwd: string;
  private store: DecisionStore;

  constructor(
    id: string,
    task: string,
    provider: LLMProvider,
    onUpdate: (state: AgentState) => void,
    cwd?: string
  ) {
    this.provider = provider;
    this.onUpdate = onUpdate;
    this.cwd = cwd || process.cwd();

    this.store = loadStore(this.cwd) || createStore(this.cwd, task);

    this.state = {
      id,
      task,
      status: "idle",
      phase: "decomposing",
      decisions: this.store.decisions,
      messages: [],
      currentOutput: "",
      parsedOptions: [],
      pendingIndex: -1,
    };
  }

  private update(partial: Partial<AgentState>): void {
    Object.assign(this.state, partial);
    this.onUpdate(this.state);
  }

  private persist(): void {
    this.store.decisions = this.state.decisions;
    saveStore(this.cwd, this.store);
  }

  private saveSession(): void {
    const sessionFile = join(this.cwd, ".defer", "session.json");
    const session = {
      agentId: this.state.id,
      task: this.state.task,
      status: this.state.status,
      phase: this.state.phase,
      messages: this.state.messages,
      savedAt: new Date().toISOString(),
    };
    writeFileSync(sessionFile, JSON.stringify(session, null, 2));
  }

  /** Find the index of the next pending decision */
  private findNextPendingIndex(afterIndex = -1): number {
    for (let i = afterIndex + 1; i < this.state.decisions.length; i++) {
      if (this.state.decisions[i].answer === null) return i;
    }
    return -1;
  }

  private buildOptionsForDecision(idx: number): ParsedOption[] {
    const d = this.state.decisions[idx];
    if (!d || d.options.length === 0) return [];
    return d.options.map((o) => ({
      label: `${o.key}) ${o.label}`,
      value: o.key,
    }));
  }

  private moveToNextPending(): void {
    const nextIdx = this.findNextPendingIndex(this.state.pendingIndex);
    if (nextIdx >= 0) {
      const parsedOptions = this.buildOptionsForDecision(nextIdx);
      this.update({
        decisions: this.state.decisions,
        pendingIndex: nextIdx,
        parsedOptions,
        status: "asking",
      });
    } else {
      // All answered, send to AI for confirmation
      const summary = this.state.decisions
        .map(
          (d) =>
            `${d.id}: ${d.question} -> ${d.delegated ? "DELEGATED: " : ""}${d.answer}`
        )
        .join("\n");

      this.state.messages.push({
        role: "user",
        content: `Here are my answers:\n${summary}\n\nPlease confirm the decision record then proceed with implementation.`,
      });
      this.update({
        status: "thinking",
        currentOutput: "",
        parsedOptions: [],
        pendingIndex: -1,
        phase: "confirming",
      });
      this.runCompletion();
    }
  }

  static loadSession(
    provider: LLMProvider,
    onUpdate: (state: AgentState) => void,
    cwd?: string
  ): Agent | null {
    const dir = cwd || process.cwd();
    const sessionFile = join(dir, ".defer", "session.json");

    if (!existsSync(sessionFile)) return null;

    try {
      const raw = readFileSync(sessionFile, "utf-8");
      const session = JSON.parse(raw);

      const agent = new Agent(
        session.agentId,
        session.task,
        provider,
        onUpdate,
        dir
      );

      agent.state.messages = session.messages || [];
      agent.state.phase = session.phase || "decomposing";

      const pendingIdx = agent.findNextPendingIndex();
      if (pendingIdx >= 0) {
        agent.state.status = "asking";
        agent.state.pendingIndex = pendingIdx;
        agent.state.parsedOptions =
          agent.buildOptionsForDecision(pendingIdx);
      } else {
        agent.state.status = session.status === "executing" ? "executing" : "done";
      }

      return agent;
    } catch {
      return null;
    }
  }

  async start(): Promise<void> {
    this.update({ status: "thinking" });

    const pendingIdx = this.findNextPendingIndex();
    if (pendingIdx >= 0) {
      const parsedOptions = this.buildOptionsForDecision(pendingIdx);
      this.update({
        status: "asking",
        phase: "decomposing",
        pendingIndex: pendingIdx,
        parsedOptions,
      });
      return;
    }

    this.state.messages.push({
      role: "user",
      content: this.state.task,
    });

    await this.runCompletion();
  }

  async sendUserMessage(content: string): Promise<void> {
    // Check if this is answering a pending decision
    const pendingIdx = this.state.pendingIndex;
    const pending =
      pendingIdx >= 0 ? this.state.decisions[pendingIdx] : null;

    if (pending && pending.answer === null) {
      // Match option key
      const optionMatch = content.trim().match(/^([A-Z])$/i);
      if (optionMatch) {
        const key = optionMatch[1].toUpperCase();
        const option = pending.options.find((o) => o.key === key);
        if (option) {
          pending.answer = option.label;
          pending.delegated =
            option.label.toLowerCase().includes("choose for me");
          pending.date = new Date().toISOString().split("T")[0];
          this.persist();
          this.moveToNextPending();
          return;
        }
      }

      // Free text answer
      pending.answer = content.trim();
      pending.date = new Date().toISOString().split("T")[0];
      this.persist();
      this.moveToNextPending();
      return;
    }

    // Regular message
    this.state.messages.push({ role: "user", content });
    this.update({ status: "thinking", currentOutput: "" });
    await this.runCompletion();
  }

  async revisitDecision(
    decisionId: string,
    newAnswer: string
  ): Promise<void> {
    const decision = this.state.decisions.find((d) => d.id === decisionId);
    if (!decision) return;

    const msg = `I'm changing ${decisionId} ("${decision.question}") from "${decision.answer}" to "${newAnswer}". Update everything that depends on this.`;

    decision.answer = newAnswer;
    decision.date = new Date().toISOString().split("T")[0];
    this.persist();
    this.saveSession();
    this.update({ decisions: this.state.decisions });

    this.state.messages.push({ role: "user", content: msg });
    this.update({ status: "thinking", currentOutput: "" });
    await this.runCompletion();
  }

  private async runCompletion(): Promise<void> {
    try {
      let fullResponse = "";

      for await (const event of this.provider.stream(
        DEFER_SYSTEM_PROMPT,
        this.state.messages
      )) {
        if (event.type === "text") {
          fullResponse += event.content;
          this.update({ currentOutput: fullResponse });
        } else if (event.type === "error") {
          this.update({ status: "error", error: event.content });
          this.saveSession();
          return;
        }
      }

      this.state.messages.push({
        role: "assistant",
        content: fullResponse,
      });

      // Parse structured decisions
      const newDecisions = this.parseStructuredDecisions(fullResponse);
      const existingQuestions = new Set(
        this.state.decisions.map((d) => d.question)
      );
      const unique = newDecisions.filter(
        (d) => !existingQuestions.has(d.question)
      );

      if (unique.length > 0) {
        this.state.decisions.push(...unique);
        this.persist();
      }

      const pendingIdx = this.findNextPendingIndex();

      if (pendingIdx >= 0) {
        const parsedOptions = this.buildOptionsForDecision(pendingIdx);
        this.update({
          status: "asking",
          phase: "decomposing",
          pendingIndex: pendingIdx,
          parsedOptions,
        });
      } else if (this.state.phase === "confirming" || this.state.phase === "executing") {
        this.update({ status: "done", phase: "executing" });
      } else {
        this.update({ status: "done" });
      }

      this.saveSession();
    } catch (err) {
      this.update({
        status: "error",
        error: err instanceof Error ? err.message : String(err),
      });
      this.saveSession();
    }
  }

  /** Parse the ```defer-decisions JSON block from AI output */
  private parseStructuredDecisions(output: string): Decision[] {
    const match = output.match(
      /```defer-decisions\s*\n([\s\S]*?)\n```/
    );
    if (!match) {
      return this.parseFallbackDecisions(output);
    }

    try {
      const raw = JSON.parse(match[1]);
      if (!Array.isArray(raw)) return [];

      const today = new Date().toISOString().split("T")[0];
      const result: Decision[] = [];

      for (const item of raw) {
        // Use accumulated result + existing decisions for ID generation
        const id = nextDecisionId([...this.state.decisions, ...result]);
        result.push({
          id,
          category: item.category || "General",
          question: item.question || "",
          options: (item.options || []).map((o: { key: string; label: string }) => ({
            key: o.key,
            label: o.label,
          })),
          context: item.context || "",
          answer: null,
          delegated: false,
          date: today,
        });
      }

      return result;
    } catch {
      return [];
    }
  }

  /** Fallback: parse Q&A format if no JSON block */
  private parseFallbackDecisions(output: string): Decision[] {
    const decisions: Decision[] = [];
    const today = new Date().toISOString().split("T")[0];
    let currentCategory = "General";
    let lastDecision: Decision | null = null;
    const lines = output.split("\n");

    for (const line of lines) {
      const catMatch = line.match(/^##\s+(.+)/);
      if (catMatch && !catMatch[1].startsWith("[")) {
        currentCategory = catMatch[1].trim();
        continue;
      }

      const qMatch = line.match(/\*\*Q\d+:\s*(.+?)\*\*/);
      if (qMatch) {
        const d: Decision = {
          id: nextDecisionId([...this.state.decisions, ...decisions]),
          category: currentCategory,
          question: qMatch[1],
          options: [],
          context: "",
          answer: null,
          delegated: false,
          date: today,
        };
        decisions.push(d);
        lastDecision = d;
        continue;
      }

      // Attach options to last decision
      if (lastDecision) {
        const optMatch = line.match(
          /^[-*]\s+\*{0,2}([A-Z])[.)]\*{0,2}\.?\s*(.+)/
        );
        if (optMatch) {
          const label = optMatch[2].trim().replace(/\*+/g, "").trim();
          if (label && !lastDecision.options.some((o) => o.key === optMatch[1])) {
            lastDecision.options.push({ key: optMatch[1], label });
          }
        }

        const ctxMatch = line.match(/Context:\s*(.+)/i);
        if (ctxMatch) {
          lastDecision.context = ctxMatch[1].trim();
        }
      }
    }

    return decisions;
  }
}
