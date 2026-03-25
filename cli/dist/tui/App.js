import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
import { Mascot, statusToMood } from "./Mascot.js";
import { DecisionModal } from "./DecisionModal.js";
const VERSION = "0.1.0";
export function App({ task, provider }) {
    const { exit } = useApp();
    const { stdout } = useStdout();
    const rows = stdout?.rows || 24;
    const [view, setView] = useState("stream");
    const [agents, setAgents] = useState([]);
    const [selectedAgent, setSelectedAgent] = useState(null);
    const [inputValue, setInputValue] = useState("");
    const [outputLines, setOutputLines] = useState([]);
    const [showBanner, setShowBanner] = useState(!task);
    const [revisitId, setRevisitId] = useState(null);
    const [model, setModel] = useState(provider.model || "sonnet");
    const [manager] = useState(() => new AgentManager(provider, (states) => setAgents([...states])));
    const prevStatus = useRef("");
    const current = agents.find((a) => a.id === selectedAgent) || agents[0];
    const mood = current
        ? statusToMood(current.status, current.phase)
        : "idle";
    const pendingCount = current
        ? current.decisions.filter((d) => d.answer === null).length
        : 0;
    // On mount: resume or start
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
            setShowBanner(false);
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
    // Auto-open decision modal when agent starts asking
    useEffect(() => {
        if (!current)
            return;
        if (current.status === "asking" && prevStatus.current !== "asking") {
            setView("decisions");
            setRevisitId(null);
        }
        if (current.status !== "asking" &&
            prevStatus.current === "asking" &&
            view === "decisions") {
            setView("stream");
        }
        prevStatus.current = current.status;
    }, [current?.status, view]);
    // Suppress raw decomposition output
    useEffect(() => {
        if (!current?.currentOutput)
            return;
        if (current.phase === "decomposing" && current.status === "thinking")
            return;
        const output = current.currentOutput;
        if (output.includes("```defer-decisions"))
            return;
        setOutputLines(output.split("\n"));
    }, [current?.currentOutput, current?.phase, current?.status]);
    const startTask = useCallback((taskText) => {
        const agent = manager.spawn(taskText);
        setSelectedAgent(agent.state.id);
        setOutputLines([]);
        setShowBanner(false);
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
                    "  /help                Show this help",
                    "  /model <name>        Switch model (sonnet, opus, haiku)",
                    "  /decisions           View all decisions inline",
                    "  /revisit <id>        Revisit a specific decision",
                    "  /clear               Clear output",
                    "  /quit                Exit",
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
            case "decisions":
            case "status":
                if (!current || current.decisions.length === 0) {
                    setOutputLines((prev) => [...prev, "  No decisions yet."]);
                    break;
                }
                // Print decisions inline
                const lines = [""];
                let lastCat = "";
                for (const d of current.decisions) {
                    if (d.category !== lastCat) {
                        lines.push(`  ${d.category}`);
                        lastCat = d.category;
                    }
                    const icon = d.answer === null ? "○" : d.delegated ? "◆" : "✓";
                    const color = d.answer === null ? "" : "";
                    const answer = d.answer === null
                        ? "pending"
                        : d.delegated
                            ? `delegated: ${d.answer}`
                            : d.answer;
                    lines.push(`    ${icon} ${d.id}  ${d.question}  →  ${answer}`);
                }
                lines.push("");
                lines.push("  Use /revisit <id> to change a decision.");
                lines.push("");
                setOutputLines((prev) => [...prev, ...lines]);
                break;
            case "revisit":
                if (!parts[1]) {
                    setOutputLines((prev) => [
                        ...prev,
                        "  Usage: /revisit <id>  (e.g. /revisit STACK-001)",
                    ]);
                    break;
                }
                if (!current) {
                    setOutputLines((prev) => [...prev, "  No active session."]);
                    break;
                }
                const id = parts[1].toUpperCase();
                const decision = current.decisions.find((d) => d.id === id);
                if (!decision) {
                    setOutputLines((prev) => [
                        ...prev,
                        `  Decision ${id} not found. Use /decisions to see all.`,
                    ]);
                    break;
                }
                setRevisitId(id);
                setView("decisions");
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
        if (current) {
            const agent = manager.get(current.id);
            if (agent) {
                setOutputLines((prev) => [...prev, "", `  > ${value}`, ""]);
                agent.sendUserMessage(value);
                setView("stream");
                return;
            }
        }
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
        if (key.return) {
            handleSubmit();
            return;
        }
        if (key.backspace || key.delete) {
            setInputValue((v) => v.slice(0, -1));
            return;
        }
        if (input === "c" && key.ctrl) {
            exit();
            return;
        }
        if (input && !key.ctrl && !key.meta && !key.tab && !key.escape) {
            setInputValue((v) => v + input);
        }
    });
    // Decision modal (full screen)
    if (view === "decisions" && current) {
        return (_jsx(DecisionModal, { agent: current, onAnswer: handleDecisionAnswer, onAsk: handleDecisionAsk, onRevise: handleDecisionRevise, onDone: () => {
                setView("stream");
                setRevisitId(null);
            }, focusId: revisitId, rows: rows }));
    }
    // Stream view
    const maxVisible = rows - 4;
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
    return (_jsxs(Box, { flexDirection: "column", height: rows, children: [_jsxs(Box, { flexGrow: 1, children: [_jsx(Box, { flexDirection: "column", paddingX: 1, paddingTop: 1, children: _jsx(Mascot, { mood: mood }) }), _jsxs(Box, { flexDirection: "column", flexGrow: 1, paddingX: 1, children: [_jsxs(Box, { paddingTop: 1, marginBottom: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer" }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", "v", VERSION, " | ", model] })] }), showBanner && !current ? (_jsxs(Box, { flexDirection: "column", children: [_jsxs(Text, { color: "gray", dimColor: true, children: ["cwd", " ", process.cwd().replace(process.env.HOME || "", "~")] }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "Type a task to start. /help for commands." }) })] })) : (_jsxs(Box, { flexDirection: "column", flexGrow: 1, children: [current?.status === "thinking" &&
                                        outputLines.length === 0 && (_jsx(Text, { color: "cyan", children: "Decomposing task..." })), visible.map((line, i) => (_jsx(Text, { wrap: "wrap", children: line }, i)))] }))] })] }), _jsxs(Box, { paddingX: 1, children: [current ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: statusColor, dimColor: true, children: current.status }), current.decisions.length > 0 && (_jsxs(Text, { color: "gray", dimColor: true, children: [" | ", current.decisions.length - pendingCount, "/", current.decisions.length, " decisions"] })), pendingCount > 0 && (_jsxs(Text, { color: "yellow", dimColor: true, children: [" ", "(", pendingCount, " pending)"] }))] })) : (_jsx(Text, { color: "gray", dimColor: true, children: model })), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "/help" })] }), _jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer > " }), _jsx(Text, { children: inputValue }), _jsx(Text, { color: "gray", children: "|" })] })] }));
}
