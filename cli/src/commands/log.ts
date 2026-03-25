import chalk from "chalk";
import { input, select } from "@inquirer/prompts";
import { addDecision, decisionsExist, createDecisionsFile } from "../decisions.js";

interface LogOptions {
  category?: string;
  question?: string;
  answer?: string;
  delegated?: boolean;
}

const CATEGORIES = [
  "Architecture",
  "Data",
  "Backend",
  "Frontend",
  "UX",
  "Security",
  "DevEx",
  "Infrastructure",
  "Other",
];

export async function logCommand(options: LogOptions): Promise<void> {
  const cwd = process.cwd();

  if (!decisionsExist(cwd)) {
    createDecisionsFile(cwd);
  }

  const category =
    options.category ||
    (await select({
      message: "Category:",
      choices: CATEGORIES.map((c) => ({ name: c, value: c })),
    }));

  const question =
    options.question ||
    (await input({ message: "What was the question/decision?" }));

  if (!question.trim()) {
    console.log(chalk.yellow("Cancelled."));
    return;
  }

  let answer =
    options.answer ||
    (await input({ message: "What was decided?" }));

  if (!answer.trim()) {
    console.log(chalk.yellow("Cancelled."));
    return;
  }

  if (options.delegated) {
    answer = `DELEGATED: ${answer}`;
  }

  const today = new Date().toISOString().split("T")[0];
  const decision = addDecision(cwd, {
    category,
    question: question.trim(),
    answer: answer.trim(),
    date: today,
  });

  console.log(chalk.green(`Logged ${decision.id}: ${question.trim()}`));
}
