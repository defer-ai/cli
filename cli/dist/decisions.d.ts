export interface Decision {
    id: string;
    category: string;
    question: string;
    answer: string;
    date: string;
}
export declare function decisionsExist(cwd: string): boolean;
export declare function createDecisionsFile(cwd: string): void;
export declare function parseDecisions(cwd: string): Decision[];
export declare function addDecision(cwd: string, decision: Omit<Decision, "id">): Decision;
export declare function updateDecision(cwd: string, id: string, newAnswer: string): void;
