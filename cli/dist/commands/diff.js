import chalk from "chalk";
import { execSync } from "node:child_process";
import { loadStore } from "../decisions.js";
export async function diffCommand() {
    const cwd = process.cwd();
    try {
        execSync("git rev-parse --is-inside-work-tree", {
            cwd,
            stdio: "pipe",
        });
    }
    catch {
        console.log(chalk.red("Not a git repository."));
        return;
    }
    const store = loadStore(cwd);
    if (!store || store.decisions.length === 0) {
        console.log(chalk.yellow("No decisions recorded yet."));
        console.log(chalk.dim("Run defer with a task to start collecting decisions."));
        return;
    }
    const lastDate = store.decisions
        .map((d) => d.date)
        .filter((d) => d.match(/^\d{4}-\d{2}-\d{2}$/))
        .sort()
        .pop();
    if (!lastDate) {
        console.log(chalk.yellow("No valid dates found in decisions."));
        return;
    }
    console.log(chalk.bold(`\n  Changes since last decision (${lastDate})\n`));
    try {
        const commits = execSync(`git log --oneline --after="${lastDate}" --no-merges`, { cwd, encoding: "utf-8" }).trim();
        if (commits) {
            console.log(chalk.cyan("  Commits:"));
            for (const line of commits.split("\n")) {
                console.log(`    ${line}`);
            }
        }
        else {
            console.log(chalk.dim("  No commits since last decision."));
        }
    }
    catch {
        console.log(chalk.dim("  Could not read git log."));
    }
    console.log();
    try {
        const diffStat = execSync(`git diff --stat HEAD~5 2>/dev/null || git diff --stat`, { cwd, encoding: "utf-8" }).trim();
        if (diffStat) {
            console.log(chalk.cyan("  Uncommitted changes:"));
            for (const line of diffStat.split("\n")) {
                console.log(`    ${line}`);
            }
        }
        else {
            console.log(chalk.dim("  No uncommitted changes."));
        }
    }
    catch {
        console.log(chalk.dim("  No uncommitted changes."));
    }
    console.log();
}
