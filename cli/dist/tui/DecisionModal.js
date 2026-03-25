import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import { MiniMascot } from "./Mascot.js";
export function DecisionModal({ agent, onAnswer, onDone, onAsk, onRevise, rows, }) {
    const { stdout } = useStdout();
    const cols = stdout?.columns || 80;
    const [selectedOption, setSelectedOption] = useState(0);
    const [mode, setMode] = useState("browse");
    const [textValue, setTextValue] = useState("");
    const [aiResponse, setAiResponse] = useState("");
    const decisions = agent.decisions;
    const pending = decisions.filter((d) => d.answer === null);
    const allDone = pending.length === 0 && decisions.length > 0;
    const currentIdx = agent.pendingIndex >= 0
        ? agent.pendingIndex
        : pending.length > 0
            ? decisions.indexOf(pending[0])
            : 0;
    const current = decisions[currentIdx] || null;
    const isPending = current?.answer === null;
    const answeredCount = decisions.filter((d) => d.answer !== null).length;
    useEffect(() => {
        setSelectedOption(0);
        setMode("browse");
        setTextValue("");
    }, [agent.pendingIndex]);
    useEffect(() => {
        if (allDone) {
            const timer = setTimeout(onDone, 2500);
            return () => clearTimeout(timer);
        }
    }, [allDone, onDone]);
    useEffect(() => {
        if (agent.currentOutput && mode === "ask") {
            setAiResponse(agent.currentOutput);
        }
    }, [agent.currentOutput, mode]);
    useInput((input, key) => {
        if (allDone) {
            onDone();
            return;
        }
        // Text modes
        if (mode === "text" || mode === "ask" || mode === "change") {
            if (key.escape) {
                setMode("browse");
                setTextValue("");
                setAiResponse("");
                return;
            }
            if (key.return && textValue.trim()) {
                if (mode === "ask" && current) {
                    onAsk(current.id, textValue.trim());
                    setAiResponse("Thinking...");
                    setTextValue("");
                }
                else if (mode === "change" && current) {
                    onRevise(current.id, textValue.trim());
                    setMode("browse");
                    setTextValue("");
                    setAiResponse("");
                }
                else if (mode === "text") {
                    onAnswer(textValue.trim());
                    setMode("browse");
                    setTextValue("");
                }
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
        // Answer mode
        if (mode === "answer" && current) {
            if (key.escape) {
                setMode("browse");
                return;
            }
            if (input === "j" || key.downArrow) {
                setSelectedOption((i) => Math.min(i + 1, (current.options.length || 1) - 1));
                return;
            }
            if (input === "k" || key.upArrow) {
                setSelectedOption((i) => Math.max(i - 1, 0));
                return;
            }
            if (key.return && current.options[selectedOption]) {
                onAnswer(current.options[selectedOption].key);
                setMode("browse");
                setSelectedOption(0);
                return;
            }
            if (input === "t") {
                setMode("text");
                return;
            }
            return;
        }
        // Browse mode
        if (key.escape || key.tab) {
            onDone();
            return;
        }
        if (key.return && current) {
            if (isPending && current.options.length > 0) {
                setMode("answer");
                setSelectedOption(0);
            }
            else if (isPending) {
                setMode("text");
            }
        }
        if (input === "c" && current && !isPending) {
            setMode("change");
            setTextValue("");
        }
        if (input === "a" && current) {
            setMode("ask");
            setTextValue("");
            setAiResponse("");
        }
    });
    // All done summary
    if (allDone) {
        return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 3, paddingY: 1, children: [_jsxs(Box, { children: [_jsx(MiniMascot, { mood: "done" }), _jsxs(Text, { color: "green", bold: true, children: [" ", "All ", decisions.length, " decisions answered"] })] }), _jsx(Box, { marginTop: 1, flexDirection: "column", children: decisions.map((d) => (_jsxs(Box, { children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsxs(Text, { color: "gray", children: [d.question, " \u2192 ", d.answer] })] }, d.id))) }), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "Proceeding..." })] }));
    }
    if (!current) {
        return (_jsx(Box, { flexDirection: "column", height: rows, padding: 3, children: _jsx(Text, { color: "gray", children: "Waiting for decisions..." }) }));
    }
    return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 3, paddingY: 1, children: [_jsxs(Box, { marginBottom: 1, children: [_jsx(MiniMascot, { mood: isPending ? "asking" : "answering" }), _jsxs(Text, { color: "cyan", bold: true, children: ["  ", answeredCount + (isPending ? 0 : 1), "/", decisions.length] }), _jsxs(Text, { color: "gray", dimColor: true, children: ["  ", current.category] }), _jsxs(Text, { color: "gray", dimColor: true, children: ["  ", current.id] })] }), _jsx(Box, { marginBottom: 1, children: _jsx(Text, { bold: true, wrap: "wrap", children: current.question }) }), current.context ? (_jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", italic: true, wrap: "wrap", children: current.context }) })) : null, !isPending ? (_jsx(Box, { marginBottom: 1, children: _jsxs(Text, { color: current.delegated ? "magenta" : "green", children: [current.delegated ? "◆ delegated: " : "✓ ", current.answer] }) })) : null, (mode === "browse" || mode === "answer") &&
                current.options.length > 0 ? (_jsx(Box, { flexDirection: "column", marginBottom: 1, children: current.options.map((opt, i) => {
                    const isSel = mode === "answer" && i === selectedOption;
                    const isCfm = opt.label.toLowerCase().includes("choose for me");
                    return (_jsxs(Box, { paddingLeft: 1, children: [_jsxs(Text, { color: isSel ? "cyan" : "gray", children: [isSel ? " >" : "  ", " "] }), _jsxs(Text, { color: isSel
                                    ? "cyan"
                                    : mode === "answer"
                                        ? isCfm
                                            ? "magenta"
                                            : "white"
                                        : "gray", bold: isSel, dimColor: mode === "browse", children: [opt.key, ") ", opt.label] })] }, opt.key));
                }) })) : null, (mode === "text" || mode === "change" || mode === "ask") ? (_jsxs(Box, { marginBottom: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: mode === "ask"
                            ? "Ask about this decision:"
                            : mode === "change"
                                ? "New answer:"
                                : "Custom answer:" }), _jsxs(Box, { children: [_jsxs(Text, { color: "yellow", children: [">", " "] }), _jsx(Text, { children: textValue }), _jsx(Text, { color: "gray", children: "|" })] })] })) : null, mode === "ask" && aiResponse ? (_jsx(Box, { marginBottom: 1, children: _jsx(Text, { wrap: "wrap", color: "gray", children: aiResponse.length > 500
                        ? aiResponse.slice(-500)
                        : aiResponse }) })) : null, answeredCount > 0 && isPending ? (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Recent:" }), decisions
                        .filter((d) => d.answer !== null)
                        .slice(-3)
                        .map((d) => (_jsxs(Box, { paddingLeft: 1, children: [_jsx(Text, { color: d.delegated ? "magenta" : "green", children: d.delegated ? "◆" : "✓" }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", d.id, " ", d.question, " \u2192 ", d.answer] })] }, d.id)))] })) : null, _jsx(Box, { flexGrow: 1 }), _jsx(Box, { children: _jsx(Text, { color: "gray", dimColor: true, children: mode === "browse"
                        ? isPending
                            ? "enter:answer  a:ask about  t:type custom  tab:back"
                            : "c:change  a:ask about  tab:back"
                        : mode === "answer"
                            ? "↑↓:pick  enter:confirm  t:custom  esc:back"
                            : "enter:submit  esc:cancel" }) })] }));
}
