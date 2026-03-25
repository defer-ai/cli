import { readFileSync, writeFileSync, existsSync, mkdirSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { homedir } from "node:os";
const HISTORY_DIR = join(homedir(), ".defer", "history");
function ensureHistoryDir() {
    if (!existsSync(HISTORY_DIR)) {
        mkdirSync(HISTORY_DIR, { recursive: true });
    }
}
/** Convert a task description into a filesystem-safe slug. */
function slugify(text) {
    return text
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "")
        .slice(0, 50);
}
/** Save a completed session to history. */
export function saveToHistory(task, decisions, cost, duration) {
    ensureHistoryDir();
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const slug = slugify(task);
    const filename = `${timestamp}-${slug}.json`;
    const entry = {
        task,
        decisions,
        completedAt: new Date().toISOString(),
        cost,
        duration,
    };
    writeFileSync(join(HISTORY_DIR, filename), JSON.stringify(entry, null, 2));
    return filename;
}
/** List recent history entries, newest first. */
export function listHistory(limit = 20) {
    ensureHistoryDir();
    return readdirSync(HISTORY_DIR)
        .filter((f) => f.endsWith(".json"))
        .sort()
        .reverse()
        .slice(0, limit);
}
/** Load a specific history entry by filename. Returns null if not found. */
export function loadHistoryEntry(filename) {
    const path = join(HISTORY_DIR, filename);
    if (!existsSync(path))
        return null;
    try {
        return JSON.parse(readFileSync(path, "utf-8"));
    }
    catch {
        return null;
    }
}
