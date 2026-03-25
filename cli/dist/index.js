#!/usr/bin/env node
import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { statusCommand } from "./commands/status.js";
import { revisitCommand } from "./commands/revisit.js";
import { logCommand } from "./commands/log.js";
import { diffCommand } from "./commands/diff.js";
const program = new Command();
program
    .name("defer")
    .description("Zero-Autonomy AI. Every decision is yours.")
    .version("0.1.0");
// Main command: `defer` or `defer "build auth"`
program
    .argument("[task]", "Task to run (or omit to start interactive mode)")
    .action(async (task) => {
    const { render } = await import("ink");
    const React = await import("react");
    const { App } = await import("./tui/App.js");
    const { ClaudeCodeProvider } = await import("./providers/claude-code.js");
    const provider = new ClaudeCodeProvider();
    if (!provider.isConfigured()) {
        console.error("Error: claude is not installed or not in PATH.");
        console.error("Install it: npm install -g @anthropic-ai/claude-code");
        console.error("Then run: claude login");
        process.exit(1);
    }
    // Alternate screen buffer
    process.stdout.write("\x1b[?1049h\x1b[2J\x1b[H");
    const instance = render(React.createElement(App, { task, provider }), { exitOnCtrlC: true });
    instance.waitUntilExit().then(() => {
        process.stdout.write("\x1b[?1049l");
    });
});
// Subcommands
program
    .command("init")
    .description("Scaffold Defer config files")
    .argument("[target]", "claude-code, cursor, chatgpt, universal, api")
    .action(initCommand);
program
    .command("status")
    .description("View decision record")
    .action(statusCommand);
program
    .command("revisit")
    .description("Revisit a decision")
    .argument("[id]", "Decision ID (e.g. STACK-001)")
    .action(revisitCommand);
program
    .command("log")
    .description("Add a decision manually")
    .option("-c, --category <category>", "Category")
    .option("-q, --question <question>", "Question")
    .option("-a, --answer <answer>", "Answer")
    .option("-d, --delegated", "Mark as delegated")
    .action(logCommand);
program
    .command("diff")
    .description("Git changes since last decision")
    .action(diffCommand);
program.parse();
