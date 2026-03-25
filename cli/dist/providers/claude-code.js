import { spawn, execSync } from "node:child_process";
import { existsSync } from "node:fs";
import { createInterface } from "node:readline";
import { join } from "node:path";
function findClaude() {
    const home = process.env.HOME || "";
    // Try direct known paths first (fastest)
    const knownPaths = [
        join(home, ".local", "bin", "claude"),
        join(home, ".npm-global", "bin", "claude"),
        "/usr/local/bin/claude",
        "/usr/bin/claude",
    ];
    for (const p of knownPaths) {
        if (existsSync(p))
            return p;
    }
    // Fall back to which
    try {
        return execSync("which claude", {
            stdio: "pipe",
            encoding: "utf-8",
            shell: process.env.SHELL || "/bin/sh",
        }).trim();
    }
    catch {
        return null;
    }
}
export class ClaudeCodeProvider {
    name = "Claude Code";
    claudePath = null;
    isConfigured() {
        this.claudePath = findClaude();
        return this.claudePath !== null;
    }
    async *stream(systemPrompt, messages) {
        if (!this.claudePath) {
            yield { type: "error", content: "claude binary not found" };
            return;
        }
        const lastUserMessage = messages.filter((m) => m.role === "user").pop();
        if (!lastUserMessage) {
            yield { type: "error", content: "No user message" };
            return;
        }
        let prompt = lastUserMessage.content;
        if (messages.length > 1) {
            const context = messages
                .slice(0, -1)
                .map((m) => {
                const prefix = m.role === "user" ? "Human" : "Assistant";
                return `${prefix}: ${m.content}`;
            })
                .join("\n\n");
            prompt = `Previous conversation:\n${context}\n\nCurrent request: ${prompt}`;
        }
        const args = [
            "-p",
            "--verbose",
            "--output-format",
            "stream-json",
            "--system-prompt",
            systemPrompt,
            prompt,
        ];
        const child = spawn(this.claudePath, args, {
            stdio: ["pipe", "pipe", "pipe"],
            env: { ...process.env },
        });
        const rl = createInterface({ input: child.stdout });
        let textEmitted = false;
        for await (const line of rl) {
            if (!line.trim())
                continue;
            try {
                const event = JSON.parse(line);
                // Content block deltas (streaming text chunks) - preferred
                if (event.type === "content_block_delta") {
                    if (event.delta?.type === "text_delta" && event.delta.text) {
                        yield { type: "text", content: event.delta.text };
                        textEmitted = true;
                    }
                    continue;
                }
                // Assistant message with full content - only if no deltas came
                if (event.type === "assistant" && !textEmitted && event.message?.content) {
                    for (const block of event.message.content) {
                        if (block.type === "text") {
                            yield { type: "text", content: block.text };
                            textEmitted = true;
                        }
                    }
                    continue;
                }
                // Final result - only if nothing else emitted text
                if (event.type === "result") {
                    if (!textEmitted && event.result) {
                        const text = typeof event.result === "string"
                            ? event.result
                            : typeof event.result.result === "string"
                                ? event.result.result
                                : event.result.content
                                    ?.filter((b) => b.type === "text")
                                    .map((b) => b.text)
                                    .join("") || "";
                        if (text) {
                            yield { type: "text", content: text };
                        }
                    }
                    continue;
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
        await new Promise((resolve) => {
            child.on("close", () => resolve());
        });
        yield { type: "done", content: "" };
    }
}
