import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import type { AgentState } from "../agents/agent.js";

interface Props {
  agent: AgentState | undefined;
}

export function DecisionsTab({ agent }: Props) {
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [showOutput, setShowOutput] = useState(false);

  // Sync selectedIdx with the current pending decision
  useEffect(() => {
    if (agent && agent.pendingIndex >= 0) {
      setSelectedIdx(agent.pendingIndex);
    }
  }, [agent?.pendingIndex]);

  useInput((input, key) => {
    if (!agent) return;
    const max = Math.max(agent.decisions.length - 1, 0);
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

  if (agent.decisions.length === 0 && agent.status === "thinking") {
    return (
      <Box padding={1}>
        <Text color="cyan">Decomposing task...</Text>
      </Box>
    );
  }

  if (agent.decisions.length === 0) {
    return (
      <Box padding={1}>
        <Text color="gray">No decisions yet.</Text>
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
  const categories = new Map<string, { decision: (typeof agent.decisions)[0]; globalIdx: number }[]>();
  agent.decisions.forEach((d, i) => {
    const cat = d.category || "General";
    if (!categories.has(cat)) categories.set(cat, []);
    categories.get(cat)!.push({ decision: d, globalIdx: i });
  });

  const selected = agent.decisions[selectedIdx];

  return (
    <Box padding={1} flexDirection="row">
      {/* Decision list */}
      <Box flexDirection="column" width="55%">
        {Array.from(categories.entries()).map(([category, items]) => (
          <Box key={category} flexDirection="column" marginBottom={1}>
            <Text color="cyan" bold>
              {category}
            </Text>
            {items.map(({ decision: d, globalIdx: idx }) => {
              const isSelected = idx === selectedIdx;
              const isCurrent = idx === agent.pendingIndex;
              const isPending = d.answer === null;
              const isDelegated = d.delegated;

              return (
                <Box key={d.id} paddingLeft={1}>
                  <Text color={isSelected ? "cyan" : "gray"}>
                    {isCurrent ? ">" : isSelected ? ">" : " "}{" "}
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
                    bold={isCurrent}
                  >
                    {isPending
                      ? "pending"
                      : isDelegated
                        ? `delegated`
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
        {agent.status === "executing" && (
          <Box marginTop={1}>
            <Text color="blue">Executing...</Text>
          </Box>
        )}
        {agent.status === "done" && (
          <Box marginTop={1}>
            <Text color="green">Done.</Text>
          </Box>
        )}
      </Box>

      {/* Detail panel */}
      {selected && (
        <Box
          flexDirection="column"
          width="45%"
          borderStyle="single"
          borderColor="gray"
          paddingX={1}
          paddingY={0}
        >
          <Text color="cyan" bold>
            {selected.id}
          </Text>
          <Text color="gray">{selected.category}</Text>
          <Box marginTop={1}>
            <Text bold wrap="wrap">{selected.question}</Text>
          </Box>

          {selected.context ? (
            <Box marginTop={1}>
              <Text color="gray" italic wrap="wrap">
                {selected.context}
              </Text>
            </Box>
          ) : null}

          {selected.options.length > 0 && (
            <Box flexDirection="column" marginTop={1}>
              {selected.options.map((o) => (
                <Text key={o.key} color="white">
                  {o.key}) {o.label}
                </Text>
              ))}
            </Box>
          )}

          <Box marginTop={1}>
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
        </Box>
      )}
    </Box>
  );
}
