import type { Decision } from "./decisions.js";
export interface HistoryEntry {
    task: string;
    decisions: Decision[];
    completedAt: string;
    cost: number;
    duration: number;
}
/** Save a completed session to history. */
export declare function saveToHistory(task: string, decisions: Decision[], cost: number, duration: number): string;
/** List recent history entries, newest first. */
export declare function listHistory(limit?: number): string[];
/** Load a specific history entry by filename. Returns null if not found. */
export declare function loadHistoryEntry(filename: string): HistoryEntry | null;
