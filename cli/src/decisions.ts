import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { join } from "node:path";

export interface Decision {
  id: string;
  category: string;
  question: string;
  answer: string;
  date: string;
}

const DECISIONS_FILE = "DECISIONS.md";
const HEADER = `# DECISIONS.md

| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|`;

function getPath(cwd: string): string {
  return join(cwd, DECISIONS_FILE);
}

export function decisionsExist(cwd: string): boolean {
  return existsSync(getPath(cwd));
}

export function createDecisionsFile(cwd: string): void {
  writeFileSync(getPath(cwd), HEADER + "\n");
}

export function parseDecisions(cwd: string): Decision[] {
  const path = getPath(cwd);
  if (!existsSync(path)) return [];

  const content = readFileSync(path, "utf-8");
  const lines = content.split("\n");
  const decisions: Decision[] = [];

  for (const line of lines) {
    const match = line.match(
      /^\|\s*(D\d+)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|$/
    );
    if (match && match[1] !== "ID") {
      decisions.push({
        id: match[1],
        category: match[2],
        question: match[3],
        answer: match[4],
        date: match[5],
      });
    }
  }

  return decisions;
}

export function addDecision(
  cwd: string,
  decision: Omit<Decision, "id">
): Decision {
  const existing = parseDecisions(cwd);
  const nextNum = existing.length + 1;
  const id = `D${String(nextNum).padStart(3, "0")}`;

  const newDecision: Decision = { id, ...decision };
  const line = `| ${id} | ${decision.category} | ${decision.question} | ${decision.answer} | ${decision.date} |`;

  const path = getPath(cwd);
  if (!existsSync(path)) {
    writeFileSync(path, HEADER + "\n" + line + "\n");
  } else {
    const content = readFileSync(path, "utf-8");
    writeFileSync(path, content.trimEnd() + "\n" + line + "\n");
  }

  return newDecision;
}

export function updateDecision(
  cwd: string,
  id: string,
  newAnswer: string
): void {
  const path = getPath(cwd);
  const content = readFileSync(path, "utf-8");
  const lines = content.split("\n");
  const today = new Date().toISOString().split("T")[0];

  const updated = lines.map((line: string) => {
    const match = line.match(
      /^\|\s*(D\d+)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|\s*(.*?)\s*\|$/
    );
    if (match && match[1] === id) {
      return `| ${match[1]} | ${match[2]} | ${match[3]} | ${newAnswer} | ${today} |`;
    }
    return line;
  });

  writeFileSync(path, updated.join("\n"));
}
