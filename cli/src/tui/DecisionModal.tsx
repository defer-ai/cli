import React, { useState, useEffect } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import type { AgentState } from "../agents/agent.js";
import { MiniMascot } from "./Mascot.js";

type Mode = "browse" | "answer" | "change" | "ask" | "text";

interface Props {
  agent: AgentState;
  onAnswer: (value: string) => void;
  onDone: () => void;
  onAsk: (decisionId: string, question: string) => void;
  onRevise: (decisionId: string, newAnswer: string) => void;
  rows: number;
}

export function DecisionModal({
  agent,
  onAnswer,
  onDone,
  onAsk,
  onRevise,
  rows,
}: Props) {
  const { stdout } = useStdout();
  const cols = stdout?.columns || 80;

  const [selectedOption, setSelectedOption] = useState(0);
  const [mode, setMode] = useState<Mode>("browse");
  const [textValue, setTextValue] = useState("");
  const [aiResponse, setAiResponse] = useState("");

  const decisions = agent.decisions;
  const pending = decisions.filter((d) => d.answer === null);
  const allDone = pending.length === 0 && decisions.length > 0;
  const currentIdx = agent.pendingIndex >= 0
    ? agent.pendingIndex
    : pending.length > 0
      ? decisions.indexOf(pending[0])
      : 0;
  const current = decisions[currentIdx] || null;
  const isPending = current?.answer === null;
  const answeredCount = decisions.filter((d) => d.answer !== null).length;

  useEffect(() => {
    setSelectedOption(0);
    setMode("browse");
    setTextValue("");
  }, [agent.pendingIndex]);

  useEffect(() => {
    if (allDone) {
      const timer = setTimeout(onDone, 2500);
      return () => clearTimeout(timer);
    }
  }, [allDone, onDone]);

  useEffect(() => {
    if (agent.currentOutput && mode === "ask") {
      setAiResponse(agent.currentOutput);
    }
  }, [agent.currentOutput, mode]);

  useInput((input, key) => {
    if (allDone) {
      onDone();
      return;
    }

    // Text modes
    if (mode === "text" || mode === "ask" || mode === "change") {
      if (key.escape) {
        setMode("browse");
        setTextValue("");
        setAiResponse("");
        return;
      }
      if (key.return && textValue.trim()) {
        if (mode === "ask" && current) {
          onAsk(current.id, textValue.trim());
          setAiResponse("Thinking...");
          setTextValue("");
        } else if (mode === "change" && current) {
          onRevise(current.id, textValue.trim());
          setMode("browse");
          setTextValue("");
          setAiResponse("");
        } else if (mode === "text") {
          onAnswer(textValue.trim());
          setMode("browse");
          setTextValue("");
        }
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

    // Answer mode
    if (mode === "answer" && current) {
      if (key.escape) {
        setMode("browse");
        return;
      }
      if (input === "j" || key.downArrow) {
        setSelectedOption((i) =>
          Math.min(i + 1, (current.options.length || 1) - 1)
        );
        return;
      }
      if (input === "k" || key.upArrow) {
        setSelectedOption((i) => Math.max(i - 1, 0));
        return;
      }
      if (key.return && current.options[selectedOption]) {
        onAnswer(current.options[selectedOption].key);
        setMode("browse");
        setSelectedOption(0);
        return;
      }
      if (input === "t") {
        setMode("text");
        return;
      }
      return;
    }

    // Browse mode
    if (key.escape || key.tab) {
      onDone();
      return;
    }
    if (key.return && current) {
      if (isPending && current.options.length > 0) {
        setMode("answer");
        setSelectedOption(0);
      } else if (isPending) {
        setMode("text");
      }
    }
    if (input === "c" && current && !isPending) {
      setMode("change");
      setTextValue("");
    }
    if (input === "a" && current) {
      setMode("ask");
      setTextValue("");
      setAiResponse("");
    }
  });

  // All done summary
  if (allDone) {
    return (
      <Box flexDirection="column" height={rows} paddingX={3} paddingY={1}>
        <Box>
          <MiniMascot mood="done" />
          <Text color="green" bold>
            {" "}
            All {decisions.length} decisions answered
          </Text>
        </Box>
        <Box marginTop={1} flexDirection="column">
          {decisions.map((d) => (
            <Box key={d.id}>
              <Text color={d.delegated ? "magenta" : "green"}>
                {d.delegated ? "◆" : "✓"}{" "}
              </Text>
              <Text color="gray">{d.id} </Text>
              <Text color="gray">
                {d.question} → {d.answer}
              </Text>
            </Box>
          ))}
        </Box>
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          Proceeding...
        </Text>
      </Box>
    );
  }

  if (!current) {
    return (
      <Box flexDirection="column" height={rows} padding={3}>
        <Text color="gray">Waiting for decisions...</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={rows} paddingX={3} paddingY={1}>
      {/* Progress bar */}
      <Box marginBottom={1}>
        <MiniMascot mood={isPending ? "asking" : "answering"} />
        <Text color="cyan" bold>
          {"  "}
          {answeredCount + (isPending ? 0 : 1)}/{decisions.length}
        </Text>
        <Text color="gray" dimColor>
          {"  "}
          {current.category}
        </Text>
        <Text color="gray" dimColor>
          {"  "}
          {current.id}
        </Text>
      </Box>

      {/* Question - full width */}
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

      {/* Current answer (if already answered) */}
      {!isPending ? (
        <Box marginBottom={1}>
          <Text color={current.delegated ? "magenta" : "green"}>
            {current.delegated ? "◆ delegated: " : "✓ "}
            {current.answer}
          </Text>
        </Box>
      ) : null}

      {/* Options (browse or answer mode) */}
      {(mode === "browse" || mode === "answer") &&
      current.options.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          {current.options.map((opt, i) => {
            const isSel = mode === "answer" && i === selectedOption;
            const isCfm = opt.label.toLowerCase().includes("choose for me");
            return (
              <Box key={opt.key} paddingLeft={1}>
                <Text color={isSel ? "cyan" : "gray"}>
                  {isSel ? " >" : "  "}{" "}
                </Text>
                <Text
                  color={
                    isSel
                      ? "cyan"
                      : mode === "answer"
                        ? isCfm
                          ? "magenta"
                          : "white"
                        : "gray"
                  }
                  bold={isSel}
                  dimColor={mode === "browse"}
                >
                  {opt.key}) {opt.label}
                </Text>
              </Box>
            );
          })}
        </Box>
      ) : null}

      {/* Text input */}
      {(mode === "text" || mode === "change" || mode === "ask") ? (
        <Box marginBottom={1} flexDirection="column">
          <Text color="gray" dimColor>
            {mode === "ask"
              ? "Ask about this decision:"
              : mode === "change"
                ? "New answer:"
                : "Custom answer:"}
          </Text>
          <Box>
            <Text color="yellow">{">"} </Text>
            <Text>{textValue}</Text>
            <Text color="gray">|</Text>
          </Box>
        </Box>
      ) : null}

      {/* AI response from ask */}
      {mode === "ask" && aiResponse ? (
        <Box marginBottom={1}>
          <Text wrap="wrap" color="gray">
            {aiResponse.length > 500
              ? aiResponse.slice(-500)
              : aiResponse}
          </Text>
        </Box>
      ) : null}

      {/* Recently answered (compact) */}
      {answeredCount > 0 && isPending ? (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray" dimColor>
            Recent:
          </Text>
          {decisions
            .filter((d) => d.answer !== null)
            .slice(-3)
            .map((d) => (
              <Box key={d.id} paddingLeft={1}>
                <Text color={d.delegated ? "magenta" : "green"}>
                  {d.delegated ? "◆" : "✓"}
                </Text>
                <Text color="gray" dimColor>
                  {" "}
                  {d.id} {d.question} → {d.answer}
                </Text>
              </Box>
            ))}
        </Box>
      ) : null}

      {/* Footer */}
      <Box flexGrow={1} />
      <Box>
        <Text color="gray" dimColor>
          {mode === "browse"
            ? isPending
              ? "enter:answer  a:ask about  t:type custom  tab:back"
              : "c:change  a:ask about  tab:back"
            : mode === "answer"
              ? "↑↓:pick  enter:confirm  t:custom  esc:back"
              : "enter:submit  esc:cancel"}
        </Text>
      </Box>
    </Box>
  );
}
