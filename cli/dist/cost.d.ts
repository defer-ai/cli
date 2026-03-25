export interface CostInfo {
    totalCost: number;
    inputTokens: number;
    outputTokens: number;
    cacheReadTokens: number;
    cacheWriteTokens: number;
}
/**
 * Extract cost data from a stream-json result event.
 * Returns a partial CostInfo if the event contains cost/token fields, or null otherwise.
 */
export declare function parseCostFromEvent(event: any): Partial<CostInfo> | null;
/** Format cost info as "$0.05 | 12.3k tokens". */
export declare function formatCost(cost: CostInfo): string;
