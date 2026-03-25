import type { AgentState } from "../agents/agent.js";
interface Props {
    agent: AgentState;
    onAnswer: (value: string) => void;
    onDone: () => void;
    onAsk: (decisionId: string, question: string) => void;
    onRevise: (decisionId: string, newAnswer: string) => void;
    focusId?: string | null;
    rows: number;
}
export declare function DecisionModal({ agent, onAnswer, onDone, onAsk, onRevise, focusId, rows, }: Props): import("react/jsx-runtime").JSX.Element;
export {};
