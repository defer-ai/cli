import React, { useState, useEffect, useCallback } from "react";
import { Box, Text, useApp, useInput } from "ink";
import { DecisionsTab } from "./DecisionsTab.js";
import { AgentsTab } from "./AgentsTab.js";
import { GitTab } from "./GitTab.js";
import { InputBar } from "./InputBar.js";
import { AgentManager } from "../agents/manager.js";
import type { AgentState } from "../agents/agent.js";
import type { LLMProvider } from "../providers/types.js";

const TABS = ["Decisions", "Agents", "Git"] as const;
type Tab = (typeof TABS)[number];

interface AppProps {
  task: string;
  provider: LLMProvider;
}

export function App({ task, provider }: AppProps) {
  const { exit } = useApp();
  const [activeTab, setActiveTab] = useState<Tab>("Decisions");
  const [agents, setAgents] = useState<AgentState[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [inputMode, setInputMode] = useState(false);
  const [manager] = useState(
    () => new AgentManager(provider, (states) => setAgents([...states]))
  );

  // Start the first agent with the task
  useEffect(() => {
    const agent = manager.spawn(task);
    setSelectedAgent(agent.state.id);
    agent.start();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const currentAgent = agents.find((a) => a.id === selectedAgent) || agents[0];

  useInput((input, key) => {
    if (inputMode) return;

    if (input === "q" || key.escape) {
      exit();
      return;
    }

    // Tab switching
    if (input === "1") setActiveTab("Decisions");
    if (input === "2") setActiveTab("Agents");
    if (input === "3") setActiveTab("Git");

    // Enter input mode to respond to AI
    if (input === "i" || key.return) {
      if (currentAgent?.status === "asking") {
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

      // Check for revisit command
      const revisitMatch = value.match(/^revisit\s+(D\d+)\s+(.+)/i);
      if (revisitMatch) {
        agent.revisitDecision(revisitMatch[1], revisitMatch[2]);
        return;
      }

      agent.sendUserMessage(value);
    },
    [currentAgent, manager]
  );

  return (
    <Box flexDirection="column" width="100%" height="100%">
      {/* Tab bar */}
      <Box paddingX={1}>
        {TABS.map((tab, i) => (
          <Box key={tab} marginRight={2}>
            <Text
              color={activeTab === tab ? "cyan" : "gray"}
              bold={activeTab === tab}
            >
              [{i + 1}:{tab}]
            </Text>
          </Box>
        ))}
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          q:quit i:input 1-3:tabs
        </Text>
      </Box>

      <Box borderStyle="single" borderColor="gray" flexDirection="column" flexGrow={1}>
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
      <Box paddingX={1}>
        <Text color="gray">
          {currentAgent ? (
            <>
              <Text color="cyan">{currentAgent.id}</Text>
              {" | "}
              <Text
                color={
                  currentAgent.status === "asking"
                    ? "yellow"
                    : currentAgent.status === "thinking"
                      ? "cyan"
                      : currentAgent.status === "error"
                        ? "red"
                        : currentAgent.status === "done"
                          ? "green"
                          : "white"
                }
              >
                {currentAgent.status}
              </Text>
              {" | "}
              {currentAgent.decisions.length} decisions
              {currentAgent.status === "asking" && (
                <Text color="yellow"> | press i to respond</Text>
              )}
            </>
          ) : (
            "No agents"
          )}
        </Text>
      </Box>

      {/* Input bar */}
      {inputMode && (
        <InputBar onSubmit={handleInput} onCancel={() => setInputMode(false)} />
      )}
    </Box>
  );
}
