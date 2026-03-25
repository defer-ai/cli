import { spawn, execSync } from "node:child_process";
import { existsSync } from "node:fs";
import { createInterface } from "node:readline";
import { join } from "node:path";
function findClaude() {
    const home = process.env.HOME || "";
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
    sessionId = null;
    model = "sonnet";
    isConfigured() {
        this.claudePath = findClaude();
        return this.claudePath !== null;
    }
    getClaudePath() {
        return this.claudePath;
    }
    setModel(model) {
        this.model = model;
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
        // Only include context if this is a new session (no resume)
        if (!this.sessionId && messages.length > 1) {
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
            "--model",
            this.model,
        ];
        // Resume session if we have one
        if (this.sessionId) {
            args.push("--resume", this.sessionId);
        }
        else {
            args.push("--system-prompt", systemPrompt);
        }
        args.push(prompt);
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
                // System init event - capture session ID
                if (event.type === "system" && event.session_id) {
                    this.sessionId = event.session_id;
                    continue;
                }
                // Content block deltas (streaming text chunks)
                if (event.type === "content_block_delta") {
                    if (event.delta?.type === "text_delta" && event.delta.text) {
                        yield { type: "text", content: event.delta.text };
                        textEmitted = true;
                    }
                    continue;
                }
                // Assistant message - only if no deltas
                if (event.type === "assistant" &&
                    !textEmitted &&
                    event.message?.content) {
                    for (const block of event.message.content) {
                        if (block.type === "text") {
                            yield { type: "text", content: block.text };
                            textEmitted = true;
                        }
                    }
                    continue;
                }
                // Result - capture session ID
                if (event.type === "result") {
                    if (event.session_id) {
                        this.sessionId = event.session_id;
                    }
                    if (event.result?.session_id) {
                        this.sessionId = event.result.session_id;
                    }
                    if (!textEmitted) {
                        const text = typeof event.result === "string"
                            ? event.result
                            : typeof event.result?.result === "string"
                                ? event.result.result
                                : event.result?.content
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
