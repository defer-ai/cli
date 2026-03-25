import Anthropic from "@anthropic-ai/sdk";
export class AnthropicProvider {
    name = "Anthropic (Claude)";
    client = null;
    model;
    constructor(model = "claude-sonnet-4-20250514") {
        this.model = model;
    }
    isConfigured() {
        return !!process.env.ANTHROPIC_API_KEY;
    }
    getClient() {
        if (!this.client) {
            this.client = new Anthropic();
        }
        return this.client;
    }
    async *stream(systemPrompt, messages) {
        const client = this.getClient();
        const stream = client.messages.stream({
            model: this.model,
            max_tokens: 8192,
            system: systemPrompt,
            messages: messages.map((m) => ({
                role: m.role,
                content: m.content,
            })),
        });
        for await (const event of stream) {
            if (event.type === "content_block_delta" &&
                event.delta.type === "text_delta") {
                yield { type: "text", content: event.delta.text };
            }
        }
        yield { type: "done", content: "" };
    }
}
