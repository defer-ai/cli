import { describe, it, expect } from "vitest";
import type { Decision } from "./decisions.js";
import { nextDecisionId } from "./decisions.js";

// Replicate the agent's parsing logic as pure functions for testing

function parseStructuredDecisions(
  output: string,
  existing: Decision[]
): Decision[] {
  const match = output.match(/```defer-decisions\s*\n([\s\S]*?)\n```/);
  if (!match) return parseFallbackDecisions(output, existing);

  try {
    const raw = JSON.parse(match[1]);
    if (!Array.isArray(raw)) return [];

    const today = "2026-03-25";
    const result: Decision[] = [];

    for (const item of raw) {
      const cat = item.category || "General";
      const id = nextDecisionId([...existing, ...result], cat);
      result.push({
        id,
        category: cat,
        question: item.question || "",
        options: (item.options || []).map((o: any) => ({
          key: o.key,
          label: o.label,
        })),
        context: item.context || "",
        answer: null,
        delegated: false,
        date: today,
      });
    }

    return result;
  } catch {
    return [];
  }
}

function parseFallbackDecisions(
  output: string,
  existing: Decision[]
): Decision[] {
  const decisions: Decision[] = [];
  const today = "2026-03-25";
  let currentCategory = "General";
  let lastDecision: Decision | null = null;
  const lines = output.split("\n");

  for (const line of lines) {
    const catMatch = line.match(/^##\s+(.+)/);
    if (catMatch && !catMatch[1].startsWith("[")) {
      currentCategory = catMatch[1].trim();
      continue;
    }

    const qMatch = line.match(/\*\*Q\d+:\s*(.+?)\*\*/);
    if (qMatch) {
      const d: Decision = {
        id: nextDecisionId([...existing, ...decisions], currentCategory),
        category: currentCategory,
        question: qMatch[1],
        options: [],
        context: "",
        answer: null,
        delegated: false,
        date: today,
      };
      decisions.push(d);
      lastDecision = d;
      continue;
    }

    if (lastDecision) {
      const optMatch = line.match(
        /^[-*]\s+\*{0,2}([A-Z])[.)]\*{0,2}\.?\s*(.+)/
      );
      if (optMatch) {
        const label = optMatch[2].trim().replace(/\*+/g, "").trim();
        if (label && !lastDecision.options.some((o) => o.key === optMatch[1])) {
          lastDecision.options.push({ key: optMatch[1], label });
        }
      }

      const ctxMatch = line.match(/Context:\s*(.+)/i);
      if (ctxMatch) {
        lastDecision.context = ctxMatch[1].trim();
      }
    }
  }

  return decisions;
}

