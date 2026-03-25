import React, { useState, useEffect } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import type { AgentState } from "../agents/agent.js";
import { MiniMascot } from "./Mascot.js";

type Mode = "pick" | "text" | "ask" | "change";

interface Props {
  agent: AgentState;
  onAnswer: (value: string) => void;
  onDone: () => void;
  onAsk: (decisionId: string, question: string) => void;
  onRevise: (decisionId: string, newAnswer: string) => void;
  focusId?: string | null;
  rows: number;
}

function truncate(text: string, maxLen: number): string {
  if (text.length <= maxLen) return text;
  return text.slice(0, maxLen - 1) + "…";
}

export function DecisionModal({
  agent,
  onAnswer,
  onDone,
  onAsk,
  onRevise,
  focusId,
  rows,
}: Props) {
  const { stdout } = useStdout();
  const cols = stdout?.columns || 80;

  const [selectedOption, setSelectedOption] = useState(0);
  const [mode, setMode] = useState<Mode>("pick");
  const [textValue, setTextValue] = useState("");
  const [aiResponse, setAiResponse] = useState("");

  const decisions = agent.decisions;
  const pending = decisions.filter((d) => d.answer === null);
  const allDone = pending.length === 0 && decisions.length > 0;

  // Which decision are we looking at?
  // If focusId is set (from /revisit), focus that one
  // Otherwise focus the current pending
  const focusIdx = focusId
    ? decisions.findIndex((d) => d.id === focusId)
    : agent.pendingIndex >= 0
      ? agent.pendingIndex
      : pending.length > 0
        ? decisions.indexOf(pending[0])
        : 0;

  const current = decisions[focusIdx >= 0 ? focusIdx : 0] || null;
  const isPending = current?.answer === null;
  const answeredCount = decisions.filter((d) => d.answer !== null).length;

  // Determine initial mode: pick if pending with options, change if revisiting answered
  useEffect(() => {
    if (focusId) {
      const d = decisions.find((d) => d.id === focusId);
      if (d && d.answer !== null) {
        setMode("change");
        setTextValue("");
      } else {
        setMode("pick");
      }
    } else {
      setMode("pick");
    }
    setSelectedOption(0);
    setAiResponse("");
  }, [agent.pendingIndex, focusId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-close when all done (only if not revisiting)
  useEffect(() => {
    if (allDone && !focusId) {
      const timer = setTimeout(onDone, 2000);
      return () => clearTimeout(timer);
    }
  }, [allDone, focusId, onDone]);

  // AI response tracking
  useEffect(() => {
    if (agent.currentOutput && mode === "ask") {
      setAiResponse(agent.currentOutput);
    }
  }, [agent.currentOutput, mode]);

  useInput((input, key) => {
    // All done - any key closes
    if (allDone && !focusId) {
      onDone();
      return;
    }

    // Text input modes (text, ask, change)
    if (mode === "text" || mode === "ask" || mode === "change") {
      if (key.escape) {
        if (focusId && mode === "change") {
          onDone(); // exit revisit
          return;
        }
        setMode("pick");
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
          setTextValue("");
          onDone();
        } else if (mode === "text") {
          onAnswer(textValue.trim());
          setMode("pick");
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

    // Pick mode - immediately navigable
    if (key.escape) {
      onDone();
      return;
    }
    if (input === "j" || key.downArrow) {
      setSelectedOption((i) =>
        Math.min(i + 1, (current?.options.length || 1) - 1)
      );
      return;
    }
    if (input === "k" || key.upArrow) {
      setSelectedOption((i) => Math.max(i - 1, 0));
      return;
    }
    if (key.return && current?.options[selectedOption]) {
      onAnswer(current.options[selectedOption].key);
      setSelectedOption(0);
      return;
    }
    if (input === "t") {
      setMode("text");
      setTextValue("");
      return;
    }
    if (input === "a" && current) {
      setMode("ask");
      setTextValue("");
      setAiResponse("");
      return;
    }
    if (input === "c" && current && !isPending) {
      setMode("change");
      setTextValue("");
      return;
    }
  });

  // All done summary
  if (allDone && !focusId) {
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
              <Text color="gray">
                {d.id} {d.question} → {d.answer}
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
      {/* Progress */}
      <Box marginBottom={1}>
        <MiniMascot mood={isPending ? "asking" : "answering"} />
        <Text color="cyan" bold>
          {"  "}
          {focusId
            ? current.id
            : `${answeredCount + (isPending ? 0 : 1)}/${decisions.length}`}
        </Text>
        <Text color="gray" dimColor>
          {"  "}
          {current.category}
        </Text>
        {!focusId && (
          <Text color="gray" dimColor>
            {"  "}
            {current.id}
          </Text>
        )}
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

      {/* Existing answer (if revisiting) */}
      {!isPending && mode !== "change" ? (
        <Box marginBottom={1}>
          <Text color={current.delegated ? "magenta" : "green"}>
            {current.delegated ? "◆ delegated: " : "✓ "}
            {current.answer}
          </Text>
        </Box>
      ) : null}

      {/* Options - immediately active in pick mode */}
      {mode === "pick" && current.options.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          {current.options.map((opt, i) => {
            const isSel = i === selectedOption;
            const isCfm = opt.label.toLowerCase().includes("choose for me");
            return (
              <Box key={opt.key} paddingLeft={1}>
                <Text color={isSel ? "cyan" : "gray"}>
                  {isSel ? ">" : " "}{" "}
                </Text>
                <Text
                  color={isSel ? "cyan" : isCfm ? "magenta" : "white"}
                  bold={isSel}
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

      {/* AI response */}
      {mode === "ask" && aiResponse ? (
        <Box marginBottom={1}>
          <Text wrap="wrap" color="gray">
            {aiResponse.length > 500
              ? aiResponse.slice(-500)
              : aiResponse}
          </Text>
        </Box>
      ) : null}

      {/* Recently answered */}
      {!focusId && answeredCount > 0 && isPending ? (
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
                  {d.id} {truncate(d.question, 30)} → {truncate(d.answer || "", 20)}
                </Text>
              </Box>
            ))}
        </Box>
      ) : null}

      {/* Footer */}
      <Box flexGrow={1} />
      <Box>
        <Text color="gray" dimColor>
          {mode === "pick"
            ? isPending
              ? "↑↓:pick  enter:confirm  t:custom  a:ask  esc:back"
              : "c:change  a:ask  esc:back"
            : "enter:submit  esc:cancel"}
        </Text>
      </Box>
    </Box>
  );
}
