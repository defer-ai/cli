import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text } from "ink";
import { execSync } from "node:child_process";
export function GitTab() {
    const [gitInfo, setGitInfo] = useState({
        isRepo: false,
        branch: "",
        recentCommits: [],
        uncommittedFiles: [],
    });
    useEffect(() => {
        try {
            execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });
            const branch = execSync("git branch --show-current", {
                encoding: "utf-8",
            }).trim();
            let recentCommits = [];
            try {
                recentCommits = execSync("git log --oneline -10", {
                    encoding: "utf-8",
                })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch {
                // no commits yet
            }
            let uncommittedFiles = [];
            try {
                uncommittedFiles = execSync("git status --short", {
                    encoding: "utf-8",
                })
                    .trim()
                    .split("\n")
                    .filter(Boolean);
            }
            catch {
                // ignore
            }
            setGitInfo({ isRepo: true, branch, recentCommits, uncommittedFiles });
        }
        catch {
            setGitInfo({
                isRepo: false,
                branch: "",
                recentCommits: [],
                uncommittedFiles: [],
            });
        }
    }, []);
    if (!gitInfo.isRepo) {
        return (_jsx(Box, { padding: 1, children: _jsx(Text, { color: "gray", children: "Not a git repository." }) }));
    }
    return (_jsxs(Box, { padding: 1, flexDirection: "column", children: [_jsxs(Box, { marginBottom: 1, children: [_jsxs(Text, { color: "cyan", bold: true, children: ["Branch:", " "] }), _jsx(Text, { children: gitInfo.branch })] }), gitInfo.uncommittedFiles.length > 0 && (_jsxs(Box, { flexDirection: "column", marginBottom: 1, children: [_jsxs(Text, { color: "yellow", bold: true, children: ["Uncommitted (", gitInfo.uncommittedFiles.length, ")"] }), gitInfo.uncommittedFiles.slice(0, 10).map((f, i) => (_jsx(Box, { paddingLeft: 1, children: _jsx(Text, { color: "gray", children: f }) }, i))), gitInfo.uncommittedFiles.length > 10 && (_jsx(Box, { paddingLeft: 1, children: _jsxs(Text, { color: "gray", dimColor: true, children: ["...and ", gitInfo.uncommittedFiles.length - 10, " more"] }) }))] })), _jsxs(Box, { flexDirection: "column", children: [_jsx(Text, { color: "cyan", bold: true, children: "Recent commits" }), gitInfo.recentCommits.length === 0 ? (_jsx(Box, { paddingLeft: 1, children: _jsx(Text, { color: "gray", children: "No commits yet." }) })) : (gitInfo.recentCommits.map((c, i) => (_jsx(Box, { paddingLeft: 1, children: _jsx(Text, { color: "gray", children: c }) }, i))))] })] }));
}
