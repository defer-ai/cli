import React, { useState, useEffect, useCallback } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { DecisionsTab } from "./DecisionsTab.js";
import { AgentsTab } from "./AgentsTab.js";
import { GitTab } from "./GitTab.js";
import { InputBar } from "./InputBar.js";
import { AgentManager } from "../agents/manager.js";
import { Agent, type AgentState } from "../agents/agent.js";
import type { LLMProvider } from "../providers/types.js";

const TABS = ["Decisions", "Agents", "Git"] as const;
type Tab = (typeof TABS)[number];

interface AppProps {
  task: string;
  provider: LLMProvider;
}

export function App({ task, provider }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const rows = stdout?.rows || 24;

  const [activeTab, setActiveTab] = useState<Tab>("Decisions");
  const [agents, setAgents] = useState<AgentState[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [inputMode, setInputMode] = useState(false);
  const [manager] = useState(
    () => new AgentManager(provider, (states) => setAgents([...states]))
  );

  // Clear screen on mount
  useEffect(() => {
    process.stdout.write("\x1b[2J\x1b[H");
  }, []);

  // Try to resume a previous session, or start fresh
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
      // If there are pending decisions, just show them (don't re-run AI)
      if (resumed.state.status !== "asking") {
        resumed.start();
      }
    } else {
      const agent = manager.spawn(task);
      setSelectedAgent(agent.state.id);
      agent.start();
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const currentAgent =
    agents.find((a) => a.id === selectedAgent) || agents[0];

  useInput((input, key) => {
    if (inputMode) return;

    if (input === "q" || key.escape) {
      exit();
      return;
    }

    if (input === "1") setActiveTab("Decisions");
    if (input === "2") setActiveTab("Agents");
    if (input === "3") setActiveTab("Git");

    if (input === "i" || key.return) {
      if (currentAgent?.status === "asking" || currentAgent?.status === "done") {
        setInputMode(true);
      }
    }
  });

  const handleInput = useCallback(
    (value: string) => {
      setInputMode(false);
      if (!value.trim() || !currentAgent) return;

      const agent = manager.get(currentAgent.id);
      if (!agent) return;

      const revisitMatch = value.match(/^revisit\s+(D\d+)\s+(.+)/i);
      if (revisitMatch) {
        agent.revisitDecision(revisitMatch[1], revisitMatch[2]);
        return;
      }

      agent.sendUserMessage(value);
    },
    [currentAgent, manager]
  );

  // Calculate content height
  const contentHeight = Math.max(rows - 4, 10); // tabs(1) + border(2) + status(1)

  const statusColor = currentAgent
    ? currentAgent.status === "asking"
      ? "yellow"
      : currentAgent.status === "thinking"
        ? "cyan"
        : currentAgent.status === "error"
          ? "red"
          : currentAgent.status === "done"
            ? "green"
            : "white"
    : "gray";

  return (
    <Box flexDirection="column">
      {/* Tab bar */}
      <Box>
        <Text> </Text>
        {TABS.map((tab, i) => (
          <React.Fragment key={tab}>
            <Text
              color={activeTab === tab ? "cyan" : "gray"}
              bold={activeTab === tab}
            >
              [{i + 1}:{tab}]
            </Text>
            <Text> </Text>
          </React.Fragment>
        ))}
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          q:quit i:input 1-3:tabs
        </Text>
      </Box>

      {/* Content */}
      <Box
        borderStyle="single"
        borderColor="gray"
        flexDirection="column"
        height={contentHeight}
        overflow="hidden"
      >
        {activeTab === "Decisions" && (
          <DecisionsTab agent={currentAgent} />
        )}
        {activeTab === "Agents" && (
          <AgentsTab
            agents={agents}
            selectedId={selectedAgent}
            onSelect={setSelectedAgent}
          />
        )}
        {activeTab === "Git" && <GitTab />}
      </Box>

      {/* Status bar */}
      <Box>
        <Text> </Text>
        {currentAgent ? (
          <>
            <Text color="cyan">{currentAgent.id}</Text>
            <Text color="gray"> | </Text>
            <Text color={statusColor}>{currentAgent.status}</Text>
            <Text color="gray"> | </Text>
            <Text color="gray">{currentAgent.decisions.length} decisions</Text>
            {currentAgent.status === "asking" && (
              <Text color="yellow"> | press i to respond</Text>
            )}
          </>
        ) : (
          <Text color="gray">No agents</Text>
        )}
      </Box>

      {/* Input bar */}
      {inputMode && (
        <InputBar
          onSubmit={handleInput}
          onCancel={() => setInputMode(false)}
        />
      )}
    </Box>
  );
}
