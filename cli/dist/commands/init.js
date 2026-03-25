import { select, confirm } from "@inquirer/prompts";
import { writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import chalk from "chalk";
import { templates } from "../templates.js";
import { storeExists, createStore } from "../decisions.js";
const targetChoices = [
    { name: "Claude Code (CLAUDE.md)", value: "claude-code" },
    { name: "Cursor (.cursor/rules/defer.mdc)", value: "cursor" },
    { name: "ChatGPT (custom instructions)", value: "chatgpt" },
    { name: "Universal (any AI tool)", value: "universal" },
    { name: "API (system prompt)", value: "api" },
];
export async function initCommand(targetArg) {
    const cwd = process.cwd();
    let target;
    if (targetArg && targetArg in templates) {
        target = targetArg;
    }
    else {
        target = await select({
            message: "Which AI tool are you using?",
            choices: targetChoices,
        });
    }
    const template = templates[target];
    const filepath = join(cwd, template.filename);
    if (existsSync(filepath)) {
        const overwrite = await confirm({
            message: `${template.filename} already exists. Overwrite?`,
            default: false,
        });
        if (!overwrite) {
            console.log(chalk.yellow("Skipped."));
            return;
        }
    }
    const dir = dirname(filepath);
    if (!existsSync(dir)) {
        mkdirSync(dir, { recursive: true });
    }
    writeFileSync(filepath, template.content);
    console.log(chalk.green(`Created ${template.filename}`));
    if (!storeExists(cwd)) {
        createStore(cwd, "(not started)");
        console.log(chalk.green("Created .defer/decisions.json"));
    }
    console.log();
    console.log(chalk.cyan("Defer mode is active."));
    console.log(`Every decision will be surfaced before any code is written.`);
}
