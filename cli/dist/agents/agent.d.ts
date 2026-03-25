import type { LLMProvider, Message } from "../providers/types.js";
import type { Decision } from "../decisions.js";
export type AgentStatus = "idle" | "thinking" | "asking" | "executing" | "done" | "error";
export type AgentPhase = "decomposing" | "confirming" | "executing";
export interface AgentState {
    id: string;
    task: string;
    status: AgentStatus;
    phase: AgentPhase;
    decisions: Decision[];
    messages: Message[];
    currentOutput: string;
    error?: string;
}
export declare class Agent {
    state: AgentState;
    private provider;
    private onUpdate;
    constructor(id: string, task: string, provider: LLMProvider, onUpdate: (state: AgentState) => void);
    private update;
    start(): Promise<void>;
    sendUserMessage(content: string): Promise<void>;
    revisitDecision(decisionId: string, newAnswer: string): Promise<void>;
    private runCompletion;
    private parseDecisionsFromOutput;
}
