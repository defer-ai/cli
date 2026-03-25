import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
import { Mascot, statusToMood } from "./Mascot.js";
import { DecisionModal } from "./DecisionModal.js";
import { DomainPriority } from "./DomainPriority.js";
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
    const [domainPrioritiesDone, setDomainPrioritiesDone] = useState(false);
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
    // Auto-open domain priority or decision modal when agent starts asking
    useEffect(() => {
        if (!current)
            return;
        if (current.status === "asking" && prevStatus.current !== "asking") {
            if (!domainPrioritiesDone) {
                setView("domains");
            }
            else {
                setView("decisions");
            }
            setRevisitId(null);
        }
        if (current.status !== "asking" &&
            prevStatus.current === "asking" &&
            (view === "decisions" || view === "domains")) {
            setView("stream");
        }
        prevStatus.current = current.status;
    }, [current?.status, view, domainPrioritiesDone]);
    // Stream output to display, suppress only the defer-decisions JSON block
    useEffect(() => {
        if (!current?.currentOutput)
            return;
        const output = current.currentOutput;
        // During initial decomposition, suppress everything (it's the JSON block)
        if (current.phase === "decomposing" &&
            current.status === "thinking" &&
            current.decisions.length === 0) {
            return;
        }
        // Strip out defer-decisions JSON blocks but keep everything else
        const cleaned = output.replace(/```defer-decisions[\s\S]*?```/g, "").trim();
        if (cleaned) {
            setOutputLines(cleaned.split("\n"));
        }
    }, [current?.currentOutput, current?.phase, current?.status, current?.decisions.length]);
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
    const handleDomainPriorities = useCallback((priorities) => {
        setDomainPrioritiesDone(true);
        if (!current) {
            setView("decisions");
            return;
        }
        const agent = manager.get(current.id);
        if (!agent) {
            setView("decisions");
            return;
        }
        // Auto-delegate "skip" categories
        let changed = false;
        for (const d of agent.state.decisions) {
            if (priorities[d.category] === "skip" && d.answer === null) {
                d.answer = "Choose for me";
                d.delegated = true;
                d.date = new Date().toISOString().split("T")[0];
                changed = true;
            }
        }
        if (changed) {
            agent["persist"]();
        }
        // For new categories the user added (not in existing decisions),
        // tell the AI to generate decisions for them
        const existingCats = new Set(agent.state.decisions.map((d) => d.category));
        const newCats = Object.keys(priorities).filter((cat) => !existingCats.has(cat) && priorities[cat] !== "skip");
        if (newCats.length > 0) {
            const paranoidCats = Object.entries(priorities)
                .filter(([_, level]) => level === "paranoid")
                .map(([cat]) => cat);
            let msg = `Additional domains to cover: ${newCats.join(", ")}.`;
            if (paranoidCats.length > 0) {
                msg += ` For these domains, go deep with sub-questions: ${paranoidCats.join(", ")}.`;
            }
            msg += ` Output a new \`\`\`defer-decisions block for these.`;
            agent.sendUserMessage(msg);
            setView("stream");
            return;
        }
        // Check if there are still pending decisions
        const stillPending = agent.state.decisions.some((d) => d.answer === null);
        if (stillPending) {
            // For paranoid categories, tell the AI to expand
            const paranoidCats = Object.entries(priorities)
                .filter(([_, level]) => level === "paranoid")
                .map(([cat]) => cat);
            if (paranoidCats.length > 0) {
                const msg = `For these domains, I want deeper sub-questions: ${paranoidCats.join(", ")}. Generate additional decisions for them in a \`\`\`defer-decisions block.`;
                agent.sendUserMessage(msg);
                setView("stream");
                return;
            }
            setView("decisions");
        }
        else {
            // All delegated, proceed
            const summary = agent.state.decisions
                .map((d) => `${d.id}: ${d.question} -> ${d.delegated ? "DELEGATED: " : ""}${d.answer}`)
                .join("\n");
            agent.sendUserMessage(`Task: ${agent.state.task}\n\nDecision record:\n${summary}\n\nAll decisions are answered. Proceed with implementation.`);
            setView("stream");
        }
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
    // Domain priority screen
    if (view === "domains" && current && current.decisions.length > 0) {
        return (_jsx(DomainPriority, { decisions: current.decisions, onComplete: handleDomainPriorities, rows: rows }));
    }
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
                                        outputLines.length === 0 && (_jsx(Text, { color: "cyan", children: "Decomposing task..." })), visible.map((line, i) => (_jsx(Text, { wrap: "wrap", children: line }, i)))] }))] })] }), _jsxs(Box, { paddingX: 1, marginTop: 1, children: [current ? (_jsxs(_Fragment, { children: [_jsx(Text, { color: statusColor, dimColor: true, children: current.status }), current.decisions.length > 0 && (_jsxs(Text, { color: "gray", dimColor: true, children: [" | ", current.decisions.length - pendingCount, "/", current.decisions.length, " decisions"] })), pendingCount > 0 && (_jsxs(Text, { color: "yellow", dimColor: true, children: [" ", "(", pendingCount, " pending)"] }))] })) : (_jsx(Text, { color: "gray", dimColor: true, children: model })), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "/help" })] }), _jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer > " }), _jsx(Text, { children: inputValue }), _jsx(Text, { color: "gray", children: "|" })] })] }));
}
