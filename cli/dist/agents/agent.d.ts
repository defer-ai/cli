import type { LLMProvider, Message } from "../providers/types.js";
import type { Decision } from "../decisions.js";
export type AgentStatus = "idle" | "thinking" | "asking" | "executing" | "done" | "error";
export type AgentPhase = "decomposing" | "confirming" | "executing";
export interface ParsedOption {
    label: string;
    value: string;
}
export interface AgentState {
    id: string;
    task: string;
    status: AgentStatus;
    phase: AgentPhase;
    decisions: Decision[];
    messages: Message[];
    currentOutput: string;
    parsedOptions: ParsedOption[];
    /** Index of the currently active pending decision */
    pendingIndex: number;
    error?: string;
}
export declare class Agent {
    state: AgentState;
    private provider;
    private onUpdate;
    private cwd;
    private store;
    constructor(id: string, task: string, provider: LLMProvider, onUpdate: (state: AgentState) => void, cwd?: string);
    private update;
    private persist;
    private saveSession;
    /** Find the index of the next pending decision */
    private findNextPendingIndex;
    private buildOptionsForDecision;
    private moveToNextPending;
    static loadSession(provider: LLMProvider, onUpdate: (state: AgentState) => void, cwd?: string): Agent | null;
    start(): Promise<void>;
    sendUserMessage(content: string): Promise<void>;
    revisitDecision(decisionId: string, newAnswer: string): Promise<void>;
    private runCompletion;
    /** Parse the ```defer-decisions JSON block from AI output */
    private parseStructuredDecisions;
    /** Fallback: parse Q&A format if no JSON block */
    private parseFallbackDecisions;
}
