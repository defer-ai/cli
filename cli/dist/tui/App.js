import { jsxs as _jsxs, jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import React, { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { Banner, Header } from "./Banner.js";
import { DecisionModal } from "./DecisionModal.js";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
import { statusToMood } from "./Mascot.js";
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
        // Tab: cycle through views
        if (key.tab) {
            const viewCycle = ["stream", "decisions", "git"];
            const currentView = view === "banner" ? "stream" : view;
            const idx = viewCycle.indexOf(currentView);
            const next = viewCycle[(idx + 1) % viewCycle.length];
            // Skip decisions tab if no decisions exist
            if (next === "decisions" && (!current || current.decisions.length === 0)) {
                setView("git");
            }
            else {
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
    // All views now render inside the same side-panel layout below
    const mood = current
        ? statusToMood(current.status, current.phase)
        : "idle";
    const tabs = [
        { key: "stream", label: "Chat", icon: ">" },
        { key: "decisions", label: "Decide", icon: "◇" },
        { key: "git", label: "Git", icon: "±" },
    ];
    const activeTabKey = view === "banner" ? "stream" : view === "dashboard" ? "stream" : view;
    // Main layout: side panel + content
    return (_jsxs(Box, { flexDirection: "row", height: rows, children: [_jsx(Box, { flexDirection: "column", width: 6, paddingTop: 1, children: tabs.map((tab) => {
                    const isActive = activeTabKey === tab.key;
                    return (_jsx(Box, { paddingX: 1, children: _jsxs(Text, { color: isActive ? "cyan" : "gray", bold: isActive, dimColor: !isActive, children: [isActive ? "▸" : " ", " ", tab.icon] }) }, tab.key));
                }) }), _jsx(Box, { flexDirection: "column", flexGrow: 1, children: _jsx(Box, { flexDirection: "column", flexGrow: 1, children: view === "decisions" && current ? (_jsx(DecisionModal, { agent: current, onAnswer: handleDecisionAnswer, onAsk: handleDecisionAsk, onRevise: handleDecisionRevise, onDone: () => setView("stream"), rows: rows - 2 })) : (_jsxs(_Fragment, { children: [_jsx(Box, { paddingX: 1, children: view === "banner" && !current ? (_jsx(Banner, { model: model, cwd: process.cwd(), mood: mood })) : (_jsx(Header, { model: model, mood: mood })) }), _jsx(Box, { flexDirection: "column", flexGrow: 1, paddingX: 1, children: view === "git" ? (_jsx(GitView, {})) : (_jsxs(_Fragment, { children: [current?.status === "thinking" &&
                                            outputLines.length === 0 && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "cyan", children: "Decomposing task..." }) })), visible.map((line, i) => (_jsx(Text, { wrap: "wrap", children: line }, i)))] })) }), _jsxs(Box, { paddingX: 1, children: [current ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: statusColor, dimColor: true, children: current.status }), current.decisions.length > 0 && (_jsxs(Text, { color: "gray", dimColor: true, children: [" | ", current.decisions.length - pendingCount, "/", current.decisions.length, " decisions"] })), pendingCount > 0 && (_jsxs(Text, { color: "yellow", dimColor: true, children: [" ", "(", pendingCount, " pending)"] }))] })) : (_jsx(Text, { color: "gray", dimColor: true, children: model })), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "tab:switch  /help" })] }), _jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer > " }), _jsx(Text, { children: inputValue }), _jsx(Text, { color: "gray", children: "|" })] })] })) }) })] }));
}
/** Inline git info view */
function GitView() {
    const [info, setInfo] = React.useState(null);
    React.useEffect(() => {
        try {
            const { execSync } = require("node:child_process");
            execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });
            const branch = execSync("git branch --show-current", {
                encoding: "utf-8",
            }).trim();
            let commits = [];
            try {
                commits = execSync("git log --oneline -10", { encoding: "utf-8" })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch { }
            let dirty = [];
            try {
                dirty = execSync("git status --short", { encoding: "utf-8" })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch { }
            setInfo({ branch, commits, dirty });
        }
        catch {
            setInfo(null);
        }
    }, []);
    if (!info) {
        return (_jsx(Box, { paddingX: 1, marginTop: 1, children: _jsx(Text, { color: "gray", children: "Not a git repository." }) }));
    }
    return (_jsxs(Box, { flexDirection: "column", paddingX: 1, marginTop: 1, children: [_jsx(Box, { children: _jsx(Text, { color: "cyan", bold: true, children: info.branch }) }), info.dirty.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsxs(Text, { color: "yellow", dimColor: true, children: [info.dirty.length, " uncommitted"] }), info.dirty.slice(0, 8).map((f, i) => (_jsxs(Text, { color: "gray", dimColor: true, children: ["  ", f] }, i)))] })), info.commits.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Recent commits" }), info.commits.map((c, i) => (_jsxs(Text, { color: "gray", dimColor: true, children: ["  ", c] }, i)))] }))] }));
}
