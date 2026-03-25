import { Agent, type AgentState } from "./agent.js";
import type { LLMProvider } from "../providers/types.js";

export class AgentManager {
  private agents: Map<string, Agent> = new Map();
  private nextId = 1;
  private provider: LLMProvider;
  private onUpdate: (agents: AgentState[]) => void;

  constructor(
    provider: LLMProvider,
    onUpdate: (agents: AgentState[]) => void
  ) {
    this.provider = provider;
    this.onUpdate = onUpdate;
  }

  private emitUpdate(): void {
    this.onUpdate(this.getAllStates());
  }

  spawn(task: string): Agent {
    const id = `agent-${this.nextId++}`;
    const agent = new Agent(id, task, this.provider, () => {
      this.emitUpdate();
    });
    this.agents.set(id, agent);
    this.emitUpdate();
    return agent;
  }

  get(id: string): Agent | undefined {
    return this.agents.get(id);
  }

  getAllStates(): AgentState[] {
    return Array.from(this.agents.values()).map((a) => a.state);
  }

  getActiveCount(): number {
    return Array.from(this.agents.values()).filter(
      (a) => a.state.status !== "done" && a.state.status !== "error"
    ).length;
  }
}
