import React from "react";
import { Box, Text, useInput } from "ink";
import type { AgentState } from "../agents/agent.js";

interface Props {
  agents: AgentState[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

const statusColors: Record<string, string> = {
  idle: "gray",
  thinking: "cyan",
  asking: "yellow",
  executing: "blue",
  done: "green",
  error: "red",
};

export function AgentsTab({ agents, selectedId, onSelect }: Props) {
  const [cursorIdx, setCursorIdx] = React.useState(0);

  useInput((input, key) => {
    if (input === "j" || key.downArrow) {
      setCursorIdx((i) => Math.min(i + 1, agents.length - 1));
    }
    if (input === "k" || key.upArrow) {
      setCursorIdx((i) => Math.max(i - 1, 0));
    }
    if (key.return && agents[cursorIdx]) {
      onSelect(agents[cursorIdx].id);
    }
  });

  if (agents.length === 0) {
    return (
      <Box padding={1}>
        <Text color="gray">No agents spawned.</Text>
      </Box>
    );
  }

  const selected = agents[cursorIdx];

  return (
    <Box padding={1} flexDirection="row">
      {/* Agent list */}
      <Box flexDirection="column" width="40%">
        <Box marginBottom={1}>
          <Text color="gray" dimColor>
            j/k:navigate enter:select
          </Text>
        </Box>

        {agents.map((agent, i) => {
          const isActive = agent.id === selectedId;
          const isCursor = i === cursorIdx;
          const answered = agent.decisions.filter(
            (d) => d.answer !== null
          ).length;
          const total = agent.decisions.length;
          const delegated = agent.decisions.filter((d) => d.delegated).length;

          return (
            <Box key={agent.id} flexDirection="column" marginBottom={1}>
              <Box>
                <Text color={isCursor ? "cyan" : "gray"}>
                  {isCursor ? "> " : "  "}
                </Text>
                <Text
                  color={isActive ? "cyan" : "white"}
                  bold={isActive}
                >
                  {agent.id}
                </Text>
                <Text> </Text>
                <Text
                  color={
                    (statusColors[agent.status] as any) || "white"
                  }
                >
                  {agent.status}
                </Text>
              </Box>
              <Box paddingLeft={4}>
                <Text color="gray">
                  {answered}/{total} answered
                  {delegated > 0 ? `, ${delegated} delegated` : ""}
                </Text>
              </Box>
            </Box>
          );
        })}
      </Box>

      {/* Agent detail */}
      {selected && (
        <Box
          flexDirection="column"
          width="60%"
          borderStyle="single"
          borderColor="gray"
          paddingX={1}
        >
          <Text color="cyan" bold>
            {selected.id}
          </Text>

          <Box marginTop={1}>
            <Text color="gray">Task: </Text>
            <Text wrap="wrap">{selected.task}</Text>
          </Box>

          <Box marginTop={1}>
            <Text color="gray">Status: </Text>
            <Text
              color={
                (statusColors[selected.status] as any) || "white"
              }
            >
              {selected.status}
            </Text>
            <Text color="gray"> | Phase: </Text>
            <Text>{selected.phase}</Text>
          </Box>

          <Box marginTop={1}>
            <Text color="gray">
              Decisions: {selected.decisions.length} total,{" "}
              {selected.decisions.filter((d) => d.answer !== null).length}{" "}
              answered,{" "}
              {selected.decisions.filter((d) => d.answer === null).length}{" "}
              pending
            </Text>
          </Box>

          {selected.currentOutput && (
            <Box marginTop={1} flexDirection="column">
              <Text color="gray" dimColor>
                Latest output:
              </Text>
              <Text wrap="wrap" color="gray">
                {selected.currentOutput.length > 300
                  ? selected.currentOutput.slice(-300) + "..."
                  : selected.currentOutput}
              </Text>
            </Box>
          )}

          {selected.error && (
            <Box marginTop={1}>
              <Text color="red">Error: {selected.error}</Text>
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
}
