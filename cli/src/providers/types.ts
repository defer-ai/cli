export interface Message {
  role: "user" | "assistant";
  content: string;
}

export interface StreamEvent {
  type: "text" | "done" | "error" | "cost";
  content: string;
  cost?: {
    totalCost: number;
    inputTokens: number;
    outputTokens: number;
  };
}

export interface LLMProvider {
  name: string;

  /** Send a message and get a streaming response */
  stream(
    systemPrompt: string,
    messages: Message[]
  ): AsyncIterable<StreamEvent>;

  /** Check if the provider is configured (API key exists, etc.) */
  isConfigured(): boolean;
}
