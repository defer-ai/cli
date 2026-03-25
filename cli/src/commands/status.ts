import chalk from "chalk";
import { select } from "@inquirer/prompts";
import { parseDecisions, type Decision } from "../decisions.js";

function formatDecision(d: Decision, index: number): string {
  const isDelegated = d.answer.startsWith("DELEGATED");
  const answerColor = isDelegated ? chalk.yellow : chalk.white;
  return `${chalk.cyan(d.id)} ${chalk.dim(d.category.padEnd(15))} ${d.question.padEnd(40)} ${answerColor(d.answer)} ${chalk.dim(d.date)}`;
}

export async function statusCommand(): Promise<void> {
  const cwd = process.cwd();
  const decisions = parseDecisions(cwd);

  if (decisions.length === 0) {
    console.log(chalk.yellow("No decisions recorded yet."));
    console.log(chalk.dim("Run your AI tool with Defer mode to start collecting decisions."));
    return;
  }

  console.log(chalk.bold(`\n  Decision Record (${decisions.length} decisions)\n`));

  // Group by category
  const categories = new Map<string, Decision[]>();
  for (const d of decisions) {
    const cat = d.category || "Uncategorized";
    if (!categories.has(cat)) categories.set(cat, []);
    categories.get(cat)!.push(d);
  }

  for (const [category, items] of categories) {
    console.log(chalk.cyan.bold(`  ${category}`));
    for (const d of items) {
      const isDelegated = d.answer.startsWith("DELEGATED");
      const marker = isDelegated ? chalk.yellow("  [delegated]") : "";
      console.log(`    ${chalk.dim(d.id)} ${d.question}`);
      console.log(`         ${chalk.white(d.answer)}${marker} ${chalk.dim(d.date)}`);
    }
    console.log();
  }

  // Summary
  const delegated = decisions.filter((d) => d.answer.startsWith("DELEGATED")).length;
  const userDecided = decisions.length - delegated;

  console.log(chalk.dim("  ---"));
  console.log(
    `  ${chalk.white(String(userDecided))} decided by you, ${chalk.yellow(String(delegated))} delegated to AI`
  );
  console.log();

  // Offer to revisit
  const action = await select({
    message: "What next?",
    choices: [
      { name: "Done", value: "done" },
      { name: "Revisit a decision", value: "revisit" },
    ],
  });

  if (action === "revisit") {
    const choices = decisions.map((d) => ({
      name: `${d.id}: ${d.question} [${d.answer}]`,
      value: d.id,
    }));

    const id = await select({
      message: "Which decision to revisit?",
      choices,
    });

    console.log();
    console.log(
      chalk.cyan(`To revisit ${id}, tell your AI: "Revisit ${id}"`)
    );
    console.log(
      chalk.dim("The AI will re-ask the question and update the record.")
    );
  }
}
