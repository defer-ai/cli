import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
export function DecisionsTab({ agent }) {
    const [selectedIdx, setSelectedIdx] = useState(0);
    const [showOutput, setShowOutput] = useState(false);
    // Sync selectedIdx with the current pending decision
    useEffect(() => {
        if (agent && agent.pendingIndex >= 0) {
            setSelectedIdx(agent.pendingIndex);
        }
    }, [agent?.pendingIndex]);
    useInput((input, key) => {
        if (!agent)
            return;
        const max = Math.max(agent.decisions.length - 1, 0);
        if (input === "j" || key.downArrow) {
            setSelectedIdx((i) => Math.min(i + 1, max));
        }
        if (input === "k" || key.upArrow) {
            setSelectedIdx((i) => Math.max(i - 1, 0));
        }
        if (input === "o") {
            setShowOutput((v) => !v);
        }
    });
    if (!agent) {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "No agent running." }) }));
    }
    if (agent.decisions.length === 0 && agent.status === "thinking") {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "cyan", children: "Decomposing task..." }) }));
    }
    if (agent.decisions.length === 0) {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "No decisions yet." }) }));
    }
    if (showOutput) {
        return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: "press o to go back" }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { wrap: "wrap", children: agent.currentOutput || "(no output yet)" }) })] }));
    }
    // Group by category
    const categories = new Map();
    agent.decisions.forEach((d, i) => {
        const cat = d.category || "General";
        if (!categories.has(cat))
            categories.set(cat, []);
        categories.get(cat).push({ decision: d, globalIdx: i });
    });
    const selected = agent.decisions[selectedIdx];
    return (_jsxs(Box, { padding: 1, flexDirection: "row", children: [_jsxs(Box, { flexDirection: "column", width: "55%", children: [Array.from(categories.entries()).map(([category, items]) => (_jsxs(Box, { flexDirection: "column", marginBottom: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: category }), items.map(({ decision: d, globalIdx: idx }) => {
                                const isSelected = idx === selectedIdx;
                                const isCurrent = idx === agent.pendingIndex;
                                const isPending = d.answer === null;
                                const isDelegated = d.delegated;
                                return (_jsxs(Box, { paddingLeft: 1, children: [_jsxs(Text, { color: isSelected ? "cyan" : "gray", children: [isCurrent ? ">" : isSelected ? ">" : " ", " "] }), _jsxs(Text, { color: "gray", dimColor: true, children: [d.id, " "] }), _jsxs(Text, { color: isSelected ? "white" : "gray", children: [d.question, " "] }), _jsx(Text, { color: isPending
                                                ? "yellow"
                                                : isDelegated
                                                    ? "magenta"
                                                    : "green", bold: isCurrent, children: isPending
                                                ? "pending"
                                                : isDelegated
                                                    ? `delegated`
                                                    : d.answer })] }, d.id));
                            })] }, category))), agent.status === "asking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "yellow", children: "Waiting for your answers. Press i to respond." }) })), agent.status === "thinking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "cyan", children: "Thinking..." }) })), agent.status === "executing" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "blue", children: "Executing..." }) })), agent.status === "done" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "green", children: "Done." }) }))] }), selected && (_jsxs(Box, { flexDirection: "column", width: "45%", borderStyle: "single", borderColor: "gray", paddingX: 1, paddingY: 0, children: [_jsx(Text, { color: "cyan", bold: true, children: selected.id }), _jsx(Text, { color: "gray", children: selected.category }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { bold: true, wrap: "wrap", children: selected.question }) }), selected.context ? (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", italic: true, wrap: "wrap", children: selected.context }) })) : null, selected.options.length > 0 && (_jsx(Box, { flexDirection: "column", marginTop: 1, children: selected.options.map((o) => (_jsxs(Text, { color: "white", children: [o.key, ") ", o.label] }, o.key))) })), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: selected.answer === null
                                ? "yellow"
                                : selected.delegated
                                    ? "magenta"
                                    : "green", children: selected.answer === null
                                ? "pending"
                                : selected.delegated
                                    ? `delegated: ${selected.answer}`
                                    : selected.answer }) })] }))] }));
}
