import chalk from "chalk";
import { select, input } from "@inquirer/prompts";
import { parseDecisions, updateDecision, } from "../decisions.js";
export async function revisitCommand(idArg) {
    const cwd = process.cwd();
    const decisions = parseDecisions(cwd);
    if (decisions.length === 0) {
        console.log(chalk.yellow("No decisions recorded yet."));
        return;
    }
    let id = idArg?.toUpperCase();
    if (!id) {
        const choices = decisions.map((d) => ({
            name: `${chalk.cyan(d.id)} ${d.question} ${chalk.dim(`[${d.answer}]`)}`,
            value: d.id,
        }));
        id = await select({
            message: "Which decision do you want to revisit?",
            choices,
        });
    }
    const decision = decisions.find((d) => d.id === id);
    if (!decision) {
        console.log(chalk.red(`Decision ${id} not found.`));
        return;
    }
    console.log();
    console.log(chalk.bold(`  Revisiting ${decision.id}`));
    console.log(`  Category: ${chalk.cyan(decision.category)}`);
    console.log(`  Question: ${decision.question}`);
    console.log(`  Current:  ${chalk.dim(decision.answer)}`);
    console.log();
    const newAnswer = await input({
        message: "New answer (or empty to cancel):",
    });
    if (!newAnswer.trim()) {
        console.log(chalk.yellow("Cancelled."));
        return;
    }
    updateDecision(cwd, id, newAnswer.trim());
    console.log(chalk.green(`Updated ${id}: ${decision.question} -> ${newAnswer.trim()}`));
    console.log(chalk.dim("The updated answer will be used for future work in this project."));
}
