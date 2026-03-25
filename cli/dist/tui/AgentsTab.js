import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import React from "react";
import { Box, Text, useInput } from "ink";
const statusColors = {
    idle: "gray",
    thinking: "cyan",
    asking: "yellow",
    executing: "blue",
    done: "green",
    error: "red",
};
export function AgentsTab({ agents, selectedId, onSelect }) {
    const [cursorIdx, setCursorIdx] = React.useState(0);
    useInput((input, key) => {
        if (input === "j" || key.downArrow) {
            setCursorIdx((i) => Math.min(i + 1, agents.length - 1));
        }
        if (input === "k" || key.upArrow) {
            setCursorIdx((i) => Math.max(i - 1, 0));
        }
        if (key.return && agents[cursorIdx]) {
            onSelect(agents[cursorIdx].id);
        }
    });
    if (agents.length === 0) {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "No agents spawned." }) }));
    }
    const selected = agents[cursorIdx];
    return (_jsxs(Box, { padding: 1, flexDirection: "row", children: [_jsxs(Box, { flexDirection: "column", width: "40%", children: [_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "j/k:navigate enter:select" }) }), agents.map((agent, i) => {
                        const isActive = agent.id === selectedId;
                        const isCursor = i === cursorIdx;
                        const answered = agent.decisions.filter((d) => d.answer !== null).length;
                        const total = agent.decisions.length;
                        const delegated = agent.decisions.filter((d) => d.delegated).length;
                        return (_jsxs(Box, { flexDirection: "column", marginBottom: 1, children: [_jsxs(Box, { children: [_jsx(Text, { color: isCursor ? "cyan" : "gray", children: isCursor ? "> " : "  " }), _jsx(Text, { color: isActive ? "cyan" : "white", bold: isActive, children: agent.id }), _jsx(Text, { children: " " }), _jsx(Text, { color: statusColors[agent.status] || "white", children: agent.status })] }), _jsx(Box, { paddingLeft: 4, children: _jsxs(Text, { color: "gray", children: [answered, "/", total, " answered", delegated > 0 ? `, ${delegated} delegated` : ""] }) })] }, agent.id));
                    })] }), selected && (_jsxs(Box, { flexDirection: "column", width: "60%", borderStyle: "single", borderColor: "gray", paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: selected.id }), _jsxs(Box, { marginTop: 1, children: [_jsx(Text, { color: "gray", children: "Task: " }), _jsx(Text, { wrap: "wrap", children: selected.task })] }), _jsxs(Box, { marginTop: 1, children: [_jsx(Text, { color: "gray", children: "Status: " }), _jsx(Text, { color: statusColors[selected.status] || "white", children: selected.status }), _jsx(Text, { color: "gray", children: " | Phase: " }), _jsx(Text, { children: selected.phase })] }), _jsx(Box, { marginTop: 1, children: _jsxs(Text, { color: "gray", children: ["Decisions: ", selected.decisions.length, " total,", " ", selected.decisions.filter((d) => d.answer !== null).length, " ", "answered,", " ", selected.decisions.filter((d) => d.answer === null).length, " ", "pending"] }) }), selected.currentOutput && (_jsxs(Box, { marginTop: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: "Latest output:" }), _jsx(Text, { wrap: "wrap", color: "gray", children: selected.currentOutput.length > 300
                                    ? selected.currentOutput.slice(-300) + "..."
                                    : selected.currentOutput })] })), selected.error && (_jsx(Box, { marginTop: 1, children: _jsxs(Text, { color: "red", children: ["Error: ", selected.error] }) }))] }))] }));
}
