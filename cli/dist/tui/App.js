import { jsxs as _jsxs, jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useEffect, useCallback } from "react";
import { Box, Text, useApp, useInput } from "ink";
import { DecisionsTab } from "./DecisionsTab.js";
import { AgentsTab } from "./AgentsTab.js";
import { GitTab } from "./GitTab.js";
import { InputBar } from "./InputBar.js";
import { AgentManager } from "../agents/manager.js";
const TABS = ["Decisions", "Agents", "Git"];
export function App({ task, provider }) {
    const { exit } = useApp();
    const [activeTab, setActiveTab] = useState("Decisions");
    const [agents, setAgents] = useState([]);
    const [selectedAgent, setSelectedAgent] = useState(null);
    const [inputMode, setInputMode] = useState(false);
    const [manager] = useState(() => new AgentManager(provider, (states) => setAgents([...states])));
    // Start the first agent with the task
    useEffect(() => {
        const agent = manager.spawn(task);
        setSelectedAgent(agent.state.id);
        agent.start();
    }, []); // eslint-disable-line react-hooks/exhaustive-deps
    const currentAgent = agents.find((a) => a.id === selectedAgent) || agents[0];
    useInput((input, key) => {
        if (inputMode)
            return;
        if (input === "q" || key.escape) {
            exit();
            return;
        }
        // Tab switching
        if (input === "1")
            setActiveTab("Decisions");
        if (input === "2")
            setActiveTab("Agents");
        if (input === "3")
            setActiveTab("Git");
        // Enter input mode to respond to AI
        if (input === "i" || key.return) {
            if (currentAgent?.status === "asking") {
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
        // Check for revisit command
        const revisitMatch = value.match(/^revisit\s+(D\d+)\s+(.+)/i);
        if (revisitMatch) {
            agent.revisitDecision(revisitMatch[1], revisitMatch[2]);
            return;
        }
        agent.sendUserMessage(value);
    }, [currentAgent, manager]);
    return (_jsxs(Box, { flexDirection: "column", width: "100%", height: "100%", children: [_jsxs(Box, { paddingX: 1, children: [TABS.map((tab, i) => (_jsx(Box, { marginRight: 2, children: _jsxs(Text, { color: activeTab === tab ? "cyan" : "gray", bold: activeTab === tab, children: ["[", i + 1, ":", tab, "]"] }) }, tab))), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "q:quit i:input 1-3:tabs" })] }), _jsxs(Box, { borderStyle: "single", borderColor: "gray", flexDirection: "column", flexGrow: 1, children: [activeTab === "Decisions" && (_jsx(DecisionsTab, { agent: currentAgent })), activeTab === "Agents" && (_jsx(AgentsTab, { agents: agents, selectedId: selectedAgent, onSelect: setSelectedAgent })), activeTab === "Git" && _jsx(GitTab, {})] }), _jsx(Box, { paddingX: 1, children: _jsx(Text, { color: "gray", children: currentAgent ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: "cyan", children: currentAgent.id }), " | ", _jsx(Text, { color: currentAgent.status === "asking"
                                    ? "yellow"
                                    : currentAgent.status === "thinking"
                                        ? "cyan"
                                        : currentAgent.status === "error"
                                            ? "red"
                                            : currentAgent.status === "done"
                                                ? "green"
                                                : "white", children: currentAgent.status }), " | ", currentAgent.decisions.length, " decisions", currentAgent.status === "asking" && (_jsx(Text, { color: "yellow", children: " | press i to respond" }))] })) : ("No agents") }) }), inputMode && (_jsx(InputBar, { onSubmit: handleInput, onCancel: () => setInputMode(false) }))] }));
}
