import React, { useState, useEffect } from "react";
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
  const current =
    agent.pendingIndex >= 0
      ? agent.decisions[agent.pendingIndex]
      : pending[0] || null;

  const allDone = pending.length === 0;
  const totalCount = agent.decisions.length;
  const answeredCount = answered.length;
  const currentNum = answeredCount + 1;

  // Reset selection when moving to next decision
  useEffect(() => {
    setSelectedOption(0);
    setTextMode(false);
    setTextValue("");
  }, [agent.pendingIndex]);

  // Auto-close when all done
  useEffect(() => {
    if (allDone && totalCount > 0) {
      const timer = setTimeout(onDone, 2000);
      return () => clearTimeout(timer);
    }
  }, [allDone, totalCount, onDone]);

  useInput((input, key) => {
    // All done state - any key closes
    if (allDone) {
      onDone();
      return;
    }

    if (!current) return;

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
        Math.min(i + 1, (current.options.length || 1) - 1)
      );
    }
    if (input === "k" || key.upArrow) {
      setSelectedOption((i) => Math.max(i - 1, 0));
    }
    if (key.return && current.options[selectedOption]) {
      onAnswer(current.options[selectedOption].key);
    }
    if (input === "t") {
      setTextMode(true);
    }
    if (key.escape) {
      onDone();
    }
  });

  // All done view
  if (allDone) {
    return (
      <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
        <Text color="green" bold>
          All {totalCount} decisions answered.
        </Text>
        <Box marginTop={1} flexDirection="column">
          {agent.decisions.map((d) => (
            <Box key={d.id}>
              <Text color={d.delegated ? "magenta" : "green"}>
                {d.delegated ? "◆" : "✓"}{" "}
              </Text>
              <Text color="gray">{d.id} </Text>
              <Text>{d.question} </Text>
              <Text color="gray">
                → {d.delegated ? `delegated: ${d.answer}` : d.answer}
              </Text>
            </Box>
          ))}
        </Box>
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          Proceeding with execution...
        </Text>
      </Box>
    );
  }

  // No current decision (shouldn't happen, but safe)
  if (!current) {
    return (
      <Box flexDirection="column" height={rows} padding={2}>
        <Text color="gray">Waiting for decisions...</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
      {/* Header bar */}
      <Box marginBottom={1}>
        <Text color="cyan" bold>
          {currentNum}/{totalCount}
        </Text>
        <Text color="gray">  </Text>
        <Text color="gray">{current.category}</Text>
        <Text color="gray">  </Text>
        <Text color="cyan" dimColor>
          {current.id}
        </Text>
      </Box>

      {/* Question */}
      <Box marginBottom={1}>
        <Text bold wrap="wrap">
          {current.question}
        </Text>
      </Box>

      {/* Context */}
      {current.context ? (
        <Box marginBottom={1}>
          <Text color="gray" italic wrap="wrap">
            {current.context}
          </Text>
        </Box>
      ) : null}

      {/* Options */}
      {!textMode && current.options.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          {current.options.map((opt, i) => {
            const isSelected = i === selectedOption;
            const isChooseForMe = opt.label
              .toLowerCase()
              .includes("choose for me");
            return (
              <Box key={opt.key} paddingLeft={1}>
                <Text color={isSelected ? "cyan" : "gray"}>
                  {isSelected ? " >" : "  "}{" "}
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
      ) : null}

      {/* Text input */}
      {textMode ? (
        <Box marginBottom={1} paddingLeft={2}>
          <Text color="yellow">{">"} </Text>
          <Text>{textValue}</Text>
          <Text color="gray">|</Text>
        </Box>
      ) : null}

      {/* Previously answered (compact) */}
      {answered.length > 0 ? (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray" dimColor>
            Answered:
          </Text>
          {answered.slice(-4).map((d) => (
            <Box key={d.id} paddingLeft={2}>
              <Text color={d.delegated ? "magenta" : "green"}>
                {d.delegated ? "◆" : "✓"}{" "}
              </Text>
              <Text color="gray">
                {d.question} → {d.answer}
              </Text>
            </Box>
          ))}
          {answered.length > 4 ? (
            <Box paddingLeft={2}>
              <Text color="gray" dimColor>
                ...and {answered.length - 4} more
              </Text>
            </Box>
          ) : null}
        </Box>
      ) : null}

      {/* Footer */}
      <Box flexGrow={1} />
      <Box>
        <Text color="gray" dimColor>
          {textMode
            ? "enter:submit  esc:back"
            : "↑/↓:navigate  enter:select  t:type custom  esc:close"}
        </Text>
      </Box>
    </Box>
  );
}
