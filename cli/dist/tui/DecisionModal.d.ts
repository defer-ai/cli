import type { AgentState } from "../agents/agent.js";
interface Props {
    agent: AgentState;
    onAnswer: (value: string) => void;
    onDone: () => void;
    onAsk: (decisionId: string, question: string) => void;
    onRevise: (decisionId: string, newAnswer: string) => void;
    onUndo: (decisionIdx: number) => void;
    onWhy: (decisionId: string, optionLabel: string) => void;
    onSuggest: (decisionId: string) => void;
    focusId?: string | null;
    rows: number;
}
export declare function DecisionModal({ agent, onAnswer, onDone, onAsk, onRevise, onUndo, onWhy, onSuggest, focusId, rows, }: Props): import("react/jsx-runtime").JSX.Element;
export {};
