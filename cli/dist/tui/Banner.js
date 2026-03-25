import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Text } from "ink";
const BANNER = `
     ██████╗ ███████╗███████╗███████╗██████╗
     ██╔══██╗██╔════╝██╔════╝██╔════╝██╔══██╗
     ██║  ██║█████╗  █████╗  █████╗  ██████╔╝
     ██║  ██║██╔══╝  ██╔══╝  ██╔══╝  ██╔══██╗
     ██████╔╝███████╗██║     ███████╗██║  ██║
     ╚═════╝ ╚══════╝╚═╝     ╚══════╝╚═╝  ╚═╝
`;
export function Banner({ model, cwd }) {
    const dir = cwd.replace(process.env.HOME || "", "~");
    return (_jsxs(Box, { flexDirection: "column", paddingX: 2, children: [_jsx(Text, { color: "cyan", children: BANNER }), _jsx(Box, { children: _jsx(Text, { color: "gray", children: "     Zero-autonomy AI. Every decision is yours." }) }), _jsxs(Box, { marginTop: 1, children: [_jsx(Text, { color: "gray", children: "     model: " }), _jsx(Text, { color: "white", children: model }), _jsx(Text, { color: "gray", children: "  cwd: " }), _jsx(Text, { color: "white", children: dir })] }), _jsx(Box, { marginTop: 1, children: _jsx(Text, { color: "gray", children: "     Type a task to start. /help for commands." }) })] }));
}
