import { writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { loadStore, saveStore, createStore, nextDecisionId, } from "../decisions.js";
const DEFER_SYSTEM_PROMPT = `You are in DEFER MODE, a zero-autonomy protocol.

YOUR ONLY JOB on the first response is to output decisions. Do NOT write code. Do NOT explain. Do NOT discuss. Just output the decisions.

Rules:
1. Identify every decision the task requires. Group by category.
2. Ask high-level first. Let answers cascade. Bundle related decisions.
3. Every decision MUST have concrete options plus "Choose for me" as the last option.
4. After the user answers, confirm the decision record, then execute.
5. If new decisions emerge during execution, stop and output more decisions.

You MUST output a \`\`\`defer-decisions JSON block. This is not optional. If you respond without this block, the CLI cannot parse your decisions and the user sees nothing.

Each decision has a "category" field. Use short, descriptive category names like "Stack", "Data", "API", "Auth", "UI", "DevEx", etc.

FORMAT — you must output EXACTLY this structure:

\`\`\`defer-decisions
[
  {
    "category": "Stack",
    "question": "Backend language and framework?",
    "options": [
      {"key": "A", "label": "Node.js with Express"},
      {"key": "B", "label": "Python with FastAPI"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Determines the entire backend ecosystem"
  }
]
\`\`\`

Rules for the JSON:
- "category": short name (will be used to generate IDs like STACK-001, DATA-001)
- "question": clear, specific question
- "options": 2-6 options, each with "key" (single uppercase letter) and "label". Last option must be "Choose for me"
- "context": one sentence explaining why this decision matters

You may include a brief human-readable summary before the JSON block, but the JSON block is MANDATORY.`;
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
            totalCost: 0,
            totalTokens: 0,
            startedAt: Date.now(),
        };
    }
    update(partial) {
        this.state = { ...this.state, ...partial };
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
    /** Find the index of the next pending decision */
    findNextPendingIndex(afterIndex = -1) {
        for (let i = afterIndex + 1; i < this.state.decisions.length; i++) {
            if (this.state.decisions[i].answer === null)
                return i;
        }
        return -1;
    }
    buildOptionsForDecision(idx) {
        const d = this.state.decisions[idx];
        if (!d || d.options.length === 0)
            return [];
        return d.options.map((o) => ({
            label: `${o.key}) ${o.label}`,
            value: o.key,
        }));
    }
    moveToNextPending() {
        const nextIdx = this.findNextPendingIndex(this.state.pendingIndex);
        if (nextIdx >= 0) {
            const parsedOptions = this.buildOptionsForDecision(nextIdx);
            this.update({
                decisions: this.state.decisions,
                pendingIndex: nextIdx,
                parsedOptions,
                status: "asking",
            });
        }
        else {
            // All answered, send to AI for confirmation
            const summary = this.state.decisions
                .map((d) => `${d.id}: ${d.question} -> ${d.delegated ? "DELEGATED: " : ""}${d.answer}`)
                .join("\n");
            this.state.messages.push({
                role: "user",
                content: `Task: ${this.state.task}\n\nDecision record:\n${summary}\n\nAll decisions are answered. Proceed with implementation based on these decisions.`,
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
            const pendingIdx = agent.findNextPendingIndex();
            if (pendingIdx >= 0) {
                agent.state.status = "asking";
                agent.state.pendingIndex = pendingIdx;
                agent.state.parsedOptions =
                    agent.buildOptionsForDecision(pendingIdx);
            }
            else {
                agent.state.status = session.status === "executing" ? "executing" : "done";
            }
            return agent;
        }
        catch {
            return null;
        }
    }
    async start() {
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
    async sendUserMessage(content) {
        // Check if this is answering a pending decision
        const pendingIdx = this.state.pendingIndex;
        const pending = pendingIdx >= 0 ? this.state.decisions[pendingIdx] : null;
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
    retryCount = 0;
    static MAX_RETRIES = 2;
    async runCompletion() {
        try {
            let fullResponse = "";
            for await (const event of this.provider.stream(DEFER_SYSTEM_PROMPT, this.state.messages)) {
                if (event.type === "text") {
                    fullResponse += event.content;
                    this.update({ currentOutput: fullResponse });
                }
                else if (event.type === "cost" && event.cost) {
                    this.update({
                        totalCost: this.state.totalCost + event.cost.totalCost,
                        totalTokens: this.state.totalTokens +
                            event.cost.inputTokens +
                            event.cost.outputTokens,
                    });
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
            // Parse structured decisions
            const newDecisions = this.parseStructuredDecisions(fullResponse);
            const existingQuestions = new Set(this.state.decisions.map((d) => d.question));
            const unique = newDecisions.filter((d) => !existingQuestions.has(d.question));
            // If this was supposed to be a decomposition but no decisions came back, retry
            if (unique.length === 0 &&
                this.state.phase === "decomposing" &&
                this.state.decisions.length === 0 &&
                this.retryCount < Agent.MAX_RETRIES) {
                this.retryCount++;
                this.state.messages.push({
                    role: "user",
                    content: "You did not output a ```defer-decisions JSON block. This is required. Please list all decisions as structured questions and include the ```defer-decisions JSON block as specified in your instructions.",
                });
                this.update({ currentOutput: "Retrying: no decisions were generated..." });
                await this.runCompletion();
                return;
            }
            if (unique.length > 0) {
                this.state.decisions.push(...unique);
                this.persist();
                this.retryCount = 0;
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
            }
            else if (this.state.phase === "confirming" ||
                this.state.phase === "executing") {
                this.update({ status: "done", phase: "executing" });
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
            return this.parseFallbackDecisions(output);
        }
        try {
            const raw = JSON.parse(match[1]);
            if (!Array.isArray(raw))
                return [];
            const today = new Date().toISOString().split("T")[0];
            const result = [];
            for (const item of raw) {
                // Use accumulated result + existing decisions for ID generation
                const cat = item.category || "General";
                const id = nextDecisionId([...this.state.decisions, ...result], cat);
                result.push({
                    id,
                    category: cat,
                    question: item.question || "",
                    options: (item.options || []).map((o) => ({
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
        let lastDecision = null;
        const lines = output.split("\n");
        for (const line of lines) {
            const catMatch = line.match(/^##\s+(.+)/);
            if (catMatch && !catMatch[1].startsWith("[")) {
                currentCategory = catMatch[1].trim();
                continue;
            }
            const qMatch = line.match(/\*\*Q\d+:\s*(.+?)\*\*/);
            if (qMatch) {
                const d = {
                    id: nextDecisionId([...this.state.decisions, ...decisions], currentCategory),
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
                const optMatch = line.match(/^[-*]\s+\*{0,2}([A-Z])[.)]\*{0,2}\.?\s*(.+)/);
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
