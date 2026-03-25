import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { Banner, Header } from "./Banner.js";
import { DecisionModal } from "./DecisionModal.js";
import { DashboardOverlay } from "./DashboardOverlay.js";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
export function App({ task, provider }) {
    const { exit } = useApp();
    const { stdout } = useStdout();
    const rows = stdout?.rows || 24;
    const [view, setView] = useState(task ? "stream" : "banner");
    const [agents, setAgents] = useState([]);
    const [selectedAgent, setSelectedAgent] = useState(null);
    const [inputValue, setInputValue] = useState("");
    const [outputLines, setOutputLines] = useState([]);
    const [model, setModel] = useState(provider.model || "sonnet");
    const [manager] = useState(() => new AgentManager(provider, (states) => setAgents([...states])));
    const prevStatus = useRef("");
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
            if (resumed.state.status !== "asking" &&
                resumed.state.status !== "done") {
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
        if (!current)
            return;
        if (current.status === "asking" && prevStatus.current !== "asking") {
            setView("decisions");
        }
        if (current.status !== "asking" &&
            prevStatus.current === "asking" &&
            view === "decisions") {
            setView("stream");
        }
        prevStatus.current = current.status;
    }, [current?.status, view]);
    // Track output lines, but suppress raw decision decomposition output
    useEffect(() => {
        if (!current?.currentOutput)
            return;
        // Don't show raw output while decomposing (it contains the JSON block)
        if (current.phase === "decomposing" && current.status === "thinking")
            return;
        // Don't show output that contains the defer-decisions JSON block
        const output = current.currentOutput;
        if (output.includes("```defer-decisions"))
            return;
        setOutputLines(output.split("\n"));
    }, [current?.currentOutput, current?.phase, current?.status]);
    const startTask = useCallback((taskText) => {
        const agent = manager.spawn(taskText);
        setSelectedAgent(agent.state.id);
        setOutputLines([]);
        setView("stream");
        agent.start();
    }, [manager]);
    const handleSlashCommand = useCallback((cmd) => {
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
                    provider.setModel(m);
                    setModel(m);
                    setOutputLines((prev) => [
                        ...prev,
                        `  Model switched to ${m}`,
                    ]);
                }
                else {
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
                }
                else {
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
    }, [provider, model, current, exit]);
    const handleSubmit = useCallback(() => {
        const value = inputValue.trim();
        setInputValue("");
        if (!value)
            return;
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
    const handleDecisionAnswer = useCallback((value) => {
        if (!current)
            return;
        const agent = manager.get(current.id);
        if (!agent)
            return;
        agent.sendUserMessage(value);
    }, [current, manager]);
    const handleDecisionAsk = useCallback((decisionId, question) => {
        if (!current)
            return;
        const agent = manager.get(current.id);
        if (!agent)
            return;
        const d = agent.state.decisions.find((d) => d.id === decisionId);
        if (!d)
            return;
        agent.sendUserMessage(`Question about ${decisionId} ("${d.question}"): ${question}`);
    }, [current, manager]);
    const handleDecisionRevise = useCallback((decisionId, newAnswer) => {
        if (!current)
            return;
        const agent = manager.get(current.id);
        if (!agent)
            return;
        agent.revisitDecision(decisionId, newAnswer);
    }, [current, manager]);
    useInput((input, key) => {
        // Decision modal handles its own input
        if (view === "decisions")
            return;
        // Dashboard handles its own input
        if (view === "dashboard")
            return;
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
        // Tab: cycle between views
        if (key.tab) {
            if (view === "stream" || view === "banner") {
                if (current && current.decisions.length > 0) {
                    setView("decisions");
                }
            }
            else {
                setView("stream");
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
    const statusColor = !current || current.status === "idle"
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
        return (_jsx(DecisionModal, { agent: current, onAnswer: handleDecisionAnswer, onAsk: handleDecisionAsk, onRevise: handleDecisionRevise, onDone: () => setView("stream"), rows: rows }));
    }
    // Dashboard overlay
    if (view === "dashboard") {
        return (_jsx(DashboardOverlay, { agents: agents, selectedId: selectedAgent, onSelect: setSelectedAgent, onClose: () => setView("stream"), rows: rows }));
    }
    const tabs = [
        { key: "stream", label: "Chat", icon: ">" },
        { key: "decisions", label: "Decisions", icon: "◇", count: current?.decisions.length || 0 },
    ];
    // Main layout: side panel + content
    return (_jsxs(Box, { flexDirection: "row", height: rows, children: [_jsx(Box, { flexDirection: "column", width: 4, borderStyle: "single", borderColor: "gray", borderRight: true, borderLeft: false, borderTop: false, borderBottom: false, paddingTop: 1, children: tabs.map((tab) => {
                    const isActive = view === tab.key || (view === "banner" && tab.key === "stream");
                    return (_jsx(Box, { paddingX: 1, marginBottom: 1, children: _jsx(Text, { color: isActive ? "cyan" : "gray", bold: isActive, dimColor: !isActive, children: tab.icon }) }, tab.key));
                }) }), _jsxs(Box, { flexDirection: "column", flexGrow: 1, children: [_jsxs(Box, { flexDirection: "column", flexGrow: 1, paddingX: 1, children: [view === "banner" && !current ? (_jsx(Banner, { model: model, cwd: process.cwd() })) : (_jsx(Header, { model: model })), current?.status === "thinking" && outputLines.length === 0 && (_jsx(Box, { marginTop: 1, paddingX: 1, children: _jsx(Text, { color: "cyan", children: "Decomposing task..." }) })), visible.map((line, i) => (_jsx(Text, { wrap: "wrap", children: line }, i)))] }), _jsxs(Box, { paddingX: 1, children: [current ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: statusColor, dimColor: true, children: current.status }), current.decisions.length > 0 && (_jsx(_Fragment, { children: _jsxs(Text, { color: "gray", dimColor: true, children: [" | ", current.decisions.length - pendingCount, "/", current.decisions.length, " decisions"] }) })), pendingCount > 0 && (_jsxs(Text, { color: "yellow", dimColor: true, children: [" ", "(", pendingCount, " pending)"] }))] })) : (_jsx(Text, { color: "gray", dimColor: true, children: model })), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "tab:switch  /help" })] }), _jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer > " }), _jsx(Text, { children: inputValue }), _jsx(Text, { color: "gray", children: "|" })] })] })] }));
}
