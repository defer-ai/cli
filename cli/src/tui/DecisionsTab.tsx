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
      <Box padding={1} flexDirection="column">
        <Text color="gray">Decomposing task...</Text>
      </Box>
    );
  }

  if (showOutput) {
    return (
      <Box padding={1} flexDirection="column">
        <Text color="gray" dimColor>
          press o to go back
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
  const selected = agent.decisions[selectedIdx];

  return (
    <Box padding={1} flexDirection="row">
      {/* Decision list */}
      <Box flexDirection="column" width="60%">
        <Box marginBottom={1}>
          <Text color="gray" dimColor>
            j/k:navigate o:raw output i:respond
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
              const isPending = d.answer === null;
              const isDelegated = d.delegated;

              return (
                <Box key={d.id} paddingLeft={1}>
                  <Text color={isSelected ? "cyan" : "gray"}>
                    {isSelected ? ">" : " "}{" "}
                  </Text>
                  <Text color="gray" dimColor>
                    {d.id}{" "}
                  </Text>
                  <Text color={isSelected ? "white" : "gray"}>
                    {d.question}{" "}
                  </Text>
                  <Text
                    color={
                      isPending
                        ? "yellow"
                        : isDelegated
                          ? "magenta"
                          : "green"
                    }
                  >
                    {isPending
                      ? "pending"
                      : isDelegated
                        ? `delegated: ${d.answer}`
                        : d.answer}
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

      {/* Detail panel */}
      {selected && (
        <Box
          flexDirection="column"
          width="40%"
          borderStyle="single"
          borderColor="gray"
          paddingX={1}
        >
          <Text color="cyan" bold>
            {selected.id}
          </Text>
          <Text color="gray" dimColor>
            {selected.category}
          </Text>
          <Box marginTop={1}>
            <Text bold>{selected.question}</Text>
          </Box>

          {selected.context ? (
            <Box marginTop={1}>
              <Text color="gray" italic>
                {selected.context}
              </Text>
            </Box>
          ) : null}

          {selected.options.length > 0 && (
            <Box flexDirection="column" marginTop={1}>
              <Text color="gray" dimColor>
                Options:
              </Text>
              {selected.options.map((o) => (
                <Box key={o.key} paddingLeft={1}>
                  <Text color="white">
                    {o.key}) {o.label}
                  </Text>
                </Box>
              ))}
            </Box>
          )}

          <Box marginTop={1}>
            <Text color="gray">Answer: </Text>
            <Text
              color={
                selected.answer === null
                  ? "yellow"
                  : selected.delegated
                    ? "magenta"
                    : "green"
              }
            >
              {selected.answer === null
                ? "pending"
                : selected.delegated
                  ? `delegated: ${selected.answer}`
                  : selected.answer}
            </Text>
          </Box>

          <Box marginTop={1}>
            <Text color="gray" dimColor>
              {selected.date}
            </Text>
          </Box>
        </Box>
      )}
    </Box>
  );
}
