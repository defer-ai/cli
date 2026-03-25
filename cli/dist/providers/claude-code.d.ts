import type { LLMProvider, Message, StreamEvent } from "./types.js";
export declare class ClaudeCodeProvider implements LLMProvider {
    name: string;
    private claudePath;
    sessionId: string | null;
    model: string;
    isConfigured(): boolean;
    getClaudePath(): string | null;
    setModel(model: string): void;
    stream(systemPrompt: string, messages: Message[]): AsyncIterable<StreamEvent>;
}
