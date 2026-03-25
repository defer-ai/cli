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
  return parseDecisionsFromString(content);
}

export function parseDecisionsFromString(content: string): Decision[] {
  const lines = content.split("\n");
  const decisions: Decision[] = [];

  for (const line of lines) {
    // Match table rows, being lenient with whitespace and content
    const trimmed = line.trim();
    if (!trimmed.startsWith("|")) continue;

    const cells = trimmed
      .split("|")
      .slice(1, -1)
      .map((c) => c.trim());

    if (cells.length < 5) continue;

    const [id, category, question, answer, date] = cells;

    // Skip header and separator rows
    if (id === "ID" || id.startsWith("-")) continue;
    if (!id.match(/^D\d+$/)) continue;

    decisions.push({ id, category, question, answer, date });
  }

  return decisions;
}

export function nextDecisionId(decisions: Decision[]): string {
  const maxNum = decisions.reduce((max, d) => {
    const num = parseInt(d.id.slice(1), 10);
    return num > max ? num : max;
  }, 0);
  return `D${String(maxNum + 1).padStart(3, "0")}`;
}

export function addDecision(
  cwd: string,
  decision: Omit<Decision, "id">
): Decision {
  const existing = parseDecisions(cwd);
  const id = nextDecisionId(existing);
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
): boolean {
  const path = getPath(cwd);
  if (!existsSync(path)) return false;

  const content = readFileSync(path, "utf-8");
  const lines = content.split("\n");
  const today = new Date().toISOString().split("T")[0];
  let found = false;

  const updated = lines.map((line: string) => {
    const trimmed = line.trim();
    if (!trimmed.startsWith("|")) return line;

    const cells = trimmed
      .split("|")
      .slice(1, -1)
      .map((c) => c.trim());

    if (cells.length >= 5 && cells[0] === id) {
      found = true;
      return `| ${cells[0]} | ${cells[1]} | ${cells[2]} | ${newAnswer} | ${today} |`;
    }
    return line;
  });

  if (found) {
    writeFileSync(path, updated.join("\n"));
  }
  return found;
}
