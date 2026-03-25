import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { AgentState } from "../agents/agent.js";

interface Props {
  agent: AgentState;
  onAnswer: (value: string) => void;
  onDone: () => void;
  rows: number;
}

export function DecisionModal({ agent, onAnswer, onDone, rows }: Props) {
  const [selectedOption, setSelectedOption] = useState(0);
  const [textMode, setTextMode] = useState(false);
  const [textValue, setTextValue] = useState("");

  const pending = agent.decisions.filter((d) => d.answer === null);
  const answered = agent.decisions.filter((d) => d.answer !== null);
  const current = agent.pendingIndex >= 0
    ? agent.decisions[agent.pendingIndex]
    : pending[0];

  // All done
  if (!current || pending.length === 0) {
    return (
      <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
        <Box flexDirection="column" flexGrow={1}>
          <Text color="green" bold>
            All decisions answered.
          </Text>
          <Box marginTop={1} flexDirection="column">
            {agent.decisions.map((d) => (
              <Box key={d.id}>
                <Text color={d.delegated ? "magenta" : "green"}>
                  {d.delegated ? "◆" : "✓"}{" "}
                </Text>
                <Text color="gray">{d.id} </Text>
                <Text>
                  {d.delegated ? `delegated: ${d.answer}` : d.answer}
                </Text>
              </Box>
            ))}
          </Box>
        </Box>
        <Box>
          <Text color="gray" dimColor>
            Proceeding...
          </Text>
        </Box>
      </Box>
    );
  }

  const totalCount = agent.decisions.length;
  const answeredCount = answered.length;
  const currentNum = answeredCount + 1;

  useInput((input, key) => {
    if (textMode) {
      if (key.escape) {
        setTextMode(false);
        setTextValue("");
        return;
      }
      if (key.return && textValue.trim()) {
        onAnswer(textValue.trim());
        setTextValue("");
        setTextMode(false);
        setSelectedOption(0);
        return;
      }
      if (key.backspace || key.delete) {
        setTextValue((v) => v.slice(0, -1));
        return;
      }
      if (input && !key.ctrl && !key.meta) {
        setTextValue((v) => v + input);
      }
      return;
    }

    // Option navigation
    if (input === "j" || key.downArrow) {
      setSelectedOption((i) =>
        Math.min(i + 1, current.options.length - 1)
      );
    }
    if (input === "k" || key.upArrow) {
      setSelectedOption((i) => Math.max(i - 1, 0));
    }
    if (key.return && current.options[selectedOption]) {
      onAnswer(current.options[selectedOption].key);
      setSelectedOption(0);
    }
    if (input === "t") {
      setTextMode(true);
    }
  });

  return (
    <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text color="cyan" bold>
          {currentNum}/{totalCount}
        </Text>
        <Text color="gray"> | </Text>
        <Text color="gray">{current.category}</Text>
        <Text color="gray"> | </Text>
        <Text color="cyan">{current.id}</Text>
      </Box>

      {/* Question */}
      <Box marginBottom={1}>
        <Text bold wrap="wrap">
          {current.question}
        </Text>
      </Box>

      {/* Context */}
      {current.context && (
        <Box marginBottom={1}>
          <Text color="gray" italic wrap="wrap">
            {current.context}
          </Text>
        </Box>
      )}

      {/* Options */}
      {!textMode && current.options.length > 0 && (
        <Box flexDirection="column" marginBottom={1}>
          {current.options.map((opt, i) => {
            const isSelected = i === selectedOption;
            const isChooseForMe = opt.label
              .toLowerCase()
              .includes("choose for me");
            return (
              <Box key={opt.key}>
                <Text color={isSelected ? "cyan" : "gray"}>
                  {isSelected ? "  > " : "    "}
                </Text>
                <Text
                  color={
                    isSelected
                      ? "cyan"
                      : isChooseForMe
                        ? "magenta"
                        : "white"
                  }
                  bold={isSelected}
                >
                  {opt.key}) {opt.label}
                </Text>
              </Box>
            );
          })}
        </Box>
      )}

      {/* Text input mode */}
      {textMode && (
        <Box marginBottom={1}>
          <Text color="yellow">{"> "}</Text>
          <Text>{textValue}</Text>
          <Text color="gray">|</Text>
        </Box>
      )}

      {/* Previously answered (compact) */}
      {answered.length > 0 && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray" dimColor>
            Answered:
          </Text>
          {answered.slice(-5).map((d) => (
            <Box key={d.id} paddingLeft={2}>
              <Text color={d.delegated ? "magenta" : "green"}>
                {d.delegated ? "◆" : "✓"}{" "}
              </Text>
              <Text color="gray">
                {d.id} {d.question}{" "}
              </Text>
              <Text color="green">{d.answer}</Text>
            </Box>
          ))}
          {answered.length > 5 && (
            <Box paddingLeft={2}>
              <Text color="gray" dimColor>
                ...and {answered.length - 5} more
              </Text>
            </Box>
          )}
        </Box>
      )}

      {/* Footer */}
      <Box flexGrow={1} />
      <Box>
        <Text color="gray" dimColor>
          {textMode
            ? "enter:submit  esc:back to options"
            : "↑/↓:navigate  enter:select  t:type custom"}
        </Text>
      </Box>
    </Box>
  );
}
