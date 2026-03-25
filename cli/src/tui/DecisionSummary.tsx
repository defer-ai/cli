import React from "react";
import { Box, Text } from "ink";
import type { Decision } from "../decisions.js";

interface Props {
  decisions: Decision[];
}

export function DecisionSummary({ decisions }: Props) {
  if (decisions.length === 0) return null;

  // Group by category
  const categories = new Map<string, Decision[]>();
  for (const d of decisions) {
    const cat = d.category || "General";
    if (!categories.has(cat)) categories.set(cat, []);
    categories.get(cat)!.push(d);
  }

  return (
    <Box flexDirection="column" paddingX={1} marginY={1}>
      {Array.from(categories.entries()).map(([cat, items]) => (
        <Box key={cat} flexDirection="column">
          <Text color="cyan" dimColor>
            {cat}
          </Text>
          {items.map((d) => (
            <Box key={d.id} paddingLeft={2}>
              <Text color={d.answer === null ? "yellow" : d.delegated ? "magenta" : "green"}>
                {d.answer === null ? "○" : d.delegated ? "◆" : "✓"}{" "}
              </Text>
              <Text color="gray">{d.id} </Text>
              <Text>
                {d.answer === null
                  ? d.question
                  : d.delegated
                    ? `${d.question} → delegated`
                    : `${d.question} → ${d.answer}`}
              </Text>
            </Box>
          ))}
        </Box>
      ))}
    </Box>
  );
}
