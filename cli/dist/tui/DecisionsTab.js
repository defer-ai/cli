import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { Box, Text, useInput } from "ink";
export function DecisionsTab({ agent }) {
    const [selectedIdx, setSelectedIdx] = useState(0);
    const [showOutput, setShowOutput] = useState(false);
    useInput((input, key) => {
        if (!agent)
            return;
        const max = agent.decisions.length - 1;
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
    if (agent.decisions.length === 0 && !agent.currentOutput) {
        return (_jsx(Box, { padding: 1, flexDirection: "column", children: _jsx(Text, { color: "gray", children: "Decomposing task..." }) }));
    }
    if (showOutput) {
        return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: "press o to go back" }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { wrap: "wrap", children: agent.currentOutput || "(no output yet)" }) })] }));
    }
    // Group by category
    const categories = new Map();
    for (const d of agent.decisions) {
        const cat = d.category || "General";
        if (!categories.has(cat))
            categories.set(cat, []);
        categories.get(cat).push(d);
    }
    let globalIdx = 0;
    const selected = agent.decisions[selectedIdx];
    return (_jsxs(Box, { padding: 1, flexDirection: "row", children: [_jsxs(Box, { flexDirection: "column", width: "60%", children: [_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "j/k:navigate o:raw output i:respond" }) }), Array.from(categories.entries()).map(([category, decisions]) => (_jsxs(Box, { flexDirection: "column", marginBottom: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: category }), decisions.map((d) => {
                                const idx = globalIdx++;
                                const isSelected = idx === selectedIdx;
                                const isPending = d.answer === null;
                                const isDelegated = d.delegated;
                                return (_jsxs(Box, { paddingLeft: 1, children: [_jsxs(Text, { color: isSelected ? "cyan" : "gray", children: [isSelected ? ">" : " ", " "] }), _jsxs(Text, { color: "gray", dimColor: true, children: [d.id, " "] }), _jsxs(Text, { color: isSelected ? "white" : "gray", children: [d.question, " "] }), _jsx(Text, { color: isPending
                                                ? "yellow"
                                                : isDelegated
                                                    ? "magenta"
                                                    : "green", children: isPending
                                                ? "pending"
                                                : isDelegated
                                                    ? `delegated: ${d.answer}`
                                                    : d.answer })] }, d.id));
                            })] }, category))), agent.status === "asking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "yellow", children: "Waiting for your answers. Press i to respond." }) })), agent.status === "thinking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "cyan", children: "Thinking..." }) }))] }), selected && (_jsxs(Box, { flexDirection: "column", width: "40%", borderStyle: "single", borderColor: "gray", paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: selected.id }), _jsx(Text, { color: "gray", dimColor: true, children: selected.category }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { bold: true, children: selected.question }) }), selected.context ? (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", italic: true, children: selected.context }) })) : null, selected.options.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Options:" }), selected.options.map((o) => (_jsx(Box, { paddingLeft: 1, children: _jsxs(Text, { color: "white", children: [o.key, ") ", o.label] }) }, o.key)))] })), _jsxs(Box, { marginTop: 1, children: [_jsx(Text, { color: "gray", children: "Answer: " }), _jsx(Text, { color: selected.answer === null
                                    ? "yellow"
                                    : selected.delegated
                                        ? "magenta"
                                        : "green", children: selected.answer === null
                                    ? "pending"
                                    : selected.delegated
                                        ? `delegated: ${selected.answer}`
                                        : selected.answer })] }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: selected.date }) })] }))] }));
}
