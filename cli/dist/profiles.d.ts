import type { Decision } from "./decisions.js";
/** Save a set of decisions as a named profile (category::question -> answer). */
export declare function saveProfile(name: string, decisions: Record<string, string>): void;
/** Load a profile by name. Returns null if it doesn't exist. */
export declare function loadProfile(name: string): Record<string, string> | null;
/** List all saved profile names. */
export declare function listProfiles(): string[];
/**
 * Apply a profile to a set of decisions.
 * For each decision whose "category::question" key exists in the profile,
 * the answer is filled in from the profile.
 * Returns the updated decisions array.
 */
export declare function applyProfile(profileName: string, decisions: Decision[]): Decision[];
