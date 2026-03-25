import React, { useState, useEffect } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import type { AgentState } from "../agents/agent.js";
import { Header } from "./Banner.js";

type Mode = "browse" | "answer" | "change" | "ask" | "text";

interface Props {
  agent: AgentState;
  onAnswer: (value: string) => void;
  onDone: () => void;
  onAsk: (decisionId: string, question: string) => void;
  onRevise: (decisionId: string, newAnswer: string) => void;
  rows: number;
}

/** Truncate text to fit width, adding ... if needed */
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
  rows,
}: Props) {
  const { stdout } = useStdout();
  const cols = stdout?.columns || 80;

  const [cursorIdx, setCursorIdx] = useState(0);
  const [selectedOption, setSelectedOption] = useState(0);
  const [mode, setMode] = useState<Mode>("browse");
  const [textValue, setTextValue] = useState("");
  const [aiResponse, setAiResponse] = useState("");

  const decisions = agent.decisions;
  const pending = decisions.filter((d) => d.answer === null);
  const allDone = pending.length === 0 && decisions.length > 0;
  const current = decisions[cursorIdx] || null;
  const isPending = current?.answer === null;

  const firstPendingIdx = decisions.findIndex((d) => d.answer === null);

  useEffect(() => {
    if (firstPendingIdx >= 0) setCursorIdx(firstPendingIdx);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (agent.pendingIndex >= 0 && mode === "browse") {
      setCursorIdx(agent.pendingIndex);
    }
  }, [agent.pendingIndex]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    setSelectedOption(0);
  }, [cursorIdx]);

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
    if (key.escape) {
      onDone();
      return;
    }
    if (input === "j" || key.downArrow) {
      setCursorIdx((i) => Math.min(i + 1, decisions.length - 1));
    }
    if (input === "k" || key.upArrow) {
      setCursorIdx((i) => Math.max(i - 1, 0));
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

  // All done
  if (allDone) {
    return (
      <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
        <Header model="" />
        <Box marginTop={1} flexDirection="column">
          <Text color="green" bold>
            All {decisions.length} decisions answered.
          </Text>
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
      <Box flexDirection="column" height={rows} padding={2}>
        <Text color="gray">Waiting for decisions...</Text>
      </Box>
    );
  }

  // Group categories
  const categories = new Map<
    string,
    { d: (typeof decisions)[0]; idx: number }[]
  >();
  decisions.forEach((d, i) => {
    const cat = d.category || "General";
    if (!categories.has(cat)) categories.set(cat, []);
    categories.get(cat)!.push({ d, idx: i });
  });

  const answeredCount = decisions.filter((d) => d.answer !== null).length;
  const treeWidth = Math.floor(cols * 0.45);
  const maxQuestion = treeWidth - 12; // icon(3) + id(~8) + padding

  return (
    <Box flexDirection="column" height={rows}>
      {/* Header */}
      <Box paddingX={2} paddingTop={1}>
        <Text color="cyan" bold>
          Decisions
        </Text>
        <Text color="gray" dimColor>
          {"  "}
          {answeredCount}/{decisions.length} answered
          {pending.length > 0 ? `, ${pending.length} pending` : ""}
        </Text>
      </Box>

      {/* Main: tree + detail */}
      <Box flexGrow={1} paddingX={1}>
        {/* Left: compact decision tree */}
        <Box flexDirection="column" width={treeWidth} overflow="hidden">
          {Array.from(categories.entries()).map(([cat, items]) => (
            <Box key={cat} flexDirection="column">
              <Box paddingLeft={1}>
                <Text color="cyan" dimColor>
                  {cat}
                </Text>
              </Box>
              {items.map(({ d, idx }) => {
                const isCursor = idx === cursorIdx;
                const isAnswered = d.answer !== null;
                const icon = isAnswered
                  ? d.delegated
                    ? "◆"
                    : "✓"
                  : "○";
                const iconColor = isAnswered
                  ? d.delegated
                    ? "magenta"
                    : "green"
                  : "yellow";

                return (
                  <Box key={d.id} paddingLeft={2}>
                    <Text color={isCursor ? "cyan" : "gray"}>
                      {isCursor ? ">" : " "}
                    </Text>
                    <Text color={iconColor}>{icon}</Text>
                    <Text color="gray" dimColor>
                      {" "}
                      {d.id}
                    </Text>
                    <Text color={isCursor ? "white" : "gray"}>
                      {" "}
                      {truncate(d.question, maxQuestion)}
                    </Text>
                  </Box>
                );
              })}
            </Box>
          ))}
        </Box>

        {/* Right: detail panel */}
        <Box
          flexDirection="column"
          flexGrow={1}
          paddingLeft={1}
          borderStyle="single"
          borderColor="gray"
          borderLeft
          borderRight={false}
          borderTop={false}
          borderBottom={false}
          paddingX={1}
        >
          <Text color="cyan" dimColor>
            {current.id}
          </Text>
          <Box marginTop={1}>
            <Text bold wrap="wrap">
              {current.question}
            </Text>
          </Box>

          {current.context ? (
            <Box marginTop={1}>
              <Text color="gray" italic wrap="wrap">
                {current.context}
              </Text>
            </Box>
          ) : null}

          {/* Answer status */}
          <Box marginTop={1}>
            {isPending ? (
              <Text color="yellow">○ pending</Text>
            ) : (
              <Text color={current.delegated ? "magenta" : "green"}>
                {current.delegated ? "◆ delegated: " : "✓ "}
                {current.answer}
              </Text>
            )}
          </Box>

          {/* Answer mode: options */}
          {mode === "answer" && current.options.length > 0 ? (
            <Box flexDirection="column" marginTop={1}>
              {current.options.map((opt, i) => {
                const isSel = i === selectedOption;
                const isCfm = opt.label
                  .toLowerCase()
                  .includes("choose for me");
                return (
                  <Box key={opt.key}>
                    <Text color={isSel ? "cyan" : "gray"}>
                      {isSel ? " > " : "   "}
                    </Text>
                    <Text
                      color={
                        isSel ? "cyan" : isCfm ? "magenta" : "white"
                      }
                      bold={isSel}
                    >
                      {opt.key}) {opt.label}
                    </Text>
                  </Box>
                );
              })}
            </Box>
          ) : null}

          {/* Browse mode: show options as reference */}
          {mode === "browse" && current.options.length > 0 ? (
            <Box flexDirection="column" marginTop={1}>
              {current.options.map((o) => (
                <Text key={o.key} color="gray" dimColor>
                  {"  "}
                  {o.key}) {o.label}
                </Text>
              ))}
            </Box>
          ) : null}

          {/* Text input */}
          {(mode === "text" || mode === "change" || mode === "ask") ? (
            <Box marginTop={1} flexDirection="column">
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
            <Box marginTop={1} flexDirection="column">
              <Text wrap="wrap" color="gray">
                {aiResponse.length > 400
                  ? aiResponse.slice(-400)
                  : aiResponse}
              </Text>
            </Box>
          ) : null}
        </Box>
      </Box>

      {/* Footer */}
      <Box paddingX={2}>
        <Text color="gray" dimColor>
          {mode === "browse"
            ? `↑↓:navigate${isPending ? "  enter:answer" : "  c:change"}  a:ask  esc:back`
            : mode === "answer"
              ? "↑↓:pick  enter:confirm  t:custom  esc:back"
              : "enter:submit  esc:cancel"}
        </Text>
      </Box>
    </Box>
  );
}