describe("parseStructuredDecisions", () => {
  it("parses a defer-decisions JSON block", () => {
    const output = `Here are the decisions:

\`\`\`defer-decisions
[
  {
    "category": "Stack",
    "question": "Backend language?",
    "options": [
      {"key": "A", "label": "TypeScript"},
      {"key": "B", "label": "Python"},
      {"key": "C", "label": "Choose for me"}
    ],
    "context": "Affects the entire backend"
  },
  {
    "category": "Stack",
    "question": "Frontend framework?",
    "options": [
      {"key": "A", "label": "React"},
      {"key": "B", "label": "Vue"}
    ],
    "context": "Determines UI architecture"
  },
  {
    "category": "Data",
    "question": "Database?",
    "options": [
      {"key": "A", "label": "PostgreSQL"},
      {"key": "B", "label": "SQLite"}
    ],
    "context": "Data persistence layer"
  }
]
\`\`\``;

    const decisions = parseStructuredDecisions(output, []);

    expect(decisions).toHaveLength(3);
    expect(decisions[0].id).toBe("STACK-001");
    expect(decisions[0].category).toBe("Stack");
    expect(decisions[0].question).toBe("Backend language?");
    expect(decisions[0].options).toHaveLength(3);
    expect(decisions[0].answer).toBeNull();

    expect(decisions[1].id).toBe("STACK-002");
    expect(decisions[2].id).toBe("DATA-001");
  });

  it("generates unique IDs when existing decisions present", () => {
    const existing: Decision[] = [
      {
        id: "STACK-001",
        category: "Stack",
        question: "Existing?",
        options: [],
        context: "",
        answer: "Yes",
        delegated: false,
        date: "2026-03-25",
      },
    ];

    const output = `\`\`\`defer-decisions
[{"category": "Stack", "question": "New?", "options": [], "context": ""}]
\`\`\``;

    const decisions = parseStructuredDecisions(output, existing);
    expect(decisions[0].id).toBe("STACK-002");
  });

  it("handles malformed JSON gracefully", () => {
    const output = `\`\`\`defer-decisions
not valid json
\`\`\``;

    const decisions = parseStructuredDecisions(output, []);
    expect(decisions).toEqual([]);
  });

  it("handles missing fields with defaults", () => {
    const output = `\`\`\`defer-decisions
[{"question": "Something?"}]
\`\`\``;

    const decisions = parseStructuredDecisions(output, []);
    expect(decisions).toHaveLength(1);
    expect(decisions[0].category).toBe("General");
    expect(decisions[0].options).toEqual([]);
    expect(decisions[0].context).toBe("");
  });
});

describe("parseFallbackDecisions", () => {
  it("parses Q&A format with categories", () => {
    const output = `## Architecture

**Q1: Backend framework?**
- **A.** Express
- **B.** Fastify
Context: Determines the backend

## Data

**Q2: Database engine?**
- **A.** PostgreSQL
- **B.** SQLite`;

    const decisions = parseFallbackDecisions(output, []);

    expect(decisions).toHaveLength(2);
    expect(decisions[0].category).toBe("Architecture");
    expect(decisions[0].question).toBe("Backend framework?");
    expect(decisions[0].options).toHaveLength(2);
    expect(decisions[0].options[0]).toEqual({ key: "A", label: "Express" });
    expect(decisions[0].context).toBe("Determines the backend");

    expect(decisions[1].category).toBe("Data");
    expect(decisions[1].question).toBe("Database engine?");
  });

  it("falls back to General category when none specified", () => {
    const output = `**Q1: Something?**`;
    const decisions = parseFallbackDecisions(output, []);
    expect(decisions[0].category).toBe("General");
  });

  it("returns empty for no questions", () => {
    const output = "Just some regular text without any questions.";
    const decisions = parseFallbackDecisions(output, []);
    expect(decisions).toEqual([]);
  });

  it("deduplicates option keys", () => {
    const output = `**Q1: Pick one?**
- **A.** First
- **A.** Duplicate
- **B.** Second`;

    const decisions = parseFallbackDecisions(output, []);
    expect(decisions[0].options).toHaveLength(2);
    expect(decisions[0].options[0].label).toBe("First");
  });
});

describe("deduplication", () => {
  it("filters out decisions with duplicate questions", () => {
    const existing: Decision[] = [
      {
        id: "STACK-001",
        category: "Stack",
        question: "Backend language?",
        options: [],
        context: "",
        answer: "TypeScript",
        delegated: false,
        date: "2026-03-25",
      },
    ];

    const output = `\`\`\`defer-decisions
[
  {"category": "Stack", "question": "Backend language?", "options": [], "context": ""},
  {"category": "Stack", "question": "Frontend framework?", "options": [], "context": ""}
]
\`\`\``;

    const newDecisions = parseStructuredDecisions(output, existing);
    const existingQuestions = new Set(existing.map((d) => d.question));
    const unique = newDecisions.filter(
      (d) => !existingQuestions.has(d.question)
    );

    expect(unique).toHaveLength(1);
    expect(unique[0].question).toBe("Frontend framework?");
  });
});
