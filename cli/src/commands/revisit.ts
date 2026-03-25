import chalk from "chalk";
import { select, input } from "@inquirer/prompts";
import { loadStore, saveStore } from "../decisions.js";

export async function revisitCommand(idArg?: string): Promise<void> {
  const cwd = process.cwd();
  const store = loadStore(cwd);

  if (!store || store.decisions.length === 0) {
    console.log(chalk.yellow("No decisions recorded yet."));
    return;
  }

  let id = idArg?.toUpperCase();

  if (!id) {
    const choices = store.decisions.map((d) => ({
      name: `${chalk.cyan(d.id)} ${d.question} ${chalk.dim(`[${d.answer ?? "pending"}]`)}`,
      value: d.id,
    }));

    id = await select({
      message: "Which decision do you want to revisit?",
      choices,
    });
  }

  const decision = store.decisions.find((d) => d.id === id);
  if (!decision) {
    console.log(chalk.red(`Decision ${id} not found.`));
    return;
  }

  console.log();
  console.log(chalk.bold(`  Revisiting ${decision.id}`));
  console.log(`  Category: ${chalk.cyan(decision.category)}`);
  console.log(`  Question: ${decision.question}`);
  if (decision.context) {
    console.log(`  Context:  ${chalk.dim(decision.context)}`);
  }
  console.log(`  Current:  ${chalk.dim(decision.answer ?? "(pending)")}`);

  if (decision.options.length > 0) {
    console.log(`  Options:`);
    for (const o of decision.options) {
      console.log(`    ${o.key}) ${o.label}`);
    }
  }
  console.log();

  const newAnswer = await input({
    message: "New answer (or empty to cancel):",
  });

  if (!newAnswer.trim()) {
    console.log(chalk.yellow("Cancelled."));
    return;
  }

  decision.answer = newAnswer.trim();
  decision.date = new Date().toISOString().split("T")[0];
  saveStore(cwd, store);

  console.log(
    chalk.green(`Updated ${id}: ${decision.question} -> ${newAnswer.trim()}`)
  );
  console.log(
    chalk.dim("The updated answer will be used for future work in this project.")
  );
}
