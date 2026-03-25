import { Agent } from "./agent.js";
export class AgentManager {
    agents = new Map();
    nextId = 1;
    provider;
    onUpdate;
    constructor(provider, onUpdate) {
        this.provider = provider;
        this.onUpdate = onUpdate;
    }
    emitUpdate() {
        this.onUpdate(this.getAllStates());
    }
    spawn(task) {
        const id = `agent-${this.nextId++}`;
        const agent = new Agent(id, task, this.provider, () => {
            this.emitUpdate();
        });
        this.agents.set(id, agent);
        this.emitUpdate();
        return agent;
    }
    get(id) {
        return this.agents.get(id);
    }
    getAllStates() {
        return Array.from(this.agents.values()).map((a) => a.state);
    }
    getActiveCount() {
        return Array.from(this.agents.values()).filter((a) => a.state.status !== "done" && a.state.status !== "error").length;
    }
}
