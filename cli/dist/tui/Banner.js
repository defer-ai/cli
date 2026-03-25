import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
// Unique mascot: a small diamond-eyed listener with a thought bubble
// Represents "I'm here to listen and ask, not decide"
const MASCOT = [
    `        ○`,
    `       ╱│╲`,
    `      ╱ │ ╲`,
    `     ◇  │  ◇`,
    `      ╲ │ ╱`,
    `       ╲│╱`,
    `        │`,
];
const VERSION = "0.1.0";
export function Banner({ model, cwd }) {
    const dir = cwd.replace(process.env.HOME || "", "~");
    return (_jsx(Box, { flexDirection: "column", paddingX: 2, paddingTop: 1, children: _jsxs(Box, { flexDirection: "row", children: [_jsx(Box, { flexDirection: "column", marginRight: 3, children: MASCOT.map((line, i) => (_jsx(Text, { color: "cyan", children: line }, i))) }), _jsxs(Box, { flexDirection: "column", paddingTop: 1, children: [_jsxs(Text, { bold: true, children: [_jsx(Text, { color: "cyan", children: "defer" }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", "v", VERSION] })] }), _jsx(Text, { color: "gray", dimColor: true, children: "Zero-autonomy AI" }), _jsxs(Box, { marginTop: 1, children: [_jsxs(Text, { color: "gray", dimColor: true, children: ["model", " "] }), _jsx(Text, { color: "white", children: model })] }), _jsx(Box, { children: _jsxs(Text, { color: "gray", dimColor: true, children: ["cwd   ", dir] }) }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "/help for commands, tab to switch views" }) })] })] }) }));
}
/** Compact header shown at the top of every view */
export function Header({ model }) {
    return (_jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", bold: true, children: "defer" }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", "v", VERSION, " | ", model] })] }));
}
