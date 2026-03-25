import { parseDecisionsFromString, nextDecisionId } from "../decisions.js";
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
    constructor(id, task, provider, onUpdate) {
        this.provider = provider;
        this.onUpdate = onUpdate;
        this.state = {
            id,
            task,
            status: "idle",
            phase: "decomposing",
            decisions: [],
            messages: [],
            currentOutput: "",
        };
    }
    update(partial) {
        Object.assign(this.state, partial);
        this.onUpdate(this.state);
    }
    async start() {
        this.update({ status: "thinking" });
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
                    return;
                }
            }
            this.state.messages.push({ role: "assistant", content: fullResponse });
            // Parse any decisions from the response
            const newDecisions = this.parseDecisionsFromOutput(fullResponse);
            if (newDecisions.length > 0) {
                this.state.decisions.push(...newDecisions);
            }
            // Determine phase based on content
            const hasQuestions = /\*\*Q\d+:/.test(fullResponse);
            const isExecuting = !hasQuestions && this.state.phase === "confirming";
            if (hasQuestions) {
                this.update({ status: "asking", phase: "decomposing" });
            }
            else if (isExecuting) {
                this.update({ status: "executing", phase: "executing" });
            }
            else {
                this.update({ status: "done" });
            }
        }
        catch (err) {
            this.update({
                status: "error",
                error: err instanceof Error ? err.message : String(err),
            });
        }
    }
    parseDecisionsFromOutput(output) {
        // Try to parse decision table rows from AI output
        const tableDecisions = parseDecisionsFromString(output);
        if (tableDecisions.length > 0)
            return tableDecisions;
        // Also try to extract from Q&A format and create provisional decisions
        const questions = [];
        const questionRegex = /\*\*Q(\d+):\s*(.+?)\*\*/g;
        let match;
        const today = new Date().toISOString().split("T")[0];
        // Find current category context
        let currentCategory = "General";
        const lines = output.split("\n");
        for (const line of lines) {
            const catMatch = line.match(/^##\s+(.+)/);
            if (catMatch && !catMatch[1].startsWith("[")) {
                currentCategory = catMatch[1].trim();
            }
            const qMatch = line.match(/\*\*Q(\d+):\s*(.+?)\*\*/);
            if (qMatch) {
                const existingIds = [
                    ...this.state.decisions.map((d) => d.id),
                    ...questions.map((d) => d.id),
                ];
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
}
