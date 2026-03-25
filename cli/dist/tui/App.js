import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect, useCallback, useRef } from "react";
import { Box, Text, useApp, useInput, useStdout } from "ink";
import { DecisionModal } from "./DecisionModal.js";
import { DashboardOverlay } from "./DashboardOverlay.js";
import { AgentManager } from "../agents/manager.js";
import { Agent } from "../agents/agent.js";
export function App({ task, provider }) {
    const { exit } = useApp();
    const { stdout } = useStdout();
    const rows = stdout?.rows || 24;
    const [view, setView] = useState("stream");
    const [agents, setAgents] = useState([]);
    const [selectedAgent, setSelectedAgent] = useState(null);
    const [manager] = useState(() => new AgentManager(provider, (states) => setAgents([...states])));
    const prevStatus = useRef("");
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
        }
        else {
            const agent = manager.spawn(task);
            setSelectedAgent(agent.state.id);
            agent.start();
        }
    }, []); // eslint-disable-line react-hooks/exhaustive-deps
    const current = agents.find((a) => a.id === selectedAgent) || agents[0];
    // Auto-switch to decision modal when agent starts asking
    useEffect(() => {
        if (!current)
            return;
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
    const handleDecisionAnswer = useCallback((value) => {
        if (!current)
            return;
        const agent = manager.get(current.id);
        if (!agent)
            return;
        agent.sendUserMessage(value);
    }, [current, manager]);
    const handleDecisionsDone = useCallback(() => {
        setView("stream");
    }, []);
    const pendingCount = current
        ? current.decisions.filter((d) => d.answer === null).length
        : 0;
    return (_jsxs(Box, { flexDirection: "column", height: rows, children: [view === "stream" && (_jsx(StreamView, { agent: current, pendingCount: pendingCount, rows: rows })), view === "decisions" && current && (_jsx(DecisionModal, { agent: current, onAnswer: handleDecisionAnswer, onDone: handleDecisionsDone, rows: rows })), view === "dashboard" && (_jsx(DashboardOverlay, { agents: agents, selectedId: selectedAgent, onSelect: setSelectedAgent, onClose: () => setView("stream"), rows: rows }))] }));
}
/** Default view: streaming output like claude code */
function StreamView({ agent, pendingCount, rows, }) {
    if (!agent) {
        return (_jsx(Box, { flexDirection: "column", padding: 1, children: _jsx(Text, { color: "gray", children: "Starting..." }) }));
    }
    // Show last N lines of output that fit the screen
    const outputLines = (agent.currentOutput || "").split("\n");
    const maxLines = rows - 4; // status bar + padding
    const visibleLines = outputLines.slice(-maxLines);
    const statusColor = agent.status === "thinking"
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
    return (_jsxs(Box, { flexDirection: "column", height: rows, children: [_jsxs(Box, { flexDirection: "column", flexGrow: 1, paddingX: 1, children: [agent.status === "thinking" && !agent.currentOutput && (_jsx(Text, { color: "cyan", children: "Decomposing task..." })), visibleLines.map((line, i) => (_jsx(Text, { wrap: "wrap", children: line }, i)))] }), _jsxs(Box, { paddingX: 1, borderStyle: "single", borderColor: "gray", borderTop: true, borderBottom: false, borderLeft: false, borderRight: false, children: [_jsx(Text, { color: statusColor, children: agent.status }), _jsx(Text, { color: "gray", children: " | " }), _jsxs(Text, { color: "gray", children: [agent.decisions.length, " decisions", pendingCount > 0 && (_jsxs(Text, { color: "yellow", children: [" (", pendingCount, " pending)"] }))] }), _jsx(Box, { flexGrow: 1 }), _jsxs(Text, { color: "gray", dimColor: true, children: [pendingCount > 0 ? "i:answer " : "", "d:dashboard q:quit"] })] })] }));
}
