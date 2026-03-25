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
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "Waiting for AI to decompose task..." }) }));
    }
    if (showOutput) {
        return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: "[o: back to decisions]" }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { wrap: "wrap", children: agent.currentOutput || "(no output yet)" }) })] }));
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
    return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "j/k:navigate o:output i:respond" }) }), Array.from(categories.entries()).map(([category, decisions]) => (_jsxs(Box, { flexDirection: "column", marginBottom: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: category }), decisions.map((d) => {
                        const idx = globalIdx++;
                        const isSelected = idx === selectedIdx;
                        const isDelegated = d.answer.startsWith("DELEGATED");
                        const isPending = d.answer === "(pending)";
                        return (_jsxs(Box, { paddingLeft: 1, children: [_jsx(Text, { color: isSelected ? "cyan" : "white", children: isSelected ? "> " : "  " }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsxs(Text, { children: [d.question, " "] }), _jsx(Text, { color: isPending
                                        ? "yellow"
                                        : isDelegated
                                            ? "magenta"
                                            : "green", children: d.answer })] }, d.id));
                    })] }, category))), agent.status === "asking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "yellow", children: "AI is waiting for your answers. Press i to respond." }) })), agent.status === "thinking" && (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "cyan", children: "AI is thinking..." }) }))] }));
}
