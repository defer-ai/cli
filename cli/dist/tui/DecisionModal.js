import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { Box, Text, useInput } from "ink";
export function DecisionModal({ agent, onAnswer, onDone, rows }) {
    const [selectedOption, setSelectedOption] = useState(0);
    const [textMode, setTextMode] = useState(false);
    const [textValue, setTextValue] = useState("");
    const pending = agent.decisions.filter((d) => d.answer === null);
    const answered = agent.decisions.filter((d) => d.answer !== null);
    const current = agent.pendingIndex >= 0
        ? agent.decisions[agent.pendingIndex]
        : pending[0];
    // All done
    if (!current || pending.length === 0) {
        return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 2, paddingY: 1, children: [_jsxs(Box, { flexDirection: "column", flexGrow: 1, children: [_jsx(Text, { color: "green", bold: true, children: "All decisions answered." }), _jsx(Box, { marginTop: 1, flexDirection: "column", children: agent.decisions.map((d) => (_jsxs(Box, { children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsx(Text, { children: d.delegated ? `delegated: ${d.answer}` : d.answer })] }, d.id))) })] }), _jsx(Box, { children: _jsx(Text, { color: "gray", dimColor: true, children: "Proceeding..." }) })] }));
    }
    const totalCount = agent.decisions.length;
    const answeredCount = answered.length;
    const currentNum = answeredCount + 1;
    useInput((input, key) => {
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
                setSelectedOption(0);
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
            setSelectedOption((i) => Math.min(i + 1, current.options.length - 1));
        }
        if (input === "k" || key.upArrow) {
            setSelectedOption((i) => Math.max(i - 1, 0));
        }
        if (key.return && current.options[selectedOption]) {
            onAnswer(current.options[selectedOption].key);
            setSelectedOption(0);
        }
        if (input === "t") {
            setTextMode(true);
        }
    });
    return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 2, paddingY: 1, children: [_jsxs(Box, { marginBottom: 1, children: [_jsxs(Text, { color: "cyan", bold: true, children: [currentNum, "/", totalCount] }), _jsx(Text, { color: "gray", children: " | " }), _jsx(Text, { color: "gray", children: current.category }), _jsx(Text, { color: "gray", children: " | " }), _jsx(Text, { color: "cyan", children: current.id })] }), _jsx(Box, { marginBottom: 1, children: _jsx(Text, { bold: true, wrap: "wrap", children: current.question }) }), current.context && (_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", italic: true, wrap: "wrap", children: current.context }) })), !textMode && current.options.length > 0 && (_jsx(Box, { flexDirection: "column", marginBottom: 1, children: current.options.map((opt, i) => {
                    const isSelected = i === selectedOption;
                    const isChooseForMe = opt.label
                        .toLowerCase()
                        .includes("choose for me");
                    return (_jsxs(Box, { children: [_jsx(Text, { color: isSelected ? "cyan" : "gray", children: isSelected ? "  > " : "    " }), _jsxs(Text, { color: isSelected
                                    ? "cyan"
                                    : isChooseForMe
                                        ? "magenta"
                                        : "white", bold: isSelected, children: [opt.key, ") ", opt.label] })] }, opt.key));
                }) })), textMode && (_jsxs(Box, { marginBottom: 1, children: [_jsx(Text, { color: "yellow", children: "> " }), _jsx(Text, { children: textValue }), _jsx(Text, { color: "gray", children: "|" })] })), answered.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Answered:" }), answered.slice(-5).map((d) => (_jsxs(Box, { paddingLeft: 2, children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " ", d.question, " "] }), _jsx(Text, { color: "green", children: d.answer })] }, d.id))), answered.length > 5 && (_jsx(Box, { paddingLeft: 2, children: _jsxs(Text, { color: "gray", dimColor: true, children: ["...and ", answered.length - 5, " more"] }) }))] })), _jsx(Box, { flexGrow: 1 }), _jsx(Box, { children: _jsx(Text, { color: "gray", dimColor: true, children: textMode
                        ? "enter:submit  esc:back to options"
                        : "↑/↓:navigate  enter:select  t:type custom" }) })] }));
}
