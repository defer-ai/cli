import chalk from "chalk";
import { writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
const DEFER_QUEUE_FILE = ".defer-ask";
export async function askCommand(topic) {
    const cwd = process.cwd();
    const filepath = join(cwd, DEFER_QUEUE_FILE);
    const timestamp = new Date().toISOString().split("T")[0];
    const entry = `[${timestamp}] Ask about: ${topic}`;
    // Append to queue file
    let content = "";
    if (existsSync(filepath)) {
        content = readFileSync(filepath, "utf-8").trimEnd() + "\n";
    }
    content += entry + "\n";
    writeFileSync(filepath, content);
    console.log(chalk.green(`Queued: "${topic}"`));
    console.log();
    console.log(`Next time your AI reads the project, it will see this and generate`);
    console.log(`decision questions specifically about ${chalk.cyan(topic)}.`);
    console.log();
    console.log(chalk.dim("You can also tell your AI directly:"));
    console.log(chalk.dim(`  "Ask about ${topic}"`));
    console.log();
    console.log(chalk.dim(`Queue file: ${DEFER_QUEUE_FILE}`));
}
