import type { LLMProvider, Message, StreamEvent } from "./types.js";
export declare class AnthropicProvider implements LLMProvider {
    name: string;
    private client;
    private model;
    constructor(model?: string);
    isConfigured(): boolean;
    private getClient;
    stream(systemPrompt: string, messages: Message[]): AsyncIterable<StreamEvent>;
}
