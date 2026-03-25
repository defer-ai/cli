import { readFileSync, writeFileSync, existsSync, mkdirSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { homedir } from "node:os";
const PROFILES_DIR = join(homedir(), ".defer", "profiles");
function ensureProfilesDir() {
    if (!existsSync(PROFILES_DIR)) {
        mkdirSync(PROFILES_DIR, { recursive: true });
    }
}
function profilePath(name) {
    return join(PROFILES_DIR, `${name}.json`);
}
/** Save a set of decisions as a named profile (category::question -> answer). */
export function saveProfile(name, decisions) {
    ensureProfilesDir();
    writeFileSync(profilePath(name), JSON.stringify(decisions, null, 2));
}
/** Load a profile by name. Returns null if it doesn't exist. */
export function loadProfile(name) {
    const path = profilePath(name);
    if (!existsSync(path))
        return null;
    try {
        return JSON.parse(readFileSync(path, "utf-8"));
    }
    catch {
        return null;
    }
}
/** List all saved profile names. */
export function listProfiles() {
    ensureProfilesDir();
    return readdirSync(PROFILES_DIR)
        .filter((f) => f.endsWith(".json"))
        .map((f) => f.replace(/\.json$/, ""));
}
/**
 * Apply a profile to a set of decisions.
 * For each decision whose "category::question" key exists in the profile,
 * the answer is filled in from the profile.
 * Returns the updated decisions array.
 */
export function applyProfile(profileName, decisions) {
    const profile = loadProfile(profileName);
    if (!profile)
        return decisions;
    return decisions.map((d) => {
        const key = `${d.category}::${d.question}`;
        if (key in profile && d.answer === null) {
            return { ...d, answer: profile[key] };
        }
        return d;
    });
}
