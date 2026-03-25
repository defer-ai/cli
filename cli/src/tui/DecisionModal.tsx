import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import type { AgentState } from "../agents/agent.js";

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

  // Track the first pending for auto-focus
  const firstPendingIdx = decisions.findIndex((d) => d.answer === null);

  // Auto-focus first pending on mount
  useEffect(() => {
    if (firstPendingIdx >= 0) {
      setCursorIdx(firstPendingIdx);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // When agent moves to next pending, follow it
  useEffect(() => {
    if (agent.pendingIndex >= 0 && mode === "browse") {
      setCursorIdx(agent.pendingIndex);
    }
  }, [agent.pendingIndex]); // eslint-disable-line react-hooks/exhaustive-deps

  // Reset option selection when cursor moves
  useEffect(() => {
    setSelectedOption(0);
  }, [cursorIdx]);

  // Auto-close when all done
  useEffect(() => {
    if (allDone) {
      const timer = setTimeout(onDone, 2500);
      return () => clearTimeout(timer);
    }
  }, [allDone, onDone]);

  // Show AI response when output changes (from an ask/revise action)
  useEffect(() => {
    if (agent.currentOutput && (mode === "ask" || mode === "change")) {
      setAiResponse(agent.currentOutput);
    }
  }, [agent.currentOutput, mode]);

  useInput((input, key) => {
    // Text input modes
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

    // Answer mode: picking from options
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

    // Navigation
    if (input === "j" || key.downArrow) {
      setCursorIdx((i) => Math.min(i + 1, decisions.length - 1));
    }
    if (input === "k" || key.upArrow) {
      setCursorIdx((i) => Math.max(i - 1, 0));
    }

    // Enter: answer if pending, or open answer mode
    if (key.return && current) {
      if (isPending && current.options.length > 0) {
        setMode("answer");
        setSelectedOption(0);
      } else if (isPending) {
        setMode("text");
      }
    }

    // c: change an existing answer
    if (input === "c" && current && !isPending) {
      setMode("change");
      setTextValue("");
    }

    // a: ask a question about this decision
    if (input === "a" && current) {
      setMode("ask");
      setTextValue("");
      setAiResponse("");
    }
  });

  // All done summary
  if (allDone) {
    return (
      <Box flexDirection="column" height={rows} paddingX={2} paddingY={1}>
        <Text color="green" bold>
          All {decisions.length} decisions answered.
        </Text>
        <Box marginTop={1} flexDirection="column">
          {decisions.map((d) => (
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

  if (!current) {
    return (
      <Box flexDirection="column" height={rows} padding={2}>
        <Text color="gray">Waiting for decisions...</Text>
      </Box>
    );
  }

  // Group for left panel
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
  const pendingCount = pending.length;

  return (
    <Box flexDirection="column" height={rows}>
      {/* Header */}
      <Box paddingX={2} paddingY={1}>
        <Text color="cyan" bold>
          Decisions
        </Text>
        <Text color="gray" dimColor>
          {"  "}
          {answeredCount}/{decisions.length} answered
          {pendingCount > 0 ? `, ${pendingCount} pending` : ""}
        </Text>
      </Box>

      {/* Main content: list + detail */}
      <Box flexGrow={1} paddingX={2}>
        {/* Left: decision tree */}
        <Box flexDirection="column" width="50%">
          {Array.from(categories.entries()).map(([cat, items]) => (
            <Box key={cat} flexDirection="column" marginBottom={1}>
              <Text color="cyan" dimColor>
                {cat}
              </Text>
              {items.map(({ d, idx }) => {
                const isCursor = idx === cursorIdx;
                const isAnswered = d.answer !== null;
                return (
                  <Box key={d.id} paddingLeft={1}>
                    <Text color={isCursor ? "cyan" : "gray"}>
                      {isCursor ? ">" : " "}{" "}
                    </Text>
                    <Text color={isAnswered ? "green" : "yellow"}>
                      {isAnswered
                        ? d.delegated
                          ? "◆"
                          : "✓"
                        : "○"}{" "}
                    </Text>
                    <Text
                      color={isCursor ? "white" : "gray"}
                      bold={isCursor}
                    >
                      {d.id}
                    </Text>
                    <Text color="gray"> {d.question}</Text>
                  </Box>
                );
              })}
            </Box>
          ))}
        </Box>

        {/* Right: detail + interaction */}
        <Box
          flexDirection="column"
          width="50%"
          paddingLeft={2}
          borderStyle="single"
          borderColor="gray"
          borderLeft
          borderRight={false}
          borderTop={false}
          borderBottom={false}
          paddingX={1}
        >
          <Text bold wrap="wrap">
            {current.question}
          </Text>

          {current.context ? (
            <Text color="gray" italic wrap="wrap">
              {current.context}
            </Text>
          ) : null}

          {/* Current answer or pending */}
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

          {/* Answer mode: show options */}
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

          {/* Text input (for custom answer, change, or ask) */}
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
                <Text color="yellow">{"> "}</Text>
                <Text>{textValue}</Text>
                <Text color="gray">|</Text>
              </Box>
            </Box>
          ) : null}

          {/* AI response (from ask) */}
          {mode === "ask" && aiResponse ? (
            <Box marginTop={1} flexDirection="column">
              <Text color="gray" dimColor>
                Response:
              </Text>
              <Text wrap="wrap" color="gray">
                {aiResponse.length > 500
                  ? aiResponse.slice(-500)
                  : aiResponse}
              </Text>
            </Box>
          ) : null}

          {/* Options (shown in browse mode for context) */}
          {mode === "browse" && current.options.length > 0 ? (
            <Box flexDirection="column" marginTop={1}>
              <Text color="gray" dimColor>
                Options:
              </Text>
              {current.options.map((o) => (
                <Text key={o.key} color="gray">
                  {"  "}
                  {o.key}) {o.label}
                </Text>
              ))}
            </Box>
          ) : null}
        </Box>
      </Box>

      {/* Footer */}
      <Box paddingX={2} paddingY={1}>
        <Text color="gray" dimColor>
          {mode === "browse"
            ? `↑/↓:navigate${isPending ? "  enter:answer" : "  c:change"}  a:ask about  esc:close`
            : mode === "answer"
              ? "↑/↓:pick  enter:confirm  t:type custom  esc:back"
              : "enter:submit  esc:back"}
        </Text>
      </Box>
    </Box>
  );
}
