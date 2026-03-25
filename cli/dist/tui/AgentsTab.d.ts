import type { AgentState } from "../agents/agent.js";
interface Props {
    agents: AgentState[];
    selectedId: string | null;
    onSelect: (id: string) => void;
}
export declare function AgentsTab({ agents, selectedId, onSelect }: Props): import("react/jsx-runtime").JSX.Element;
export {};
