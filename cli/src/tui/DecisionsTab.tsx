import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { AgentState } from "../agents/agent.js";

interface Props {
  agent: AgentState | undefined;
}

export function DecisionsTab({ agent }: Props) {
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [showOutput, setShowOutput] = useState(false);

  useInput((input, key) => {
    if (!agent) return;

    const max = agent.decisions.length - 1;
    if (input === "j" || key.downArrow) {
      setSelectedIdx((i) => Math.min(i + 1, max));
    }
    if (input === "k" || key.upArrow) {
      setSelectedIdx((i) => Math.max(i - 1, 0));
    }
    if (input === "o") {
      setShowOutput((v) => !v);
    }
  });

  if (!agent) {
    return (
      <Box padding={1}>
        <Text color="gray">No agent running.</Text>
      </Box>
    );
  }

  if (agent.decisions.length === 0 && !agent.currentOutput) {
    return (
      <Box padding={1}>
        <Text color="gray">Decomposing task...</Text>
      </Box>
    );
  }

  if (showOutput) {
    return (
      <Box padding={1} flexDirection="column">
        <Text color="gray" dimColor>
          [o: back to decisions]
        </Text>
        <Box marginTop={1}>
          <Text wrap="wrap">
            {agent.currentOutput || "(no output yet)"}
          </Text>
        </Box>
      </Box>
    );
  }

  // Group by category
  const categories = new Map<string, typeof agent.decisions>();
  for (const d of agent.decisions) {
    const cat = d.category || "General";
    if (!categories.has(cat)) categories.set(cat, []);
    categories.get(cat)!.push(d);
  }

  let globalIdx = 0;

  return (
    <Box padding={1} flexDirection="column">
      <Box marginBottom={1}>
        <Text color="gray" dimColor>
          j/k:navigate o:output i:respond
        </Text>
      </Box>

      {Array.from(categories.entries()).map(([category, decisions]) => (
        <Box key={category} flexDirection="column" marginBottom={1}>
          <Text color="cyan" bold>
            {category}
          </Text>
          {decisions.map((d) => {
            const idx = globalIdx++;
            const isSelected = idx === selectedIdx;
            const isDelegated = d.answer.startsWith("DELEGATED");
            const isPending = d.answer === "(pending)";

            return (
              <Box key={d.id} paddingLeft={1}>
                <Text color={isSelected ? "cyan" : "white"}>
                  {isSelected ? "> " : "  "}
                </Text>
                <Text color="gray">{d.id} </Text>
                <Text>{d.question} </Text>
                <Text
                  color={
                    isPending
                      ? "yellow"
                      : isDelegated
                        ? "magenta"
                        : "green"
                  }
                >
                  {d.answer}
                </Text>
              </Box>
            );
          })}
        </Box>
      ))}

      {agent.status === "asking" && (
        <Box marginTop={1}>
          <Text color="yellow">
            Waiting for your answers. Press i to respond.
          </Text>
        </Box>
      )}

      {agent.status === "thinking" && (
        <Box marginTop={1}>
          <Text color="cyan">Thinking...</Text>
        </Box>
      )}
    </Box>
  );
}
