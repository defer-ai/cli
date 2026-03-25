export interface DecisionOption {
    key: string;
    label: string;
}
export interface Decision {
    id: string;
    category: string;
    question: string;
    options: DecisionOption[];
    context: string;
    answer: string | null;
    delegated: boolean;
    date: string;
}
export interface DecisionStore {
    task: string;
    decisions: Decision[];
    createdAt: string;
    updatedAt: string;
}
export declare function storeExists(cwd: string): boolean;
export declare function loadStore(cwd: string): DecisionStore | null;
export declare function saveStore(cwd: string, store: DecisionStore): void;
export declare function createStore(cwd: string, task: string): DecisionStore;
/** Generate a category-scoped ID like STACK-001, DATA-002 */
export declare function nextDecisionId(decisions: Decision[], category: string): string;
export declare function parseLegacyDecisions(cwd: string): Decision[];
