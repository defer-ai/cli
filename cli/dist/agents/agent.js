import { writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { parseDecisions, parseDecisionsFromString, nextDecisionId, createDecisionsFile, decisionsExist, } from "../decisions.js";
const DEFER_SYSTEM_PROMPT = `You are in DEFER MODE, a zero-autonomy protocol.

Before acting on any task:
1. Identify every decision the task requires. Group by category.
2. Ask high-level questions first. Let answers cascade. Bundle related decisions (3+ similar = one question).
3. For each decision, offer concrete options plus "Choose for me."
4. After answers, show the decision record as a markdown table for confirmation.
5. If new decisions emerge during execution, stop and ask.

IMPORTANT: Format your decision questions exactly like this so they can be parsed:

## [Category Name]

**Q1: [Question text]**
Options: A) [option] B) [option] C) Choose for me
Context: [Why this matters]

After the user confirms the decision record, proceed with execution.
When you log decisions, use this exact format:
| ID | Category | Question | Answer | Date |`;
export class Agent {
    state;
    provider;
    onUpdate;
    cwd;
    constructor(id, task, provider, onUpdate, cwd) {
        this.provider = provider;
        this.onUpdate = onUpdate;
        this.cwd = cwd || process.cwd();
        // Load existing decisions from DECISIONS.md
        const existing = decisionsExist(this.cwd)
            ? parseDecisions(this.cwd)
            : [];
        this.state = {
            id,
            task,
            status: "idle",
            phase: "decomposing",
            decisions: existing,
            messages: [],
            currentOutput: "",
            parsedOptions: [],
        };
    }
    update(partial) {
        Object.assign(this.state, partial);
        this.onUpdate(this.state);
    }
    /** Persist current decisions to DECISIONS.md */
    persistDecisions() {
        if (!decisionsExist(this.cwd)) {
            createDecisionsFile(this.cwd);
        }
        const header = `# DECISIONS.md\n\n| ID | Category | Question | Answer | Date |\n|----|----------|----------|--------|------|`;
        const rows = this.state.decisions
            .map((d) => `| ${d.id} | ${d.category} | ${d.question} | ${d.answer} | ${d.date} |`)
            .join("\n");
        writeFileSync(join(this.cwd, "DECISIONS.md"), header + "\n" + rows + "\n");
    }
    /** Save session state so it can be resumed later */
    saveSession() {
        const sessionFile = join(this.cwd, ".defer-session.json");
        const session = {
            agentId: this.state.id,
            task: this.state.task,
            status: this.state.status,
            phase: this.state.phase,
            messages: this.state.messages,
            currentOutput: this.state.currentOutput,
            savedAt: new Date().toISOString(),
        };
        writeFileSync(sessionFile, JSON.stringify(session, null, 2));
    }
    /** Try to load a previous session */
    static loadSession(provider, onUpdate, cwd) {
        const dir = cwd || process.cwd();
        const sessionFile = join(dir, ".defer-session.json");
        if (!existsSync(sessionFile))
            return null;
        try {
            const raw = readFileSync(sessionFile, "utf-8");
            const session = JSON.parse(raw);
            const agent = new Agent(session.agentId, session.task, provider, onUpdate, dir);
            // Restore conversation history
            agent.state.messages = session.messages || [];
            agent.state.phase = session.phase || "decomposing";
            agent.state.currentOutput = session.currentOutput || "";
            // Check if there are pending decisions
            const hasPending = agent.state.decisions.some((d) => d.answer === "(pending)");
            agent.state.status = hasPending ? "asking" : "done";
            return agent;
        }
        catch {
            return null;
        }
    }
    async start() {
        this.update({ status: "thinking" });
        // If resuming with pending decisions, don't re-ask the AI
        const hasPending = this.state.decisions.some((d) => d.answer === "(pending)");
        if (hasPending) {
            this.update({ status: "asking", phase: "decomposing" });
            return;
        }
        this.state.messages.push({
            role: "user",
            content: this.state.task,
        });
        await this.runCompletion();
    }
    async sendUserMessage(content) {
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
        this.persistDecisions();
        this.saveSession();
        this.update({ decisions: this.state.decisions });
        await this.sendUserMessage(msg);
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
            // Parse any decisions from the response, dedup by question text
            const newDecisions = this.parseDecisionsFromOutput(fullResponse);
            const existingQuestions = new Set(this.state.decisions.map((d) => d.question));
            const unique = newDecisions.filter((d) => !existingQuestions.has(d.question));
            if (unique.length > 0) {
                this.state.decisions.push(...unique);
                this.persistDecisions();
            }
            // Determine phase based on content
            const hasQuestions = /\*\*Q\d+:/.test(fullResponse);
            const isExecuting = !hasQuestions && this.state.phase === "confirming";
            if (hasQuestions) {
                const parsedOptions = this.parseOptionsFromOutput(fullResponse);
                this.update({ status: "asking", phase: "decomposing", parsedOptions });
            }
            else if (isExecuting) {
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
    parseDecisionsFromOutput(output) {
        // Try to parse decision table rows from AI output
        const tableDecisions = parseDecisionsFromString(output);
        if (tableDecisions.length > 0)
            return tableDecisions;
        // Extract from Q&A format as provisional decisions
        const questions = [];
        const today = new Date().toISOString().split("T")[0];
        let currentCategory = "General";
        const lines = output.split("\n");
        for (const line of lines) {
            const catMatch = line.match(/^##\s+(.+)/);
            if (catMatch && !catMatch[1].startsWith("[")) {
                currentCategory = catMatch[1].trim();
            }
            const qMatch = line.match(/\*\*Q(\d+):\s*(.+?)\*\*/);
            if (qMatch) {
                const id = nextDecisionId([
                    ...this.state.decisions,
                    ...questions,
                ]);
                questions.push({
                    id,
                    category: currentCategory,
                    question: qMatch[2],
                    answer: "(pending)",
                    date: today,
                });
            }
        }
        return questions;
    }
    /** Extract selectable options from AI output */
    parseOptionsFromOutput(output) {
        const options = [];
        const seen = new Set();
        const lines = output.split("\n");
        for (const line of lines) {
            // Match: "- **A.** Description" or "- **A)** Description"
            const mdMatch = line.match(/^[-*]\s+\*{0,2}([A-Z])[.)]\*{0,2}\.?\s*(.+)/);
            if (mdMatch) {
                const key = mdMatch[1];
                const label = mdMatch[2].trim().replace(/\*+/g, "").trim();
                if (label && !seen.has(key)) {
                    seen.add(key);
                    options.push({ label: `${key}) ${label}`, value: key });
                }
                continue;
            }
            // Match: "A) Description" or "A. Description" (inline options)
            const inlineMatch = line.match(/^\s*([A-Z])[.)]\s+(.+)/);
            if (inlineMatch) {
                const key = inlineMatch[1];
                const label = inlineMatch[2].trim().replace(/\*+/g, "").trim();
                if (label && !seen.has(key)) {
                    seen.add(key);
                    options.push({ label: `${key}) ${label}`, value: key });
                }
                continue;
            }
            // Match: "Options: A) foo B) bar C) baz" on a single line
            if (/options:/i.test(line)) {
                const optRegex = /([A-Z])\)\s*([^A-Z)]+?)(?=\s+[A-Z]\)|\s*$)/g;
                let match;
                while ((match = optRegex.exec(line)) !== null) {
                    const key = match[1];
                    const label = match[2].trim();
                    if (label && !seen.has(key)) {
                        seen.add(key);
                        options.push({ label: `${key}) ${label}`, value: key });
                    }
                }
            }
        }
        return options;
    }
}
