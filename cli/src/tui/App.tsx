import React, { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { Banner } from "./Banner.js";
import { DecisionModal } from "./DecisionModal.js";
import { DecisionSummary } from "./DecisionSummary.js";
import { DashboardOverlay } from "./DashboardOverlay.js";
import { AgentManager } from "../agents/manager.js";
import { Agent, type AgentState } from "../agents/agent.js";
import type { LLMProvider } from "../providers/types.js";
import type { ClaudeCodeProvider } from "../providers/claude-code.js";

type View = string; // "banner" | "stream" | "decisions" | "dashboard"

interface AppProps {
  task?: string;
  provider: LLMProvider;
}

export function App({ task, provider }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const rows = stdout?.rows || 24;

  const [view, setView] = useState<View>(task ? "stream" : "banner");
  const [agents, setAgents] = useState<AgentState[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [model, setModel] = useState(
    (provider as ClaudeCodeProvider).model || "sonnet"
  );
  const [manager] = useState(
    () => new AgentManager(provider, (states) => setAgents([...states]))
  );
  const prevStatus = useRef<string>("");

  const current = agents.find((a) => a.id === selectedAgent) || agents[0];

  // Start task if provided as argument
  useEffect(() => {
    if (task) {
      startTask(task);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-switch to decision modal when agent starts asking
  useEffect(() => {
    if (!current) return;
    if (current.status === "asking" && prevStatus.current !== "asking") {
      setView("decisions");
    }
    if (
      current.status !== "asking" &&
      prevStatus.current === "asking" &&
      view === "decisions"
    ) {
      setView("stream");
    }
    prevStatus.current = current.status;
  }, [current?.status, view]);

  // Track output lines
  useEffect(() => {
    if (current?.currentOutput) {
      setOutputLines(current.currentOutput.split("\n"));
    }
  }, [current?.currentOutput]);

  const startTask = useCallback(
    (taskText: string) => {
      const agent = manager.spawn(taskText);
      setSelectedAgent(agent.state.id);
      setOutputLines([]);
      setView("stream");
      agent.start();
    },
    [manager]
  );

  const handleSlashCommand = useCallback(
    (cmd: string) => {
      const parts = cmd.slice(1).split(/\s+/);
      const command = parts[0].toLowerCase();

      switch (command) {
        case "help":
          setOutputLines((prev) => [
            ...prev,
            "",
            "  Commands:",
            "  /help              Show this help",
            "  /model <name>      Switch model (sonnet, opus, haiku)",
            "  /status            Show decision record",
            "  /decisions         Open decision view",
            "  /dashboard         Open dashboard overlay",
            "  /clear             Clear output",
            "  /quit              Exit",
            "",
          ]);
          break;
        case "model":
          if (parts[1]) {
            const m = parts[1].toLowerCase();
            (provider as ClaudeCodeProvider).setModel(m);
            setModel(m);
            setOutputLines((prev) => [
              ...prev,
              `  Model switched to ${m}`,
            ]);
          } else {
            setOutputLines((prev) => [
              ...prev,
              `  Current model: ${model}`,
              "  Usage: /model <sonnet|opus|haiku>",
            ]);
          }
          break;
        case "status":
        case "decisions":
          if (current && current.decisions.length > 0) {
            setView("decisions");
          } else {
            setOutputLines((prev) => [
              ...prev,
              "  No decisions yet.",
            ]);
          }
          break;
        case "dashboard":
          setView("dashboard");
          break;
        case "clear":
          setOutputLines([]);
          break;
        case "quit":
        case "exit":
          exit();
          break;
        default:
          setOutputLines((prev) => [
            ...prev,
            `  Unknown command: /${command}. Type /help for commands.`,
          ]);
      }
    },
    [provider, model, current, exit]
  );

  const handleSubmit = useCallback(() => {
    const value = inputValue.trim();
    setInputValue("");

    if (!value) return;

    if (value.startsWith("/")) {
      handleSlashCommand(value);
      return;
    }

    // If there's an active agent in asking/done state, send message
    if (current) {
      const agent = manager.get(current.id);
      if (agent) {
        setOutputLines((prev) => [...prev, "", `  > ${value}`, ""]);
        agent.sendUserMessage(value);
        setView("stream");
        return;
      }
    }

    // Otherwise start a new task
    startTask(value);
  }, [inputValue, handleSlashCommand, current, manager, startTask]);

  const handleDecisionAnswer = useCallback(
    (value: string) => {
      if (!current) return;
      const agent = manager.get(current.id);
      if (!agent) return;
      agent.sendUserMessage(value);
    },
    [current, manager]
  );

  useInput((input, key) => {
    // Decision modal handles its own input
    if (view === "decisions") return;
    // Dashboard handles its own input
    if (view === "dashboard") return;

    // Escape: close overlays
    if (key.escape) {
      if (view === "dashboard") {
        setView("stream");
        return;
      }
    }

    // In stream/banner view, all typing goes to input
    if (key.return) {
      handleSubmit();
      return;
    }

    if (key.backspace || key.delete) {
      setInputValue((v) => v.slice(0, -1));
      return;
    }

    // Ctrl+C to quit
    if (input === "c" && key.ctrl) {
      exit();
      return;
    }

    // Ctrl+D for dashboard
    if (input === "d" && key.ctrl) {
      setView(view === "dashboard" ? "stream" : "dashboard");
      return;
    }

    // Regular character input
    if (input && !key.ctrl && !key.meta && !key.tab) {
      setInputValue((v) => v + input);
    }
  });

  const pendingCount = current
    ? current.decisions.filter((d) => d.answer === null).length
    : 0;

  // Visible output (last N lines)
  const maxVisible = rows - 5; // banner line + input + status + padding
  const visible = outputLines.slice(-maxVisible);

  const statusColor =
    !current || current.status === "idle"
      ? "gray"
      : current.status === "thinking"
        ? "cyan"
        : current.status === "asking"
          ? "yellow"
          : current.status === "executing"
            ? "blue"
            : current.status === "done"
              ? "green"
              : "red";

  // Decision modal (full screen takeover)
  if (view === "decisions" && current) {
    return (
      <DecisionModal
        agent={current}
        onAnswer={handleDecisionAnswer}
        onDone={() => setView("stream")}
        rows={rows}
      />
    );
  }

  // Dashboard overlay
  if (view === "dashboard") {
    return (
      <DashboardOverlay
        agents={agents}
        selectedId={selectedAgent}
        onSelect={setSelectedAgent}
        onClose={() => setView("stream")}
        rows={rows}
      />
    );
  }

  // Main view: banner/stream + input prompt
  return (
    <Box flexDirection="column" height={rows}>
      {/* Content area */}
      <Box flexDirection="column" flexGrow={1} paddingX={1}>
        {view === "banner" && !current && (
          <Banner model={model} cwd={process.cwd()} />
        )}

        {current?.status === "thinking" && outputLines.length === 0 && (
          <Box marginTop={1} paddingX={1}>
            <Text color="cyan">Decomposing task...</Text>
          </Box>
        )}

        {visible.map((line, i) => (
          <Text key={i} wrap="wrap">
            {line}
          </Text>
        ))}
      </Box>

      {/* Status bar */}
      <Box paddingX={1}>
        {current ? (
          <>
            <Text color={statusColor} bold>
              {current.status}
            </Text>
            {current.decisions.length > 0 && (
              <>
                <Text color="gray"> | </Text>
                <Text color="gray">
                  {current.decisions.length - pendingCount}/
                  {current.decisions.length} decisions
                </Text>
              </>
            )}
            {pendingCount > 0 && (
              <Text color="yellow">
                {" "}
                ({pendingCount} pending)
              </Text>
            )}
          </>
        ) : (
          <Text color="gray">{model}</Text>
        )}
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          /help  ctrl+d:dashboard
        </Text>
      </Box>

      {/* Input prompt */}
      <Box paddingX={1}>
        <Text color="cyan" bold>
          {"defer > "}
        </Text>
        <Text>{inputValue}</Text>
        <Text color="gray">|</Text>
      </Box>
    </Box>
  );
}
