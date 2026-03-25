import React, { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { Banner, Header } from "./Banner.js";
import { DecisionModal } from "./DecisionModal.js";
import { DecisionSummary } from "./DecisionSummary.js";
import { DashboardOverlay } from "./DashboardOverlay.js";
import { AgentManager } from "../agents/manager.js";
import { Agent, type AgentState } from "../agents/agent.js";
import type { LLMProvider } from "../providers/types.js";
import type { ClaudeCodeProvider } from "../providers/claude-code.js";
import { statusToMood, type MascotMood } from "./Mascot.js";

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

  // On mount: try to resume existing session, or start new task
  useEffect(() => {
    const resumed = Agent.loadSession(provider, (state) => {
      setAgents((prev) => {
        const idx = prev.findIndex((a) => a.id === state.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = { ...state };
          return next;
        }
        return [...prev, { ...state }];
      });
    });

    if (resumed) {
      setSelectedAgent(resumed.state.id);
      setAgents([{ ...resumed.state }]);
      if (
        resumed.state.status !== "asking" &&
        resumed.state.status !== "done"
      ) {
        resumed.start();
      }
      return;
    }

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

  // Track output lines, but suppress raw decision decomposition output
  useEffect(() => {
    if (!current?.currentOutput) return;
    // Don't show raw output while decomposing (it contains the JSON block)
    if (current.phase === "decomposing" && current.status === "thinking") return;
    // Don't show output that contains the defer-decisions JSON block
    const output = current.currentOutput;
    if (output.includes("```defer-decisions")) return;
    setOutputLines(output.split("\n"));
  }, [current?.currentOutput, current?.phase, current?.status]);

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

  const handleDecisionAsk = useCallback(
    (decisionId: string, question: string) => {
      if (!current) return;
      const agent = manager.get(current.id);
      if (!agent) return;
      const d = agent.state.decisions.find((d) => d.id === decisionId);
      if (!d) return;
      agent.sendUserMessage(
        `Question about ${decisionId} ("${d.question}"): ${question}`
      );
    },
    [current, manager]
  );

  const handleDecisionRevise = useCallback(
    (decisionId: string, newAnswer: string) => {
      if (!current) return;
      const agent = manager.get(current.id);
      if (!agent) return;
      agent.revisitDecision(decisionId, newAnswer);
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

    // Tab: cycle through views
    if (key.tab) {
      const viewCycle = ["stream", "decisions", "git"];
      const currentView = view === "banner" ? "stream" : view;
      const idx = viewCycle.indexOf(currentView);
      const next = viewCycle[(idx + 1) % viewCycle.length];
      // Skip decisions tab if no decisions exist
      if (next === "decisions" && (!current || current.decisions.length === 0)) {
        setView("git");
      } else {
        setView(next);
      }
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
        onAsk={handleDecisionAsk}
        onRevise={handleDecisionRevise}
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

  const mood: MascotMood = current
    ? statusToMood(current.status, current.phase)
    : "idle";

  const tabs = [
    { key: "stream", label: "Chat", icon: ">" },
    { key: "decisions", label: "Decide", icon: "◇" },
    { key: "git", label: "Git", icon: "±" },
  ];

  const activeTabKey =
    view === "banner" ? "stream" : view === "dashboard" ? "stream" : view;

  // Main layout: side panel + content
  return (
    <Box flexDirection="row" height={rows}>
      {/* Side panel */}
      <Box
        flexDirection="column"
        width={6}
        paddingTop={1}
      >
        {tabs.map((tab) => {
          const isActive = activeTabKey === tab.key;
          return (
            <Box key={tab.key} paddingX={1}>
              <Text
                color={isActive ? "cyan" : "gray"}
                bold={isActive}
                dimColor={!isActive}
              >
                {isActive ? "▸" : " "} {tab.icon}
              </Text>
            </Box>
          );
        })}
      </Box>

      {/* Main content area */}
      <Box flexDirection="column" flexGrow={1}>
        {/* Content */}
        <Box flexDirection="column" flexGrow={1} paddingX={1}>
          {view === "banner" && !current ? (
            <Banner model={model} cwd={process.cwd()} mood={mood} />
          ) : (
            <Header model={model} mood={mood} />
          )}

          {current?.status === "thinking" && outputLines.length === 0 && (
            <Box marginTop={1} paddingX={1}>
              <Text color="cyan">Decomposing task...</Text>
            </Box>
          )}

          {view === "git" ? (
            <GitView />
          ) : (
            visible.map((line, i) => (
              <Text key={i} wrap="wrap">
                {line}
              </Text>
            ))
          )}
        </Box>

        {/* Status bar */}
        <Box paddingX={1}>
          {current ? (
            <>
              <Text color={statusColor} dimColor>
                {current.status}
              </Text>
              {current.decisions.length > 0 && (
                <>
                  <Text color="gray" dimColor>
                    {" | "}
                    {current.decisions.length - pendingCount}/
                    {current.decisions.length} decisions
                  </Text>
                </>
              )}
              {pendingCount > 0 && (
                <Text color="yellow" dimColor>
                  {" "}({pendingCount} pending)
                </Text>
              )}
            </>
          ) : (
            <Text color="gray" dimColor>
              {model}
            </Text>
          )}
          <Box flexGrow={1} />
          <Text color="gray" dimColor>
            tab:switch  /help
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
    </Box>
  );
}

/** Inline git info view */
function GitView() {
  const [info, setInfo] = React.useState<{
    branch: string;
    commits: string[];
    dirty: string[];
  } | null>(null);

  React.useEffect(() => {
    try {
      const { execSync } = require("node:child_process");
      execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });
      const branch = execSync("git branch --show-current", {
        encoding: "utf-8",
      }).trim();
      let commits: string[] = [];
      try {
        commits = execSync("git log --oneline -10", { encoding: "utf-8" })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {}
      let dirty: string[] = [];
      try {
        dirty = execSync("git status --short", { encoding: "utf-8" })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {}
      setInfo({ branch, commits, dirty });
    } catch {
      setInfo(null);
    }
  }, []);

  if (!info) {
    return (
      <Box paddingX={1} marginTop={1}>
        <Text color="gray">Not a git repository.</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={1} marginTop={1}>
      <Box>
        <Text color="cyan" bold>
          {info.branch}
        </Text>
      </Box>
      {info.dirty.length > 0 && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="yellow" dimColor>
            {info.dirty.length} uncommitted
          </Text>
          {info.dirty.slice(0, 8).map((f, i) => (
            <Text key={i} color="gray" dimColor>
              {"  "}{f}
            </Text>
          ))}
        </Box>
      )}
      {info.commits.length > 0 && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray" dimColor>
            Recent commits
          </Text>
          {info.commits.map((c, i) => (
            <Text key={i} color="gray" dimColor>
              {"  "}{c}
            </Text>
          ))}
        </Box>
      )}
    </Box>
  );
}
