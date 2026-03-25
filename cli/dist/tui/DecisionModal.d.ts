import type { AgentState } from "../agents/agent.js";
interface Props {
    agent: AgentState;
    onAnswer: (value: string) => void;
    onDone: () => void;
    rows: number;
}
export declare function DecisionModal({ agent, onAnswer, onDone, rows }: Props): import("react/jsx-runtime").JSX.Element;
export {};
