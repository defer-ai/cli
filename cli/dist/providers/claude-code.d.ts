import type { LLMProvider, Message, StreamEvent } from "./types.js";
export declare class ClaudeCodeProvider implements LLMProvider {
    name: string;
    isConfigured(): boolean;
    stream(systemPrompt: string, messages: Message[]): AsyncIterable<StreamEvent>;
}
