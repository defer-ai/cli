import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
const DEFER_DIR = ".defer";
const DECISIONS_JSON = "decisions.json";
const DECISIONS_MD = "DECISIONS.md";
function ensureDir(cwd) {
    const dir = join(cwd, DEFER_DIR);
    if (!existsSync(dir)) {
        mkdirSync(dir, { recursive: true });
    }
}
function jsonPath(cwd) {
    return join(cwd, DEFER_DIR, DECISIONS_JSON);
}
function mdPath(cwd) {
    return join(cwd, DECISIONS_MD);
}
export function storeExists(cwd) {
    return existsSync(jsonPath(cwd));
}
export function loadStore(cwd) {
    const path = jsonPath(cwd);
    if (!existsSync(path))
        return null;
    try {
        return JSON.parse(readFileSync(path, "utf-8"));
    }
    catch {
        return null;
    }
}
export function saveStore(cwd, store) {
    ensureDir(cwd);
    store.updatedAt = new Date().toISOString();
    writeFileSync(jsonPath(cwd), JSON.stringify(store, null, 2));
    generateMarkdown(cwd, store);
}
export function createStore(cwd, task) {
    const store = {
        task,
        decisions: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    };
    saveStore(cwd, store);
    return store;
}
/** Generate a category-scoped ID like STACK-001, DATA-002 */
export function nextDecisionId(decisions, category) {
    const prefix = categoryPrefix(category);
    const existing = decisions.filter((d) => d.id.startsWith(prefix + "-"));
    const maxNum = existing.reduce((max, d) => {
        const parts = d.id.split("-");
        const num = parseInt(parts[parts.length - 1], 10);
        return num > max ? num : max;
    }, 0);
    return `${prefix}-${String(maxNum + 1).padStart(3, "0")}`;
}
/** Convert a category name to a short uppercase prefix */
function categoryPrefix(category) {
    // Use the category as-is if it's already short and uppercase
    const clean = category
        .replace(/[^a-zA-Z0-9\s]/g, "")
        .trim()
        .toUpperCase();
    // If single word and short, use it directly
    if (clean.length <= 6 && !clean.includes(" ")) {
        return clean;
    }
    // Take first letters of each word, or first 4 chars
    const words = clean.split(/\s+/);
    if (words.length > 1) {
        return words
            .map((w) => w[0])
            .join("")
            .slice(0, 5);
    }
    return clean.slice(0, 4);
}
/** Generate DECISIONS.md from the JSON store */
function generateMarkdown(cwd, store) {
    const userDecisions = store.decisions.filter((d) => !d.implicit);
    const aiDecisions = store.decisions.filter((d) => d.implicit);
    const lines = [
        "# DECISIONS.md",
        "",
        `> Task: ${store.task}`,
        "",
        "## Decisions",
        "",
        "| ID | Category | Question | Answer | Date |",
        "|----|----------|----------|--------|------|",
    ];
    for (const d of userDecisions) {
        const answer = d.answer
            ? d.delegated
                ? `DELEGATED: ${d.answer}`
                : d.answer
            : "(pending)";
        lines.push(`| ${d.id} | ${d.category} | ${d.question} | ${answer} | ${d.date} |`);
    }
    if (aiDecisions.length > 0) {
        lines.push("");
        lines.push("## AI Choices");
        lines.push("");
        lines.push("| ID | Category | What was decided | Reasoning |");
        lines.push("|----|----------|------------------|-----------|");
        for (const d of aiDecisions) {
            lines.push(`| ${d.id} | ${d.category} | ${d.answer || d.question} | ${d.reasoning || ""} |`);
        }
    }
    lines.push("");
    writeFileSync(mdPath(cwd), lines.join("\n"));
}
