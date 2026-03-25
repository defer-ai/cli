import { select, confirm } from "@inquirer/prompts";
import { writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import chalk from "chalk";
import { templates, type Target } from "../templates.js";
import { storeExists, createStore } from "../decisions.js";

const targetChoices = [
  { name: "Claude Code (CLAUDE.md)", value: "claude-code" as Target },
  { name: "Cursor (.cursor/rules/defer.mdc)", value: "cursor" as Target },
  { name: "ChatGPT (custom instructions)", value: "chatgpt" as Target },
  { name: "Universal (any AI tool)", value: "universal" as Target },
  { name: "API (system prompt)", value: "api" as Target },
];

export async function initCommand(targetArg?: string): Promise<void> {
  const cwd = process.cwd();

  let target: Target;

  if (targetArg && targetArg in templates) {
    target = targetArg as Target;
  } else {
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
