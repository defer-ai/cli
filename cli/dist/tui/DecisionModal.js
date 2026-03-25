import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text, useInput, useStdout } from "ink";
import { Header } from "./Banner.js";
/** Truncate text to fit width, adding ... if needed */
function truncate(text, maxLen) {
    if (text.length <= maxLen)
        return text;
    return text.slice(0, maxLen - 1) + "…";
}
export function DecisionModal({ agent, onAnswer, onDone, onAsk, onRevise, rows, }) {
    const { stdout } = useStdout();
    const cols = stdout?.columns || 80;
    const [cursorIdx, setCursorIdx] = useState(0);
    const [selectedOption, setSelectedOption] = useState(0);
    const [mode, setMode] = useState("browse");
    const [textValue, setTextValue] = useState("");
    const [aiResponse, setAiResponse] = useState("");
    const decisions = agent.decisions;
    const pending = decisions.filter((d) => d.answer === null);
    const allDone = pending.length === 0 && decisions.length > 0;
    const current = decisions[cursorIdx] || null;
    const isPending = current?.answer === null;
    const firstPendingIdx = decisions.findIndex((d) => d.answer === null);
    useEffect(() => {
        if (firstPendingIdx >= 0)
            setCursorIdx(firstPendingIdx);
    }, []); // eslint-disable-line react-hooks/exhaustive-deps
    useEffect(() => {
        if (agent.pendingIndex >= 0 && mode === "browse") {
            setCursorIdx(agent.pendingIndex);
        }
    }, [agent.pendingIndex]); // eslint-disable-line react-hooks/exhaustive-deps
    useEffect(() => {
        setSelectedOption(0);
    }, [cursorIdx]);
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
        if (key.escape) {
            onDone();
            return;
        }
        if (input === "j" || key.downArrow) {
            setCursorIdx((i) => Math.min(i + 1, decisions.length - 1));
        }
        if (input === "k" || key.upArrow) {
            setCursorIdx((i) => Math.max(i - 1, 0));
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
    // All done
    if (allDone) {
        return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 2, paddingY: 1, children: [_jsx(Header, { model: "" }), _jsxs(Box, { marginTop: 1, flexDirection: "column", children: [_jsxs(Text, { color: "green", bold: true, children: ["All ", decisions.length, " decisions answered."] }), decisions.map((d) => (_jsxs(Box, { children: [_jsxs(Text, { color: d.delegated ? "magenta" : "green", children: [d.delegated ? "◆" : "✓", " "] }), _jsxs(Text, { color: "gray", children: [d.id, " "] }), _jsxs(Text, { color: "gray", children: [d.question, " \u2192 ", d.answer] })] }, d.id)))] }), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "Proceeding..." })] }));
    }
    if (!current) {
        return (_jsx(Box, { flexDirection: "column", height: rows, padding: 2, children: _jsx(Text, { color: "gray", children: "Waiting for decisions..." }) }));
    }
    // Group categories
    const categories = new Map();
    decisions.forEach((d, i) => {
        const cat = d.category || "General";
        if (!categories.has(cat))
            categories.set(cat, []);
        categories.get(cat).push({ d, idx: i });
    });
    const answeredCount = decisions.filter((d) => d.answer !== null).length;
    const treeWidth = Math.floor(cols * 0.45);
    const maxQuestion = treeWidth - 12; // icon(3) + id(~8) + padding
    return (_jsxs(Box, { flexDirection: "column", height: rows, children: [_jsxs(Box, { paddingX: 2, paddingTop: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "Decisions" }), _jsxs(Text, { color: "gray", dimColor: true, children: ["  ", answeredCount, "/", decisions.length, " answered", pending.length > 0 ? `, ${pending.length} pending` : ""] })] }), _jsxs(Box, { flexGrow: 1, paddingX: 1, children: [_jsx(Box, { flexDirection: "column", width: treeWidth, overflow: "hidden", children: Array.from(categories.entries()).map(([cat, items]) => (_jsxs(Box, { flexDirection: "column", children: [_jsx(Box, { paddingLeft: 1, children: _jsx(Text, { color: "cyan", dimColor: true, children: cat }) }), items.map(({ d, idx }) => {
                                    const isCursor = idx === cursorIdx;
                                    const isAnswered = d.answer !== null;
                                    const icon = isAnswered
                                        ? d.delegated
                                            ? "◆"
                                            : "✓"
                                        : "○";
                                    const iconColor = isAnswered
                                        ? d.delegated
                                            ? "magenta"
                                            : "green"
                                        : "yellow";
                                    return (_jsxs(Box, { paddingLeft: 2, children: [_jsx(Text, { color: isCursor ? "cyan" : "gray", children: isCursor ? ">" : " " }), _jsx(Text, { color: iconColor, children: icon }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", d.id] }), _jsxs(Text, { color: isCursor ? "white" : "gray", children: [" ", truncate(d.question, maxQuestion)] })] }, d.id));
                                })] }, cat))) }), _jsxs(Box, { flexDirection: "column", flexGrow: 1, paddingLeft: 1, borderStyle: "single", borderColor: "gray", borderLeft: true, borderRight: false, borderTop: false, borderBottom: false, paddingX: 1, children: [_jsx(Text, { color: "cyan", dimColor: true, children: current.id }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { bold: true, wrap: "wrap", children: current.question }) }), current.context ? (_jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", italic: true, wrap: "wrap", children: current.context }) })) : null, _jsx(Box, { marginTop: 1, children: isPending ? (_jsx(Text, { color: "yellow", children: "\u25CB pending" })) : (_jsxs(Text, { color: current.delegated ? "magenta" : "green", children: [current.delegated ? "◆ delegated: " : "✓ ", current.answer] })) }), mode === "answer" && current.options.length > 0 ? (_jsx(Box, { flexDirection: "column", marginTop: 1, children: current.options.map((opt, i) => {
                                    const isSel = i === selectedOption;
                                    const isCfm = opt.label
                                        .toLowerCase()
                                        .includes("choose for me");
                                    return (_jsxs(Box, { children: [_jsx(Text, { color: isSel ? "cyan" : "gray", children: isSel ? " > " : "   " }), _jsxs(Text, { color: isSel ? "cyan" : isCfm ? "magenta" : "white", bold: isSel, children: [opt.key, ") ", opt.label] })] }, opt.key));
                                }) })) : null, mode === "browse" && current.options.length > 0 ? (_jsx(Box, { flexDirection: "column", marginTop: 1, children: current.options.map((o) => (_jsxs(Text, { color: "gray", dimColor: true, children: ["  ", o.key, ") ", o.label] }, o.key))) })) : null, (mode === "text" || mode === "change" || mode === "ask") ? (_jsxs(Box, { marginTop: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: mode === "ask"
                                            ? "Ask about this decision:"
                                            : mode === "change"
                                                ? "New answer:"
                                                : "Custom answer:" }), _jsxs(Box, { children: [_jsxs(Text, { color: "yellow", children: [">", " "] }), _jsx(Text, { children: textValue }), _jsx(Text, { color: "gray", children: "|" })] })] })) : null, mode === "ask" && aiResponse ? (_jsx(Box, { marginTop: 1, flexDirection: "column", children: _jsx(Text, { wrap: "wrap", color: "gray", children: aiResponse.length > 400
                                        ? aiResponse.slice(-400)
                                        : aiResponse }) })) : null] })] }), _jsx(Box, { paddingX: 2, children: _jsx(Text, { color: "gray", dimColor: true, children: mode === "browse"
                        ? `↑↓:navigate${isPending ? "  enter:answer" : "  c:change"}  a:ask  esc:back`
                        : mode === "answer"
                            ? "↑↓:pick  enter:confirm  t:custom  esc:back"
                            : "enter:submit  esc:cancel" }) })] }));
}
