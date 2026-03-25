import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import React, { useState, useEffect, useCallback } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { DecisionsTab } from "./DecisionsTab.js";
import { AgentsTab } from "./AgentsTab.js";
import { GitTab } from "./GitTab.js";
import { InputBar } from "./InputBar.js";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
const TABS = ["Decisions", "Agents", "Git"];
export function App({ task, provider }) {
    const { exit } = useApp();
    const { stdout } = useStdout();
    const rows = stdout?.rows || 24;
    const [activeTab, setActiveTab] = useState("Decisions");
    const [agents, setAgents] = useState([]);
    const [selectedAgent, setSelectedAgent] = useState(null);
    const [inputMode, setInputMode] = useState(false);
    const [manager] = useState(() => new AgentManager(provider, (states) => setAgents([...states])));
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
            if (resumed.state.status !== "asking") {
                resumed.start();
            }
        }
        else {
            const agent = manager.spawn(task);
            setSelectedAgent(agent.state.id);
            agent.start();
        }
    }, []); // eslint-disable-line react-hooks/exhaustive-deps
    const currentAgent = agents.find((a) => a.id === selectedAgent) || agents[0];
    useInput((input, key) => {
        if (inputMode)
            return;
        if (input === "q") {
            exit();
            return;
        }
        // Tab key cycles through tabs
        if (key.tab) {
            setActiveTab((prev) => {
                const idx = TABS.indexOf(prev);
                return TABS[(idx + 1) % TABS.length];
            });
            return;
        }
        // Enter input mode
        if (input === "i" || key.return) {
            if (currentAgent?.status === "asking" ||
                currentAgent?.status === "done") {
                setInputMode(true);
            }
        }
    });
    const handleInput = useCallback((value) => {
        setInputMode(false);
        if (!value.trim() || !currentAgent)
            return;
        const agent = manager.get(currentAgent.id);
        if (!agent)
            return;
        const revisitMatch = value.match(/^revisit\s+(D\d+)\s+(.+)/i);
        if (revisitMatch) {
            agent.revisitDecision(revisitMatch[1], revisitMatch[2]);
            return;
        }
        agent.sendUserMessage(value);
    }, [currentAgent, manager]);
    const contentHeight = Math.max(rows - 4, 10);
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
    return (_jsxs(Box, { flexDirection: "column", children: [_jsxs(Box, { children: [_jsx(Text, { children: " " }), TABS.map((tab) => (_jsxs(React.Fragment, { children: [_jsxs(Text, { color: activeTab === tab ? "cyan" : "gray", bold: activeTab === tab, children: ["[", tab, "]"] }), _jsx(Text, { children: " " })] }, tab))), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "q:quit i:respond tab:switch" })] }), _jsxs(Box, { borderStyle: "single", borderColor: "gray", flexDirection: "column", height: contentHeight, overflow: "hidden", children: [activeTab === "Decisions" && (_jsx(DecisionsTab, { agent: currentAgent })), activeTab === "Agents" && (_jsx(AgentsTab, { agents: agents, selectedId: selectedAgent, onSelect: setSelectedAgent })), activeTab === "Git" && _jsx(GitTab, {})] }), _jsxs(Box, { children: [_jsx(Text, { children: " " }), currentAgent ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: "cyan", children: currentAgent.id }), _jsx(Text, { color: "gray", children: " | " }), _jsx(Text, { color: statusColor, children: currentAgent.status }), _jsx(Text, { color: "gray", children: " | " }), _jsxs(Text, { color: "gray", children: [currentAgent.decisions.length, " decisions"] }), currentAgent.status === "asking" && (_jsx(Text, { color: "yellow", children: " | press i to respond" }))] })) : (_jsx(Text, { color: "gray", children: "No agents" }))] }), inputMode && (_jsx(InputBar, { options: currentAgent?.parsedOptions && currentAgent.parsedOptions.length > 0
                    ? currentAgent.parsedOptions
                    : undefined, onSubmit: handleInput, onCancel: () => setInputMode(false) }))] }));
}
