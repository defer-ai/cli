import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
export class ClaudeCodeProvider {
    name = "Claude Code";
    isConfigured() {
        // Check if `claude` binary exists
        try {
            const { execSync } = require("node:child_process");
            execSync("which claude", { stdio: "pipe" });
            return true;
        }
        catch {
            return false;
        }
    }
    async *stream(systemPrompt, messages) {
        // Build the prompt from messages
        // For multi-turn, we send the full conversation as context
        const lastUserMessage = messages.filter((m) => m.role === "user").pop();
        if (!lastUserMessage) {
            yield { type: "error", content: "No user message" };
            return;
        }
        // Build context from previous messages
        let prompt = lastUserMessage.content;
        if (messages.length > 1) {
            const context = messages.slice(0, -1).map((m) => {
                const prefix = m.role === "user" ? "Human" : "Assistant";
                return `${prefix}: ${m.content}`;
            }).join("\n\n");
            prompt = `Previous conversation:\n${context}\n\nCurrent request: ${prompt}`;
        }
        const args = [
            "-p",
            "--output-format", "stream-json",
            "--system-prompt", systemPrompt,
            prompt,
        ];
        const child = spawn("claude", args, {
            stdio: ["pipe", "pipe", "pipe"],
            env: { ...process.env },
        });
        const rl = createInterface({ input: child.stdout });
        for await (const line of rl) {
            if (!line.trim())
                continue;
            try {
                const event = JSON.parse(line);
                if (event.type === "assistant" && event.message?.content) {
                    for (const block of event.message.content) {
                        if (block.type === "text") {
                            yield { type: "text", content: block.text };
                        }
                    }
                }
                if (event.type === "result") {
                    if (event.result?.content) {
                        // Final result text
                        for (const block of event.result.content) {
                            if (block.type === "text" && block.text) {
                                yield { type: "text", content: block.text };
                            }
                        }
                    }
                }
                if (event.type === "error") {
                    yield {
                        type: "error",
                        content: event.error?.message || "Unknown error",
                    };
                    return;
                }
            }
            catch {
                // Skip unparseable lines
            }
        }
        // Wait for process to exit
        await new Promise((resolve) => {
            child.on("close", () => resolve());
        });
        yield { type: "done", content: "" };
    }
}
