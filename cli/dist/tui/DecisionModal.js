import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
export function DecisionModal({ agent, onAnswer, onDone, rows }) {
    const [selectedOption, setSelectedOption] = useState(0);
    const [textMode, setTextMode] = useState(false);
    const [textValue, setTextValue] = useState("");
    const pending = agent.decisions.filter((d) => d.answer === null);
    const answered = agent.decisions.filter((d) => d.answer !== null);
    const current = agent.pendingIndex >= 0
        ? agent.decisions[agent.pendingIndex]
        : pending[0] || null;
    const allDone = pending.length === 0;
    const totalCount = agent.decisions.length;
    const answeredCount = answered.length;
    const currentNum = answeredCount + 1;
    // Reset selection when moving to next decision
    useEffect(() => {
        setSelectedOption(0);
        setTextMode(false);
        setTextValue("");
    }, [agent.pendingIndex]);
    // Auto-close when all done
    useEffect(() => {
        if (allDone && totalCount > 0) {
            const timer = setTimeout(onDone, 2000);
            return () => clearTimeout(timer);
        }
    }, [allDone, totalCount, onDone]);
    useInput((input, key) => {
        // All done state - any key closes
        if (allDone) {
            onDone();
            return;
        }
        if (!current)
            return;
        if (textMode) {
            if (key.escape) {
                setTextMode(false);
                setTextValue("");
                return;
            }
            if (key.return && textValue.trim()) {
                onAnswer(textValue.trim());
                setTextValue("");
                setTextMode(false);
                return;
            }
            if (key.backspace || key.delete) {
                setTextValue((v) => v.slice(0, -1));
                return;
            }
            if (input && !key.ctrl && !key.meta) {
                setTextValue((v) => v + input);
            }
            return;
        }
        // Option navigation
        if (input === "j" || key.downArrow) {
            setSelectedOption((i) => Math.min(i + 1, (current.options.length || 1) - 1));
        }
        if (input === "k" || key.upArrow) {
            setSelectedOption((i) => Math.max(i - 1, 0));
        }
        if (key.return && current.options[selectedOption]) {
            onAnswer(current.options[selectedOption].key);
        }
        if (input === "t") {
            setTextMode(true);
        }
        if (key.escape) {
            onDone();
        }
    });
    // All done view
    if (allDone) {
        return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 2, paddingY: 1, children: [_jsxs(Text, { color: "green", bold: true, children: ["All ", totalCount, " decisions answered."] }), _jsx(Box, { marginTop: 1, flexDirection: "column", children: agent.decisions.map((d) => (_jsxs(Box, { children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsxs(Text, { children: [d.question, " "] }), _jsxs(Text, { color: "gray", children: ["\u2192 ", d.delegated ? `delegated: ${d.answer}` : d.answer] })] }, d.id))) }), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "Proceeding with execution..." })] }));
    }
    // No current decision (shouldn't happen, but safe)
    if (!current) {
        return (_jsx(Box, { flexDirection: "column", height: rows, padding: 2, children: _jsx(Text, { color: "gray", children: "Waiting for decisions..." }) }));
    }
    return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 2, paddingY: 1, children: [_jsxs(Box, { marginBottom: 1, children: [_jsxs(Text, { color: "cyan", bold: true, children: [currentNum, "/", totalCount] }), _jsx(Text, { color: "gray", children: "  " }), _jsx(Text, { color: "gray", children: current.category }), _jsx(Text, { color: "gray", children: "  " }), _jsx(Text, { color: "cyan", dimColor: true, children: current.id })] }), _jsx(Box, { marginBottom: 1, children: _jsx(Text, { bold: true, wrap: "wrap", children: current.question }) }), current.context ? (_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", italic: true, wrap: "wrap", children: current.context }) })) : null, !textMode && current.options.length > 0 ? (_jsx(Box, { flexDirection: "column", marginBottom: 1, children: current.options.map((opt, i) => {
                    const isSelected = i === selectedOption;
                    const isChooseForMe = opt.label
                        .toLowerCase()
                        .includes("choose for me");
                    return (_jsxs(Box, { paddingLeft: 1, children: [_jsxs(Text, { color: isSelected ? "cyan" : "gray", children: [isSelected ? " >" : "  ", " "] }), _jsxs(Text, { color: isSelected
                                    ? "cyan"
                                    : isChooseForMe
                                        ? "magenta"
                                        : "white", bold: isSelected, children: [opt.key, ") ", opt.label] })] }, opt.key));
                }) })) : null, textMode ? (_jsxs(Box, { marginBottom: 1, paddingLeft: 2, children: [_jsxs(Text, { color: "yellow", children: [">", " "] }), _jsx(Text, { children: textValue }), _jsx(Text, { color: "gray", children: "|" })] })) : null, answered.length > 0 ? (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Answered:" }), answered.slice(-4).map((d) => (_jsxs(Box, { paddingLeft: 2, children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.question, " \u2192 ", d.answer] })] }, d.id))), answered.length > 4 ? (_jsx(Box, { paddingLeft: 2, children: _jsxs(Text, { color: "gray", dimColor: true, children: ["...and ", answered.length - 4, " more"] }) })) : null] })) : null, _jsx(Box, { flexGrow: 1 }), _jsx(Box, { children: _jsx(Text, { color: "gray", dimColor: true, children: textMode
                        ? "enter:submit  esc:back"
                        : "↑/↓:navigate  enter:select  t:type custom  esc:close" }) })] }));
}
