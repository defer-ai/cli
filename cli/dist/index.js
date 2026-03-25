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
// Main command: `defer "build auth"` launches the TUI
program
    .argument("[task]", "Task to run with Defer mode (launches TUI dashboard)")
    .action(async (task) => {
    if (!task) {
        program.help();
        return;
    }
    // Dynamic import to avoid loading ink for subcommands
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
    // Enter alternate screen buffer to prevent line duplication
    process.stdout.write("\x1b[?1049h");
    process.stdout.write("\x1b[2J\x1b[H");
    const instance = render(React.createElement(App, { task, provider }), {
        exitOnCtrlC: true,
    });
    instance.waitUntilExit().then(() => {
        // Restore main screen buffer
        process.stdout.write("\x1b[?1049l");
    });
});
// Subcommands for non-TUI operations
program
    .command("init")
    .description("Scaffold Defer mode config files into your project")
    .argument("[target]", "Target tool: claude-code, cursor, chatgpt, universal, api")
    .action(initCommand);
program
    .command("status")
    .description("View and navigate your decision record")
    .action(statusCommand);
program
    .command("revisit")
    .description("Revisit and change a previous decision")
    .argument("[id]", "Decision ID to revisit (e.g. D001)")
    .action(revisitCommand);
program
    .command("log")
    .description("Add a decision to the record")
    .option("-c, --category <category>", "Decision category")
    .option("-q, --question <question>", "The question that was decided")
    .option("-a, --answer <answer>", "The answer/choice made")
    .option("-d, --delegated", "Mark as delegated to AI")
    .action(logCommand);
program
    .command("diff")
    .description("Show git changes since last decision review")
    .action(diffCommand);
program.parse();
