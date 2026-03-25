import type { AgentState } from "../agents/agent.js";
interface Props {
    agents: AgentState[];
    selectedId: string | null;
    onSelect: (id: string) => void;
    onClose: () => void;
    rows: number;
}
export declare function DashboardOverlay({ agents, selectedId, onSelect, onClose, rows, }: Props): import("react/jsx-runtime").JSX.Element;
export {};
