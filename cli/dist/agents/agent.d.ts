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
    private cwd;
    constructor(id: string, task: string, provider: LLMProvider, onUpdate: (state: AgentState) => void, cwd?: string);
    private update;
    /** Persist current decisions to DECISIONS.md */
    private persistDecisions;
    /** Save session state so it can be resumed later */
    private saveSession;
    /** Try to load a previous session */
    static loadSession(provider: LLMProvider, onUpdate: (state: AgentState) => void, cwd?: string): Agent | null;
    start(): Promise<void>;
    sendUserMessage(content: string): Promise<void>;
    revisitDecision(decisionId: string, newAnswer: string): Promise<void>;
    private runCompletion;
    private parseDecisionsFromOutput;
}
