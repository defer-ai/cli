import { Agent, type AgentState } from "./agent.js";
import type { LLMProvider } from "../providers/types.js";
export declare class AgentManager {
    private agents;
    private nextId;
    private provider;
    private onUpdate;
    constructor(provider: LLMProvider, onUpdate: (agents: AgentState[]) => void);
    private emitUpdate;
    spawn(task: string): Agent;
    get(id: string): Agent | undefined;
    getAllStates(): AgentState[];
    getActiveCount(): number;
}
