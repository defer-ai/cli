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
export function parseCostFromEvent(event: any): Partial<CostInfo> | null {
  if (!event || typeof event !== "object") return null;

  const info: Partial<CostInfo> = {};
  let found = false;

  if (typeof event.total_cost === "number") {
    info.totalCost = event.total_cost;
    found = true;
  }

  if (typeof event.input_tokens === "number") {
    info.inputTokens = event.input_tokens;
    found = true;
  }

  if (typeof event.output_tokens === "number") {
    info.outputTokens = event.output_tokens;
    found = true;
  }

  if (typeof event.cache_read_tokens === "number") {
    info.cacheReadTokens = event.cache_read_tokens;
    found = true;
  }

  if (typeof event.cache_write_tokens === "number") {
    info.cacheWriteTokens = event.cache_write_tokens;
    found = true;
  }

  // Also check nested usage object (common in API responses)
  if (event.usage && typeof event.usage === "object") {
    const u = event.usage;

    if (typeof u.input_tokens === "number") {
      info.inputTokens = u.input_tokens;
      found = true;
    }

    if (typeof u.output_tokens === "number") {
      info.outputTokens = u.output_tokens;
      found = true;
    }

    if (typeof u.cache_read_input_tokens === "number") {
      info.cacheReadTokens = u.cache_read_input_tokens;
      found = true;
    }

    if (typeof u.cache_creation_input_tokens === "number") {
      info.cacheWriteTokens = u.cache_creation_input_tokens;
      found = true;
    }
  }

  return found ? info : null;
}

/** Format a token count as a human-readable string (e.g. 12345 -> "12.3k"). */
function formatTokenCount(tokens: number): string {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(1)}M`;
  }
  if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(1)}k`;
  }
  return String(tokens);
}

/** Format cost info as "$0.05 | 12.3k tokens". */
export function formatCost(cost: CostInfo): string {
  const dollars = `$${cost.totalCost.toFixed(2)}`;
  const totalTokens =
    cost.inputTokens +
    cost.outputTokens +
    cost.cacheReadTokens +
    cost.cacheWriteTokens;
  return `${dollars} | ${formatTokenCount(totalTokens)} tokens`;
}
