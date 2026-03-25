import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
// Pixel-art faces using block characters
// Each face is 16 chars wide, 7 lines tall
const FACES = {
    idle: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ▄▄▄  ▄▄▄   █ ",
        " █              █ ",
        " █    ══════    █ ",
        " █              █ ",
        "  ▀████████████▀  ",
    ],
    thinking: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ████  ████  █ ",
        " █              █ ",
        " █      ██      █ ",
        " █              █ ",
        "  ▀████████████▀  ",
    ],
    asking: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ████  ████  █ ",
        " █  ████  ████  █ ",
        " █     ▄▄▄▄    █ ",
        " █     ▀▀      █ ",
        "  ▀████████████▀  ",
    ],
    answering: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ▀▀▀  ▀▀▀   █ ",
        " █              █ ",
        " █   ▄██████▄   █ ",
        " █              █ ",
        "  ▀████████████▀  ",
    ],
    executing: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ▀██▀ ▀██▀  █ ",
        " █              █ ",
        " █   ████████   █ ",
        " █              █ ",
        "  ▀████████████▀  ",
    ],
    done: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ▀▀▀  ▀▀▀   █ ",
        " █              █ ",
        " █  ▄████████▄  █ ",
        " █  ▀▀▀▀▀▀▀▀▀▀  █ ",
        "  ▀████████████▀  ",
    ],
    error: [
        "  ▄████████████▄  ",
        " █              █ ",
        " █  ▀██▄ ▄██▀  █ ",
        " █   ▄██▄██▄   █ ",
        " █              █ ",
        " █   ▄▀▀▀▀▄    █ ",
        "  ▀████████████▀  ",
    ],
};
const LABELS = {
    idle: "",
    thinking: "thinking...",
    asking: "your turn",
    answering: "got it",
    executing: "working...",
    done: "all done",
    error: "uh oh",
};
export function Mascot({ mood }) {
    const face = FACES[mood];
    const label = LABELS[mood];
    return (_jsxs(Box, { flexDirection: "column", children: [face.map((line, i) => (_jsx(Text, { color: "cyan", children: line }, i))), label ? (_jsx(Box, { justifyContent: "center", children: _jsx(Text, { color: "gray", dimColor: true, children: label }) })) : null] }));
}
// Inline mini face for the header (single line)
export function MiniMascot({ mood }) {
    const mini = {
        idle: "[· ·]",
        thinking: "[◠ ◠]",
        asking: "[◉ ◉]",
        answering: "[◠‿◠]",
        executing: "[▪ ▪]",
        done: "[^ ^]",
        error: "[x x]",
    };
    return _jsx(Text, { color: "cyan", children: mini[mood] });
}
export function statusToMood(status, _phase) {
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
