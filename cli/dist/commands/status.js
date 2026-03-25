import chalk from "chalk";
import { select } from "@inquirer/prompts";
import { loadStore } from "../decisions.js";
export async function statusCommand() {
    const cwd = process.cwd();
    const store = loadStore(cwd);
    if (!store || store.decisions.length === 0) {
        console.log(chalk.yellow("No decisions recorded yet."));
        console.log(chalk.dim("Run defer with a task to start collecting decisions."));
        return;
    }
    console.log(chalk.bold(`\n  Decision Record (${store.decisions.length} decisions)\n`));
    console.log(chalk.dim(`  Task: ${store.task}\n`));
    const categories = new Map();
    for (const d of store.decisions) {
        const cat = d.category || "Uncategorized";
        if (!categories.has(cat))
            categories.set(cat, []);
        categories.get(cat).push(d);
    }
    for (const [category, items] of categories) {
        console.log(chalk.cyan.bold(`  ${category}`));
        for (const d of items) {
            const isPending = d.answer === null;
            const marker = d.delegated ? chalk.magenta("  [delegated]") : "";
            console.log(`    ${chalk.dim(d.id)} ${d.question}`);
            console.log(`         ${isPending ? chalk.yellow("(pending)") : chalk.white(d.answer)}${marker} ${chalk.dim(d.date)}`);
        }
        console.log();
    }
    const delegated = store.decisions.filter((d) => d.delegated).length;
    const pending = store.decisions.filter((d) => d.answer === null).length;
    const answered = store.decisions.length - pending - delegated;
    console.log(chalk.dim("  ---"));
    console.log(`  ${chalk.white(String(answered))} decided by you, ${chalk.magenta(String(delegated))} delegated, ${chalk.yellow(String(pending))} pending`);
    console.log();
    const action = await select({
        message: "What next?",
        choices: [
            { name: "Done", value: "done" },
            { name: "Revisit a decision", value: "revisit" },
        ],
    });
    if (action === "revisit") {
        const choices = store.decisions.map((d) => ({
            name: `${d.id}: ${d.question} [${d.answer ?? "pending"}]`,
            value: d.id,
        }));
        const id = await select({
            message: "Which decision to revisit?",
            choices,
        });
        console.log();
        console.log(chalk.cyan(`To revisit ${id}, run: defer revisit ${id}`));
    }
}
