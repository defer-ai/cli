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
    return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "j/k:navigate enter:select" }) }), agents.map((agent, i) => {
                const isSelected = agent.id === selectedId;
                const isCursor = i === cursorIdx;
                return (_jsxs(Box, { paddingLeft: 1, children: [_jsx(Text, { color: isCursor ? "cyan" : "white", children: isCursor ? "> " : "  " }), _jsx(Text, { color: isSelected ? "cyan" : "white", bold: isSelected, children: agent.id }), _jsx(Text, { children: " " }), _jsxs(Text, { color: statusColors[agent.status] || "white", children: ["[", agent.status, "]"] }), _jsx(Text, { children: " " }), _jsx(Text, { color: "gray", children: agent.task.length > 50
                                ? agent.task.slice(0, 50) + "..."
                                : agent.task }), _jsx(Text, { children: " " }), _jsxs(Text, { color: "gray", dimColor: true, children: [agent.decisions.length, "d"] })] }, agent.id));
            })] }));
}
