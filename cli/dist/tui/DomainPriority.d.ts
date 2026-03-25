import type { Decision } from "../decisions.js";
export type CareLevel = "skip" | "low" | "medium" | "high" | "paranoid";
interface Props {
    decisions: Decision[];
    onComplete: (priorities: Record<string, CareLevel>) => void;
    rows: number;
}
export declare function DomainPriority({ decisions, onComplete, rows }: Props): import("react/jsx-runtime").JSX.Element;
export {};
