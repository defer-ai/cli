import chalk from "chalk";
import { input, select } from "@inquirer/prompts";
import { loadStore, saveStore, createStore, nextDecisionId } from "../decisions.js";
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
export async function logCommand(options) {
    const cwd = process.cwd();
    let store = loadStore(cwd);
    if (!store) {
        store = createStore(cwd, "(manual)");
    }
    const category = options.category ||
        (await select({
            message: "Category:",
            choices: CATEGORIES.map((c) => ({ name: c, value: c })),
        }));
    const question = options.question ||
        (await input({ message: "What was the question/decision?" }));
    if (!question.trim()) {
        console.log(chalk.yellow("Cancelled."));
        return;
    }
    let answer = options.answer ||
        (await input({ message: "What was decided?" }));
    if (!answer.trim()) {
        console.log(chalk.yellow("Cancelled."));
        return;
    }
    const today = new Date().toISOString().split("T")[0];
    const id = nextDecisionId(store.decisions, category);
    store.decisions.push({
        id,
        category,
        question: question.trim(),
        options: [],
        context: "",
        answer: answer.trim(),
        delegated: !!options.delegated,
        assumption: false,
        date: today,
    });
    saveStore(cwd, store);
    console.log(chalk.green(`Logged ${id}: ${question.trim()}`));
}
