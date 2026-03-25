import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { Box, Text, useInput } from "ink";
export function InputBar({ options, onSubmit, onCancel }) {
    const [mode, setMode] = useState(options && options.length > 0 ? "select" : "text");
    const [selectedIdx, setSelectedIdx] = useState(0);
    const [textValue, setTextValue] = useState("");
    useInput((input, key) => {
        if (key.escape) {
            if (mode === "text" && options && options.length > 0) {
                setMode("select");
                setTextValue("");
                return;
            }
            onCancel();
            return;
        }
        if (mode === "select") {
            if (input === "j" || key.downArrow) {
                setSelectedIdx((i) => Math.min(i + 1, (options?.length || 1) - 1));
                return;
            }
            if (input === "k" || key.upArrow) {
                setSelectedIdx((i) => Math.max(i - 1, 0));
                return;
            }
            if (key.return) {
                const opt = options?.[selectedIdx];
                if (opt) {
                    onSubmit(opt.value);
                }
                return;
            }
            // Any other key switches to text mode
            if (input === "t" || input === "/") {
                setMode("text");
                return;
            }
        }
        if (mode === "text") {
            if (key.return) {
                onSubmit(textValue);
                return;
            }
            if (key.backspace || key.delete) {
                setTextValue((v) => v.slice(0, -1));
                return;
            }
            if (input && !key.ctrl && !key.meta) {
                setTextValue((v) => v + input);
            }
        }
    });
    if (mode === "select" && options && options.length > 0) {
        return (_jsxs(Box, { borderStyle: "single", borderColor: "yellow", flexDirection: "column", paddingX: 1, children: [options.map((opt, i) => (_jsxs(Box, { children: [_jsx(Text, { color: i === selectedIdx ? "cyan" : "white", children: i === selectedIdx ? "> " : "  " }), _jsx(Text, { color: i === selectedIdx ? "cyan" : "white", children: opt.label })] }, i))), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", dimColor: true, children: "j/k:move enter:select t:type custom esc:cancel" }) })] }));
    }
    return (_jsxs(Box, { borderStyle: "single", borderColor: "yellow", paddingX: 1, children: [_jsx(Text, { color: "yellow", children: "> " }), _jsx(Text, { children: textValue }), _jsx(Text, { color: "gray", children: "|" }), _jsx(Box, { flexGrow: 1 }), _jsxs(Text, { color: "gray", dimColor: true, children: ["enter:send esc:", options ? "back" : "cancel"] })] }));
}
