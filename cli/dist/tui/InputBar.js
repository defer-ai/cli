import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { Box, Text, useInput } from "ink";
export function InputBar({ onSubmit, onCancel }) {
    const [value, setValue] = useState("");
    useInput((input, key) => {
        if (key.escape) {
            onCancel();
            return;
        }
        if (key.return) {
            onSubmit(value);
            return;
        }
        if (key.backspace || key.delete) {
            setValue((v) => v.slice(0, -1));
            return;
        }
        if (input && !key.ctrl && !key.meta) {
            setValue((v) => v + input);
        }
    });
    return (_jsxs(Box, { borderStyle: "single", borderColor: "yellow", paddingX: 1, children: [_jsx(Text, { color: "yellow", children: "> " }), _jsx(Text, { children: value }), _jsx(Text, { color: "gray", children: "|" }), _jsx(Box, { flexGrow: 1 }), _jsx(Text, { color: "gray", dimColor: true, children: "enter:send esc:cancel" })] }));
}
