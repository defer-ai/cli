import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import React, { useState, useMemo } from "react";
import { Box, Text, useInput } from "ink";
import { MiniMascot } from "./Mascot.js";
const LEVELS = [
    { key: "skip", label: "skip", color: "gray", description: "delegate everything" },
    { key: "low", label: "low", color: "blue", description: "only the key question" },
    { key: "medium", label: "medium", color: "yellow", description: "important decisions" },
    { key: "high", label: "high", color: "green", description: "ask me everything" },
    { key: "paranoid", label: "paranoid", color: "red", description: "deep dive, sub-questions" },
];
const LEVEL_BAR = {
    skip: "░░░░░",
    low: "█░░░░",
    medium: "██░░░",
    high: "████░",
    paranoid: "█████",
};
export function DomainPriority({ decisions, onComplete, rows }) {
    // Extract unique categories
    const categories = useMemo(() => {
        const cats = [];
        for (const d of decisions) {
            if (!cats.includes(d.category))
                cats.push(d.category);
        }
        return cats;
    }, [decisions]);
    const [cursorIdx, setCursorIdx] = useState(0);
    const [priorities, setPriorities] = useState(() => {
        const initial = {};
        for (const cat of categories) {
            initial[cat] = "medium";
        }
        return initial;
    });
    const currentCat = categories[cursorIdx];
    const currentLevel = priorities[currentCat];
    const currentLevelIdx = LEVELS.findIndex((l) => l.key === currentLevel);
    const decisionsInCat = decisions.filter((d) => d.category === currentCat);
    const [addingCategory, setAddingCategory] = useState(false);
    const [newCatValue, setNewCatValue] = useState("");
    useInput((input, key) => {
        // Adding a new category
        if (addingCategory) {
            if (key.escape) {
                setAddingCategory(false);
                setNewCatValue("");
                return;
            }
            if (key.return && newCatValue.trim()) {
                const name = newCatValue.trim();
                if (!categories.includes(name)) {
                    categories.push(name);
                    setPriorities((prev) => ({ ...prev, [name]: "high" }));
                    setCursorIdx(categories.length - 1);
                }
                setAddingCategory(false);
                setNewCatValue("");
                return;
            }
            if (key.backspace || key.delete) {
                setNewCatValue((v) => v.slice(0, -1));
                return;
            }
            if (input && !key.ctrl && !key.meta) {
                setNewCatValue((v) => v + input);
            }
            return;
        }
        // Navigate categories
        if (input === "j" || key.downArrow) {
            setCursorIdx((i) => Math.min(i + 1, categories.length - 1));
        }
        if (input === "k" || key.upArrow) {
            setCursorIdx((i) => Math.max(i - 1, 0));
        }
        // Adjust care level with left/right
        if ((input === "h" || key.leftArrow) && currentLevelIdx > 0) {
            setPriorities((prev) => ({
                ...prev,
                [currentCat]: LEVELS[currentLevelIdx - 1].key,
            }));
        }
        if ((input === "l" || key.rightArrow) &&
            currentLevelIdx < LEVELS.length - 1) {
            setPriorities((prev) => ({
                ...prev,
                [currentCat]: LEVELS[currentLevelIdx + 1].key,
            }));
        }
        // Add new category
        if (input === "n") {
            setAddingCategory(true);
            setNewCatValue("");
        }
        // Confirm
        if (key.return) {
            onComplete(priorities);
        }
        // Escape: confirm with current settings
        if (key.escape) {
            onComplete(priorities);
        }
    });
    return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 3, paddingY: 1, children: [_jsxs(Box, { marginBottom: 1, children: [_jsx(MiniMascot, { mood: "asking" }), _jsxs(Text, { color: "cyan", bold: true, children: ["  ", "How much do you care about each area?"] })] }), _jsx(Box, { marginBottom: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "This controls how many questions you get per domain. Use \u2190\u2192 to adjust, \u2191\u2193 to navigate, enter to confirm." }) }), _jsx(Box, { flexDirection: "column", marginBottom: 1, children: categories.map((cat, i) => {
                    const isCursor = i === cursorIdx;
                    const level = priorities[cat];
                    const levelInfo = LEVELS.find((l) => l.key === level);
                    const count = decisions.filter((d) => d.category === cat).length;
                    return (_jsxs(Box, { paddingLeft: 1, marginBottom: 0, children: [_jsxs(Text, { color: isCursor ? "cyan" : "gray", children: [isCursor ? ">" : " ", " "] }), _jsx(Text, { color: isCursor ? "white" : "gray", bold: isCursor, children: cat.padEnd(18) }), _jsx(Text, { color: levelInfo.color, children: LEVEL_BAR[level] }), _jsxs(Text, { color: levelInfo.color, children: [" ", levelInfo.label.padEnd(10)] }), _jsxs(Text, { color: "gray", dimColor: true, children: [count, " decision", count !== 1 ? "s" : ""] })] }, cat));
                }) }), _jsxs(Box, { flexDirection: "column", marginTop: 1, paddingX: 1, borderStyle: "single", borderColor: "gray", borderTop: true, borderBottom: false, borderLeft: false, borderRight: false, children: [_jsxs(Box, { marginTop: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: currentCat }), _jsxs(Text, { color: "gray", children: ["  ", LEVELS.find((l) => l.key === currentLevel)?.description] })] }), _jsxs(Box, { marginTop: 1, flexDirection: "column", children: [_jsx(Text, { color: "gray", dimColor: true, children: "Questions in this domain:" }), decisionsInCat.slice(0, 5).map((d) => (_jsx(Box, { paddingLeft: 1, children: _jsxs(Text, { color: "gray", dimColor: true, children: [currentLevel === "skip" ? "◆" : "○", " ", d.question] }) }, d.id))), decisionsInCat.length > 5 ? (_jsx(Box, { paddingLeft: 1, children: _jsxs(Text, { color: "gray", dimColor: true, children: ["...and ", decisionsInCat.length - 5, " more"] }) })) : null] })] }), addingCategory ? (_jsxs(Box, { marginTop: 1, paddingLeft: 2, children: [_jsx(Text, { color: "yellow", children: "new domain: " }), _jsx(Text, { children: newCatValue }), _jsx(Text, { color: "gray", children: "|" }), _jsxs(Text, { color: "gray", dimColor: true, children: ["  ", "enter:add  esc:cancel"] })] })) : null, _jsx(Box, { flexGrow: 1 }), _jsx(Box, { children: LEVELS.map((l, i) => (_jsxs(React.Fragment, { children: [_jsx(Text, { color: l.color, dimColor: l.key !== currentLevel, children: l.label }), i < LEVELS.length - 1 ? (_jsx(Text, { color: "gray", dimColor: true, children: "  " })) : null] }, l.key))) }), _jsx(Box, { children: _jsx(Text, { color: "gray", dimColor: true, children: "\u2190\u2192:adjust  \u2191\u2193:navigate  n:add domain  enter:confirm" }) })] }));
}
