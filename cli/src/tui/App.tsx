import React, { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { DecisionModal } from "./DecisionModal.js";
import { DecisionSummary } from "./DecisionSummary.js";
import { DashboardOverlay } from "./DashboardOverlay.js";
import { AgentManager } from "../agents/manager.js";
import { Agent, type AgentState } from "../agents/agent.js";
import type { LLMProvider } from "../providers/types.js";

type View = "stream" | "decisions" | "dashboard";

interface AppProps {
  task: string;
  provider: LLMProvider;
}

export function App({ task, provider }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const rows = stdout?.rows || 24;

  const [view, setView] = useState<View>("stream");
  const [agents, setAgents] = useState<AgentState[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [manager] = useState(
    () => new AgentManager(provider, (states) => setAgents([...states]))
  );
  const prevStatus = useRef<string>("");

  // Start or resume agent
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
      if (resumed.state.status !== "asking") {
        resumed.start();
      }
    } else {
      const agent = manager.spawn(task);
      setSelectedAgent(agent.state.id);
      agent.start();
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const current = agents.find((a) => a.id === selectedAgent) || agents[0];

  // Auto-switch to decision modal when agent starts asking
  useEffect(() => {
    if (!current) return;
    if (current.status === "asking" && prevStatus.current !== "asking") {
      setView("decisions");
    }
    if (current.status !== "asking" && prevStatus.current === "asking") {
      setView("stream");
    }
    prevStatus.current = current.status;
  }, [current?.status]);

  useInput((input, key) => {
    // Global: quit
    if (input === "q" && view !== "decisions") {
      exit();
      return;
    }

    // Escape: close overlays, go back to stream
    if (key.escape) {
      if (view === "dashboard") {
        setView("stream");
        return;
      }
    }

    // d: toggle dashboard
    if (input === "d" && view !== "decisions") {
      setView(view === "dashboard" ? "stream" : "dashboard");
      return;
    }

    // Open decision view manually
    if (input === "i" && current?.status === "asking") {
      setView("decisions");
      return;
    }
  });

  const handleDecisionAnswer = useCallback(
    (value: string) => {
      if (!current) return;
      const agent = manager.get(current.id);
      if (!agent) return;
      agent.sendUserMessage(value);
    },
    [current, manager]
  );

  const handleDecisionsDone = useCallback(() => {
    setView("stream");
  }, []);

  const pendingCount = current
    ? current.decisions.filter((d) => d.answer === null).length
    : 0;

  return (
    <Box flexDirection="column" height={rows}>
      {view === "stream" && (
        <StreamView
          agent={current}
          pendingCount={pendingCount}
          rows={rows}
        />
      )}

      {view === "decisions" && current && (
        <DecisionModal
          agent={current}
          onAnswer={handleDecisionAnswer}
          onDone={handleDecisionsDone}
          rows={rows}
        />
      )}

      {view === "dashboard" && (
        <DashboardOverlay
          agents={agents}
          selectedId={selectedAgent}
          onSelect={setSelectedAgent}
          onClose={() => setView("stream")}
          rows={rows}
        />
      )}
    </Box>
  );
}

/** Default view: streaming output like claude code */
function StreamView({
  agent,
  pendingCount,
  rows,
}: {
  agent: AgentState | undefined;
  pendingCount: number;
  rows: number;
}) {
  if (!agent) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="gray">Starting...</Text>
      </Box>
    );
  }

  // Show last N lines of output that fit the screen
  const outputLines = (agent.currentOutput || "").split("\n");
  const maxLines = rows - 4; // status bar + padding
  const visibleLines = outputLines.slice(-maxLines);

  const statusColor =
    agent.status === "thinking"
      ? "cyan"
      : agent.status === "asking"
        ? "yellow"
        : agent.status === "executing"
          ? "blue"
          : agent.status === "done"
            ? "green"
            : agent.status === "error"
              ? "red"
              : "gray";

  return (
    <Box flexDirection="column" height={rows}>
      {/* Output */}
      <Box flexDirection="column" flexGrow={1} paddingX={1}>
        {agent.status === "thinking" && !agent.currentOutput && (
          <Text color="cyan">Decomposing task...</Text>
        )}
        {visibleLines.map((line, i) => (
          <Text key={i} wrap="wrap">
            {line}
          </Text>
        ))}
      </Box>

      {/* Status bar */}
      <Box paddingX={1} borderStyle="single" borderColor="gray" borderTop borderBottom={false} borderLeft={false} borderRight={false}>
        <Text color={statusColor}>{agent.status}</Text>
        <Text color="gray"> | </Text>
        <Text color="gray">
          {agent.decisions.length} decisions
          {pendingCount > 0 && (
            <Text color="yellow"> ({pendingCount} pending)</Text>
          )}
        </Text>
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          {pendingCount > 0 ? "i:answer " : ""}d:dashboard q:quit
        </Text>
      </Box>
    </Box>
  );
}
