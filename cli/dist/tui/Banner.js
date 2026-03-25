import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
import { Mascot } from "./Mascot.js";
const VERSION = "0.1.0";
export function Banner({ model, cwd, mood }) {
    const dir = cwd.replace(process.env.HOME || "", "~");
    return (_jsx(Box, { flexDirection: "column", paddingX: 1, paddingTop: 1, children: _jsxs(Box, { flexDirection: "row", children: [_jsx(Box, { marginRight: 2, children: _jsx(Mascot, { mood: mood }) }), _jsxs(Box, { flexDirection: "column", paddingTop: 1, children: [_jsxs(Text, { bold: true, children: [_jsx(Text, { color: "cyan", children: "defer" }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", "v", VERSION] })] }), _jsxs(Box, { marginTop: 1, children: [_jsxs(Text, { color: "gray", dimColor: true, children: ["model", " "] }), _jsx(Text, { color: "white", children: model })] }), _jsx(Box, { children: _jsxs(Text, { color: "gray", dimColor: true, children: ["cwd   ", dir] }) }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "/help for commands, tab to switch views" }) })] })] }) }));
}
/** Compact header with mini mascot face, always visible */
export function Header({ model, mood }) {
    const face = {
        idle: "( - - )",
        thinking: "( ◠ ◠ )",
        asking: "( ◉ ◉ )",
        answering: "( ◠‿◠ )",
        executing: "( ▪ ▪ )",
        done: "( ^ ^ )",
        error: "( x x )",
    }[mood];
    return (_jsxs(Box, { paddingX: 1, children: [_jsx(Text, { color: "cyan", children: face }), _jsxs(Text, { color: "cyan", bold: true, children: [" ", "defer"] }), _jsxs(Text, { color: "gray", dimColor: true, children: [" ", "v", VERSION, " | ", model] })] }));
}
