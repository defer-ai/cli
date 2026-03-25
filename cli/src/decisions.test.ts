import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, rmSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  createStore,
  loadStore,
  saveStore,
  storeExists,
  nextDecisionId,
  type Decision,
  type DecisionStore,
} from "./decisions.js";

function makeDecision(overrides: Partial<Decision> = {}): Decision {
  return {
    id: "TEST-001",
    category: "Test",
    question: "Test question?",
    options: [
      { key: "A", label: "Option A" },
      { key: "B", label: "Option B" },
    ],
    context: "Test context",
    answer: null,
    delegated: false,
    implicit: false,
    date: "2026-03-25",
    ...overrides,
  };
}

describe("Decision Store", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = mkdtempSync(join(tmpdir(), "defer-test-"));
  });

  afterEach(() => {
    rmSync(tmpDir, { recursive: true, force: true });
  });

  describe("createStore", () => {
    it("creates .defer/decisions.json and DECISIONS.md", () => {
      const store = createStore(tmpDir, "build a todo app");

      expect(store.task).toBe("build a todo app");
      expect(store.decisions).toEqual([]);
      expect(existsSync(join(tmpDir, ".defer", "decisions.json"))).toBe(true);
      expect(existsSync(join(tmpDir, "DECISIONS.md"))).toBe(true);
    });

    it("sets timestamps", () => {
      const store = createStore(tmpDir, "test");

      expect(store.createdAt).toBeDefined();
      expect(store.updatedAt).toBeDefined();
    });
  });

  describe("storeExists", () => {
    it("returns false for empty directory", () => {
      expect(storeExists(tmpDir)).toBe(false);
    });

    it("returns true after creating store", () => {
      createStore(tmpDir, "test");
      expect(storeExists(tmpDir)).toBe(true);
    });
  });

  describe("loadStore", () => {
    it("returns null for non-existent store", () => {
      expect(loadStore(tmpDir)).toBeNull();
    });

    it("loads a saved store", () => {
      createStore(tmpDir, "my task");
      const loaded = loadStore(tmpDir);

      expect(loaded).not.toBeNull();
      expect(loaded!.task).toBe("my task");
    });

    it("preserves decisions through save/load cycle", () => {
      const store = createStore(tmpDir, "test");
      store.decisions.push(makeDecision({ id: "STACK-001", answer: "TypeScript" }));
      saveStore(tmpDir, store);

      const loaded = loadStore(tmpDir);
      expect(loaded!.decisions).toHaveLength(1);
      expect(loaded!.decisions[0].id).toBe("STACK-001");
      expect(loaded!.decisions[0].answer).toBe("TypeScript");
    });
  });

  describe("saveStore", () => {
    it("generates DECISIONS.md with correct format", () => {
      const store = createStore(tmpDir, "test task");
      store.decisions.push(
        makeDecision({ id: "STACK-001", category: "Stack", question: "Language?", answer: "TypeScript" }),
        makeDecision({ id: "DATA-001", category: "Data", question: "Database?", answer: null }),
        makeDecision({ id: "UI-001", category: "UI", question: "Framework?", answer: "React", delegated: true }),
      );
      saveStore(tmpDir, store);

      const md = readFileSync(join(tmpDir, "DECISIONS.md"), "utf-8");

      expect(md).toContain("# DECISIONS.md");
      expect(md).toContain("> Task: test task");
      expect(md).toContain("| STACK-001 | Stack | Language? | TypeScript |");
      expect(md).toContain("| DATA-001 | Data | Database? | (pending) |");
      expect(md).toContain("| UI-001 | UI | Framework? | DELEGATED: React |");
    });

    it("sets updatedAt on save", () => {
      const store = createStore(tmpDir, "test");
      store.decisions.push(makeDecision());
      saveStore(tmpDir, store);
      const loaded = loadStore(tmpDir);

      expect(loaded!.updatedAt).toBeDefined();
      expect(new Date(loaded!.updatedAt).getTime()).toBeGreaterThan(0);
    });
  });
});

describe("nextDecisionId", () => {
  it("returns PREFIX-001 for empty list", () => {
    expect(nextDecisionId([], "Stack")).toBe("STACK-001");
  });

  it("increments within category", () => {
    const decisions = [
      makeDecision({ id: "STACK-001", category: "Stack" }),
      makeDecision({ id: "STACK-002", category: "Stack" }),
    ];
    expect(nextDecisionId(decisions, "Stack")).toBe("STACK-003");
  });

  it("counts independently per category", () => {
    const decisions = [
      makeDecision({ id: "STACK-001", category: "Stack" }),
      makeDecision({ id: "STACK-002", category: "Stack" }),
      makeDecision({ id: "DATA-001", category: "Data" }),
    ];
    expect(nextDecisionId(decisions, "Data")).toBe("DATA-002");
    expect(nextDecisionId(decisions, "Stack")).toBe("STACK-003");
    expect(nextDecisionId(decisions, "UI")).toBe("UI-001");
  });

  it("truncates long single-word categories to 4 chars", () => {
    expect(nextDecisionId([], "Infrastructure")).toBe("INFR-001");
  });

  it("uses initials for multi-word categories", () => {
    expect(nextDecisionId([], "Error Handling")).toBe("EH-001");
  });

  it("handles short categories directly", () => {
    expect(nextDecisionId([], "API")).toBe("API-001");
    expect(nextDecisionId([], "UX")).toBe("UX-001");
  });

  it("strips special characters", () => {
    expect(nextDecisionId([], "UI/UX")).toBe("UIUX-001");
  });

  it("handles accumulation within a batch", () => {
    const batch: Decision[] = [];
    const d1 = makeDecision({ id: nextDecisionId(batch, "Stack"), category: "Stack" });
    batch.push(d1);
    const d2 = makeDecision({ id: nextDecisionId(batch, "Stack"), category: "Stack" });
    batch.push(d2);
    const d3 = makeDecision({ id: nextDecisionId(batch, "Stack"), category: "Stack" });

    expect(d1.id).toBe("STACK-001");
    expect(d2.id).toBe("STACK-002");
    expect(d3.id).toBe("STACK-003");
  });
});

