import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import { DecisionSummary } from "./DecisionSummary.js";
export function DashboardOverlay({ agents, selectedId, onSelect, onClose, rows, }) {
    const [tab, setTab] = useState("decisions");
    const current = agents.find((a) => a.id === selectedId) || agents[0];
    useInput((input, key) => {
        if (key.escape || input === "d") {
            onClose();
            return;
        }
        if (key.tab) {
            setTab((prev) => prev === "decisions"
                ? "agents"
                : prev === "agents"
                    ? "git"
                    : "decisions");
        }
    });
    return (_jsxs(Box, { flexDirection: "column", height: rows, paddingX: 1, children: [_jsxs(Box, { marginBottom: 1, children: [["decisions", "agents", "git"].map((t) => (_jsxs(React.Fragment, { children: [_jsxs(Text, { color: tab === t ? "cyan" : "gray", bold: tab === t, children: ["[", t, "]"] }), _jsx(Text, { children: " " })] }, t))), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "tab:switch  esc:close" })] }), _jsxs(Box, { flexDirection: "column", flexGrow: 1, children: [tab === "decisions" && current && (_jsx(DecisionSummary, { decisions: current.decisions })), tab === "agents" && (_jsx(Box, { flexDirection: "column", paddingX: 1, children: agents.map((a) => {
                            const answered = a.decisions.filter((d) => d.answer !== null).length;
                            return (_jsxs(Box, { marginBottom: 1, flexDirection: "column", children: [_jsxs(Box, { children: [_jsx(Text, { color: a.id === selectedId ? "cyan" : "white", bold: a.id === selectedId, children: a.id }), _jsx(Text, { color: "gray", children: " | " }), _jsx(Text, { color: a.status === "thinking"
                                                    ? "cyan"
                                                    : a.status === "asking"
                                                        ? "yellow"
                                                        : a.status === "done"
                                                            ? "green"
                                                            : a.status === "error"
                                                                ? "red"
                                                                : "gray", children: a.status }), _jsxs(Text, { color: "gray", children: [" ", "| ", answered, "/", a.decisions.length, " decisions"] })] }), _jsx(Box, { paddingLeft: 2, children: _jsx(Text, { color: "gray", wrap: "wrap", children: a.task }) })] }, a.id));
                        }) })), tab === "git" && _jsx(GitInfo, {})] })] }));
}
function GitInfo() {
    const [info, setInfo] = React.useState(null);
    React.useEffect(() => {
        try {
            const { execSync } = require("node:child_process");
            execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });
            const branch = execSync("git branch --show-current", {
                encoding: "utf-8",
            }).trim();
            let commits = [];
            try {
                commits = execSync("git log --oneline -8", { encoding: "utf-8" })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch { }
            let dirty = [];
            try {
                dirty = execSync("git status --short", { encoding: "utf-8" })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch { }
            setInfo({ branch, commits, dirty });
        }
        catch {
            setInfo(null);
        }
    }, []);
    if (!info) {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "Not a git repository." }) }));
    }
    return (_jsxs(Box, { flexDirection: "column", paddingX: 1, children: [_jsx(Box, { children: _jsx(Text, { color: "cyan", bold: true, children: info.branch }) }), info.dirty.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsxs(Text, { color: "yellow", children: [info.dirty.length, " uncommitted"] }), info.dirty.slice(0, 5).map((f, i) => (_jsxs(Text, { color: "gray", children: ["  ", f] }, i)))] })), info.commits.length > 0 && (_jsxs(Box, { flexDirection: "column", marginTop: 1, children: [_jsx(Text, { color: "gray", dimColor: true, children: "Recent commits" }), info.commits.map((c, i) => (_jsxs(Text, { color: "gray", children: ["  ", c] }, i)))] }))] }));
}
