import { writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { loadStore, saveStore, createStore, nextDecisionId, } from "../decisions.js";
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
    state;
    provider;
    onUpdate;
    cwd;
    store;
    constructor(id, task, provider, onUpdate, cwd) {
        this.provider = provider;
        this.onUpdate = onUpdate;
        this.cwd = cwd || process.cwd();
        // Load or create store
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
        };
    }
    update(partial) {
        Object.assign(this.state, partial);
        this.onUpdate(this.state);
    }
    persist() {
        this.store.decisions = this.state.decisions;
        saveStore(this.cwd, this.store);
    }
    saveSession() {
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
    static loadSession(provider, onUpdate, cwd) {
        const dir = cwd || process.cwd();
        const sessionFile = join(dir, ".defer", "session.json");
        if (!existsSync(sessionFile))
            return null;
        try {
            const raw = readFileSync(sessionFile, "utf-8");
            const session = JSON.parse(raw);
            const agent = new Agent(session.agentId, session.task, provider, onUpdate, dir);
            agent.state.messages = session.messages || [];
            agent.state.phase = session.phase || "decomposing";
            const hasPending = agent.state.decisions.some((d) => d.answer === null);
            agent.state.status = hasPending ? "asking" : "done";
            // Build options from pending decisions
            if (hasPending) {
                agent.state.parsedOptions = agent.buildOptionsFromDecisions();
            }
            return agent;
        }
        catch {
            return null;
        }
    }
    buildOptionsFromDecisions() {
        // Find first pending decision and return its options
        const pending = this.state.decisions.find((d) => d.answer === null);
        if (!pending || pending.options.length === 0)
            return [];
        return pending.options.map((o) => ({
            label: `${o.key}) ${o.label}`,
            value: o.key,
        }));
    }
    async start() {
        this.update({ status: "thinking" });
        const hasPending = this.state.decisions.some((d) => d.answer === null);
        if (hasPending) {
            const parsedOptions = this.buildOptionsFromDecisions();
            this.update({ status: "asking", phase: "decomposing", parsedOptions });
            return;
        }
        this.state.messages.push({
            role: "user",
            content: this.state.task,
        });
        await this.runCompletion();
    }
    async sendUserMessage(content) {
        // Check if this is answering a pending decision
        const pending = this.state.decisions.find((d) => d.answer === null);
        if (pending) {
            // Match option key (single letter)
            const optionMatch = content.trim().match(/^([A-Z])$/i);
            if (optionMatch) {
                const key = optionMatch[1].toUpperCase();
                const option = pending.options.find((o) => o.key === key);
                if (option) {
                    const isDelegate = option.label.toLowerCase().includes("choose for me");
                    pending.answer = option.label;
                    pending.delegated = isDelegate;
                    pending.date = new Date().toISOString().split("T")[0];
                    this.persist();
                    // Check if more pending
                    const nextPending = this.state.decisions.find((d) => d.answer === null);
                    if (nextPending) {
                        const parsedOptions = this.buildOptionsFromDecisions();
                        this.update({
                            decisions: this.state.decisions,
                            parsedOptions,
                        });
                        return;
                    }
                    // All answered - send summary to AI
                    const summary = this.state.decisions
                        .map((d) => `${d.id}: ${d.question} -> ${d.delegated ? "DELEGATED: " : ""}${d.answer}`)
                        .join("\n");
                    this.state.messages.push({
                        role: "user",
                        content: `Here are my answers:\n${summary}\n\nPlease confirm the decision record then proceed.`,
                    });
                    this.update({
                        status: "thinking",
                        currentOutput: "",
                        parsedOptions: [],
                        phase: "confirming",
                    });
                    await this.runCompletion();
                    return;
                }
            }
            // Free text answer
            pending.answer = content.trim();
            pending.date = new Date().toISOString().split("T")[0];
            this.persist();
            const nextPending = this.state.decisions.find((d) => d.answer === null);
            if (nextPending) {
                const parsedOptions = this.buildOptionsFromDecisions();
                this.update({
                    decisions: this.state.decisions,
                    parsedOptions,
                });
                return;
            }
            // All done
            const summary = this.state.decisions
                .map((d) => `${d.id}: ${d.question} -> ${d.delegated ? "DELEGATED: " : ""}${d.answer}`)
                .join("\n");
            this.state.messages.push({
                role: "user",
                content: `Here are my answers:\n${summary}\n\nPlease confirm the decision record then proceed.`,
            });
            this.update({
                status: "thinking",
                currentOutput: "",
                parsedOptions: [],
                phase: "confirming",
            });
            await this.runCompletion();
            return;
        }
        // Regular message
        this.state.messages.push({ role: "user", content });
        this.update({ status: "thinking", currentOutput: "" });
        await this.runCompletion();
    }
    async revisitDecision(decisionId, newAnswer) {
        const decision = this.state.decisions.find((d) => d.id === decisionId);
        if (!decision)
            return;
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
    async runCompletion() {
        try {
            let fullResponse = "";
            for await (const event of this.provider.stream(DEFER_SYSTEM_PROMPT, this.state.messages)) {
                if (event.type === "text") {
                    fullResponse += event.content;
                    this.update({ currentOutput: fullResponse });
                }
                else if (event.type === "error") {
                    this.update({ status: "error", error: event.content });
                    this.saveSession();
                    return;
                }
            }
            this.state.messages.push({
                role: "assistant",
                content: fullResponse,
            });
            // Parse structured decisions from JSON block
            const newDecisions = this.parseStructuredDecisions(fullResponse);
            const existingQuestions = new Set(this.state.decisions.map((d) => d.question));
            const unique = newDecisions.filter((d) => !existingQuestions.has(d.question));
            if (unique.length > 0) {
                this.state.decisions.push(...unique);
                this.persist();
            }
            const hasQuestions = unique.length > 0;
            const hasPending = this.state.decisions.some((d) => d.answer === null);
            if (hasQuestions || hasPending) {
                const parsedOptions = this.buildOptionsFromDecisions();
                this.update({
                    status: "asking",
                    phase: "decomposing",
                    parsedOptions,
                });
            }
            else if (this.state.phase === "confirming") {
                this.update({ status: "executing", phase: "executing" });
            }
            else {
                this.update({ status: "done" });
            }
            this.saveSession();
        }
        catch (err) {
            this.update({
                status: "error",
                error: err instanceof Error ? err.message : String(err),
            });
            this.saveSession();
        }
    }
    /** Parse the ```defer-decisions JSON block from AI output */
    parseStructuredDecisions(output) {
        const match = output.match(/```defer-decisions\s*\n([\s\S]*?)\n```/);
        if (!match) {
            // Fallback: try to find any JSON array of decisions
            return this.parseFallbackDecisions(output);
        }
        try {
            const raw = JSON.parse(match[1]);
            if (!Array.isArray(raw))
                return [];
            const today = new Date().toISOString().split("T")[0];
            return raw.map((item) => {
                const id = nextDecisionId(this.state.decisions);
                return {
                    id,
                    category: item.category || "General",
                    question: item.question || "",
                    options: (item.options || []).map((o) => ({
                        key: o.key,
                        label: o.label,
                    })),
                    context: item.context || "",
                    answer: null,
                    delegated: false,
                    date: today,
                };
            });
        }
        catch {
            return [];
        }
    }
    /** Fallback: parse Q&A format if no JSON block */
    parseFallbackDecisions(output) {
        const decisions = [];
        const today = new Date().toISOString().split("T")[0];
        let currentCategory = "General";
        const lines = output.split("\n");
        for (const line of lines) {
            const catMatch = line.match(/^##\s+(.+)/);
            if (catMatch && !catMatch[1].startsWith("[")) {
                currentCategory = catMatch[1].trim();
            }
            const qMatch = line.match(/\*\*Q\d+:\s*(.+?)\*\*/);
            if (qMatch) {
                decisions.push({
                    id: nextDecisionId([...this.state.decisions, ...decisions]),
                    category: currentCategory,
                    question: qMatch[1],
                    options: [],
                    context: "",
                    answer: null,
                    delegated: false,
                    date: today,
                });
            }
        }
        // Try to attach options to the last decision
        for (const line of lines) {
            const optMatch = line.match(/^[-*]\s+\*{0,2}([A-Z])[.)]\*{0,2}\.?\s*(.+)/);
            if (optMatch && decisions.length > 0) {
                const last = decisions[decisions.length - 1];
                const label = optMatch[2].trim().replace(/\*+/g, "").trim();
                if (label && !last.options.some((o) => o.key === optMatch[1])) {
                    last.options.push({ key: optMatch[1], label });
                }
            }
        }
        return decisions;
    }
}
