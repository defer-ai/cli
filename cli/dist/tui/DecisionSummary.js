import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
export function DecisionSummary({ decisions }) {
    if (decisions.length === 0)
        return null;
    // Group by category
    const categories = new Map();
    for (const d of decisions) {
        const cat = d.category || "General";
        if (!categories.has(cat))
            categories.set(cat, []);
        categories.get(cat).push(d);
    }
    return (_jsx(Box, { flexDirection: "column", paddingX: 1, marginY: 1, children: Array.from(categories.entries()).map(([cat, items]) => (_jsxs(Box, { flexDirection: "column", children: [_jsx(Text, { color: "cyan", dimColor: true, children: cat }), items.map((d) => (_jsxs(Box, { paddingLeft: 2, children: [_jsxs(Text, { color: d.answer === null ? "yellow" : d.delegated ? "magenta" : "green", children: [d.answer === null ? "○" : d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsx(Text, { children: d.answer === null
                                ? d.question
                                : d.delegated
                                    ? `${d.question} → delegated`
                                    : `${d.question} → ${d.answer}` })] }, d.id)))] }, cat))) }));
}
