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

  return (
    <Box padding={1} flexDirection="column">
      <Box marginBottom={1}>
        <Text color="gray" dimColor>
          j/k:navigate enter:select
        </Text>
      </Box>

      {agents.map((agent, i) => {
        const isSelected = agent.id === selectedId;
        const isCursor = i === cursorIdx;

        return (
          <Box key={agent.id} paddingLeft={1}>
            <Text color={isCursor ? "cyan" : "white"}>
              {isCursor ? "> " : "  "}
            </Text>
            <Text color={isSelected ? "cyan" : "white"} bold={isSelected}>
              {agent.id}
            </Text>
            <Text> </Text>
            <Text color={(statusColors[agent.status] as any) || "white"}>
              [{agent.status}]
            </Text>
            <Text> </Text>
            <Text color="gray">
              {agent.task.length > 50
                ? agent.task.slice(0, 50) + "..."
                : agent.task}
            </Text>
            <Text> </Text>
            <Text color="gray" dimColor>
              {agent.decisions.length}d
            </Text>
          </Box>
        );
      })}
    </Box>
  );
}
