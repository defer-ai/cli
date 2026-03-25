import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
const FACES = {
    idle: { eyes: "-   -", mouth: "  ~  " },
    thinking: { eyes: "◠   ◠", mouth: "  o  " },
    asking: { eyes: "◉   ◉", mouth: "  ?  " },
    answering: { eyes: "◠   ◠", mouth: "  ◡  ", blush: true },
    executing: { eyes: "▪   ▪", mouth: " ─── " },
    done: { eyes: "^   ^", mouth: "  ◡  ", blush: true },
    error: { eyes: "x   x", mouth: "  _  " },
};
const LABELS = {
    idle: "",
    thinking: "thinking...",
    asking: "your turn",
    answering: "got it!",
    executing: "working...",
    done: "all done",
    error: "uh oh",
};
export function Mascot({ mood }) {
    const face = FACES[mood];
    const label = LABELS[mood];
    return (_jsxs(Box, { flexDirection: "column", children: [_jsx(Text, { color: "cyan", children: "┌───────┐" }), _jsxs(Text, { children: [_jsx(Text, { color: "cyan", children: "│ " }), _jsx(Text, { color: "white", children: face.eyes }), _jsx(Text, { color: "cyan", children: " │" })] }), _jsxs(Text, { children: [_jsx(Text, { color: "cyan", children: "│ " }), _jsx(Text, { color: face.blush ? "magenta" : "white", children: face.mouth }), _jsx(Text, { color: "cyan", children: " │" })] }), _jsx(Text, { color: "cyan", children: "└───────┘" }), label ? (_jsxs(Text, { color: "gray", dimColor: true, children: ["  ", label] })) : null] }));
}
/** Map agent status to mascot mood */
export function statusToMood(status, phase) {
    switch (status) {
        case "thinking":
            return "thinking";
        case "asking":
            return "asking";
        case "executing":
            return "executing";
        case "done":
            return "done";
        case "error":
            return "error";
        default:
            return "idle";
    }
}
