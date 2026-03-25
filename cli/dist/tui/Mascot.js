import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useEffect } from "react";
import { Box, Text } from "ink";
// Each mood has multiple frames for animation
// Eyes are symmetric (both eyes always identical)
// Mouth is always the same centered block
const FRAMES = {
    // idle: slow blink (frame 1 = open, frame 2 = half, frame 3 = closed)
    idle: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██    ██         ██    ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▄▄ ██         ██ ▄▄ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██▄▄▄▄██         ██▄▄▄▄██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▄▄ ██         ██ ▄▄ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // thinking: snake chasing tail
    thinking: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▀▄ ██         ██ ▀▄ ██"],
            ["   ██ ▄▀ ██         ██ ▄▀ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▄▀ ██         ██ ▄▀ ██"],
            ["   ██ ▀▄ ██         ██ ▀▄ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▀▄ ██         ██ ▀▄ ██"],
            ["   ██ ▄▀ ██         ██ ▄▀ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▄▀ ██         ██ ▄▀ ██"],
            ["   ██ ▀▄ ██         ██ ▀▄ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // asking: question mark pulse
    asking: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▀▀ ██         ██ ▀▀ ██"],
            ["   ██ ▀  ██         ██ ▀  ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██    ██         ██    ██"],
            ["   ██ ▀  ██         ██ ▀  ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // answering: same as done
    answering: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ◆  ██         ██ ◆  ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██  ◆ ██         ██  ◆ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // executing: dot bouncing
    executing: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ▄▄ ██         ██ ▄▄ ██"],
            ["   ██    ██         ██    ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██    ██         ██    ██"],
            ["   ██ ▄▄ ██         ██ ▄▄ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // done: sparkle
    done: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██ ◆  ██         ██ ◆  ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██  ◆ ██         ██  ◆ ██"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
    // error: symmetric noise
    error: [
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██▓░▒███         ██▓░▒███"],
            ["   ██▒▓░███         ██▒▓░███"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██░▒▓███         ██░▒▓███"],
            ["   ██▓░▒███         ██▓░▒███"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
        [
            ["   ▄██████▄         ▄██████▄"],
            ["   ██▒░▓███         ██▒░▓███"],
            ["   ██░▓▒███         ██░▓▒███"],
            ["   ▀██████▀         ▀██████▀"],
            [""],
            ["            ████████"],
        ],
    ],
};
// Animation speeds per mood (ms)
const SPEEDS = {
    idle: 600,
    thinking: 150,
    asking: 500,
    answering: 350,
    executing: 300,
    done: 400,
    error: 100,
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
    const [frame, setFrame] = useState(0);
    const frames = FRAMES[mood];
    const speed = SPEEDS[mood];
    const label = LABELS[mood];
    useEffect(() => {
        setFrame(0);
        const interval = setInterval(() => {
            setFrame((f) => (f + 1) % frames.length);
        }, speed);
        return () => clearInterval(interval);
    }, [mood, frames.length, speed]);
    const currentFrame = frames[frame % frames.length];
    return (_jsxs(Box, { flexDirection: "column", children: [currentFrame.map((line, i) => (_jsx(Text, { color: "cyan", children: line[0] }, i))), label ? (_jsxs(Text, { color: "gray", dimColor: true, children: ["            ", label] })) : null] }));
}
// Inline mini face for headers
const MINI_FRAMES = {
    idle: ["[· ·]", "[- -]"],
    thinking: ["[◠ ◠]", "[◡ ◡]"],
    asking: ["[? ?]", "[  ?]"],
    answering: ["[◆ ◆]", "[◇ ◇]"],
    executing: ["[▪ ▪]", "[· ·]"],
    done: ["[◆ ◆]", "[◇ ◇]"],
    error: ["[░ ░]", "[▓ ▓]"],
};
export function MiniMascot({ mood }) {
    const [frame, setFrame] = useState(0);
    const frames = MINI_FRAMES[mood];
    const speed = SPEEDS[mood];
    useEffect(() => {
        setFrame(0);
        const interval = setInterval(() => {
            setFrame((f) => (f + 1) % frames.length);
        }, speed);
        return () => clearInterval(interval);
    }, [mood, frames.length, speed]);
    return _jsx(Text, { color: "cyan", children: frames[frame % frames.length] });
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
