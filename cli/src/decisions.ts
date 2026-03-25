import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";

export interface DecisionOption {
  key: string;
  label: string;
}

export interface Decision {
  id: string;
  category: string;
  question: string;
  options: DecisionOption[];
  context: string;
  answer: string | null;
  delegated: boolean;
  date: string;
}

export interface DecisionStore {
  task: string;
  decisions: Decision[];
  createdAt: string;
  updatedAt: string;
}

const DEFER_DIR = ".defer";
const DECISIONS_JSON = "decisions.json";
const DECISIONS_MD = "DECISIONS.md";

function ensureDir(cwd: string): void {
  const dir = join(cwd, DEFER_DIR);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
}

function jsonPath(cwd: string): string {
  return join(cwd, DEFER_DIR, DECISIONS_JSON);
}

function mdPath(cwd: string): string {
  return join(cwd, DECISIONS_MD);
}

export function storeExists(cwd: string): boolean {
  return existsSync(jsonPath(cwd));
}

export function loadStore(cwd: string): DecisionStore | null {
  const path = jsonPath(cwd);
  if (!existsSync(path)) return null;
  try {
    return JSON.parse(readFileSync(path, "utf-8"));
  } catch {
    return null;
  }
}

export function saveStore(cwd: string, store: DecisionStore): void {
  ensureDir(cwd);
  store.updatedAt = new Date().toISOString();
  writeFileSync(jsonPath(cwd), JSON.stringify(store, null, 2));
  generateMarkdown(cwd, store);
}

export function createStore(cwd: string, task: string): DecisionStore {
  const store: DecisionStore = {
    task,
    decisions: [],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
  saveStore(cwd, store);
  return store;
}

export function nextDecisionId(decisions: Decision[]): string {
  const maxNum = decisions.reduce((max, d) => {
    const num = parseInt(d.id.slice(1), 10);
    return num > max ? num : max;
  }, 0);
  return `D${String(maxNum + 1).padStart(3, "0")}`;
}

/** Generate DECISIONS.md from the JSON store */
function generateMarkdown(cwd: string, store: DecisionStore): void {
  const lines = [
    "# DECISIONS.md",
    "",
    `> Task: ${store.task}`,
    "",
    "| ID | Category | Question | Answer | Date |",
    "|----|----------|----------|--------|------|",
  ];

  for (const d of store.decisions) {
    const answer = d.answer
      ? d.delegated
        ? `DELEGATED: ${d.answer}`
        : d.answer
      : "(pending)";
    lines.push(
      `| ${d.id} | ${d.category} | ${d.question} | ${answer} | ${d.date} |`
    );
  }

  lines.push("");
  writeFileSync(mdPath(cwd), lines.join("\n"));
}

// Legacy support: parse old-format DECISIONS.md
export function parseLegacyDecisions(cwd: string): Decision[] {
  const path = mdPath(cwd);
  if (!existsSync(path)) return [];

  const content = readFileSync(path, "utf-8");
  const decisions: Decision[] = [];

  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed.startsWith("|")) continue;

    const cells = trimmed
      .split("|")
      .slice(1, -1)
      .map((c) => c.trim());

    if (cells.length < 5) continue;
    const [id, category, question, answer, date] = cells;
    if (id === "ID" || id.startsWith("-") || !id.match(/^D\d+$/)) continue;

    decisions.push({
      id,
      category,
      question,
      options: [],
      context: "",
      answer: answer === "(pending)" ? null : answer,
      delegated: answer.startsWith("DELEGATED"),
      date,
    });
  }

  return decisions;
}
