"use client";

import { useState, useEffect, useRef } from "react";
import { WebMascot } from "./mascot";

type Phase =
  | "idle"
  | "boot"
  | "decomposing"
  | "domains"
  | "domain-adjust"
  | "decision-1"
  | "decision-1-pick"
  | "decision-2"
  | "decision-2-pick"
  | "executing"
  | "summary"
  | "revisit-select"
  | "revisit-why"
  | "revisit-suggest"
  | "revisit-pick"
  | "done";

type Mood = "idle" | "thinking" | "asking" | "done";

function phaseToMood(phase: Phase): Mood {
  if (
    phase === "boot" ||
    phase === "decomposing" ||
    phase === "executing" ||
    phase === "revisit-why" ||
    phase === "revisit-suggest"
  )
    return "thinking";
  if (phase === "done") return "done";
  if (phase === "idle") return "idle";
  return "asking";
}

export function Demo() {
  const [phase, setPhase] = useState<Phase>("idle");
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (phase === "idle" || phase === "done") return;

    const delays: Partial<Record<Phase, number>> = {
      boot: 1500,
      decomposing: 2000,
      domains: 2000,
      "domain-adjust": 1500,
      "decision-1": 800,
      "decision-1-pick": 1000,
      "decision-2": 800,
      "decision-2-pick": 1000,
      executing: 2500,
      summary: 2500,
      "revisit-select": 2000,
      "revisit-why": 2000,
      "revisit-suggest": 2000,
      "revisit-pick": 1500,
    };

    const nextPhase: Partial<Record<Phase, Phase>> = {
      boot: "decomposing",
      decomposing: "domains",
      domains: "domain-adjust",
      "domain-adjust": "decision-1",
      "decision-1": "decision-1-pick",
      "decision-1-pick": "decision-2",
      "decision-2": "decision-2-pick",
      "decision-2-pick": "executing",
      executing: "summary",
      summary: "revisit-select",
      "revisit-select": "revisit-why",
      "revisit-why": "revisit-suggest",
      "revisit-suggest": "revisit-pick",
      "revisit-pick": "done",
    };

    const delay = delays[phase];
    const next = nextPhase[phase];
    if (delay && next) {
      const timer = setTimeout(() => setPhase(next), delay);
      return () => clearTimeout(timer);
    }
  }, [phase]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [phase]);

  const mood = phaseToMood(phase);

  return (
    <div className="border border-border rounded-xl bg-surface overflow-hidden">
      {/* Title bar */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-black/20">
        <div className="w-2.5 h-2.5 rounded-full bg-red-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-green-500/50" />
        <span className="text-xs text-muted ml-2 font-mono flex-1">
          defer &quot;build a todo app&quot;
        </span>
        {phase === "idle" ? (
          <button
            onClick={() => setPhase("boot")}
            className="w-6 h-6 flex items-center justify-center rounded-full bg-accent/20 hover:bg-accent/40 transition-colors cursor-pointer"
          >
            <svg
              className="w-3 h-3 text-accent ml-0.5"
              fill="currentColor"
              viewBox="0 0 24 24"
            >
              <path d="M8 5v14l11-7z" />
            </svg>
          </button>
        ) : phase === "done" ? (
          <button
            onClick={() => setPhase("idle")}
            className="text-[10px] text-muted hover:text-foreground transition-colors cursor-pointer"
          >
            replay
          </button>
        ) : null}
      </div>

      {phase === "idle" ? null : (
        <div
          ref={containerRef}
          className="font-mono text-xs max-h-[520px] overflow-y-auto"
        >
          {/* Mascot + content */}
          <div className="flex p-4 gap-6">
            {/* Mascot */}
            <div className="shrink-0 hidden sm:flex items-start pt-2">
              <WebMascot
                mood={mood}
                pixelSize={4}
              />
            </div>

            {/* Content */}
            <div className="flex-1 space-y-2">
              <div>
                <span className="text-orange-500 font-bold">defer</span>
                <span className="text-gray-500"> v0.1.0 | sonnet</span>
              </div>

              {/* Boot / Decomposing */}
              {(phase === "boot" || phase === "decomposing") && (
                <div className="text-orange-500 animate-pulse">
                  Decomposing task...
                </div>
              )}

              {/* Domain priorities */}
              {(phase === "domains" || phase === "domain-adjust") && (
                <div className="space-y-1">
                  <div className="text-orange-500 font-bold">
                    How much do you care about each area?
                  </div>
                  <div className="mt-2 space-y-0.5">
                    {[
                      { name: "Stack", level: "medium", bar: "██░░░", decisions: 3, active: true },
                      {
                        name: "Data",
                        level: phase === "domain-adjust" ? "paranoid" : "medium",
                        bar: phase === "domain-adjust" ? "█████" : "██░░░",
                        decisions: 2,
                        color: phase === "domain-adjust" ? "text-red-400" : "text-yellow-400",
                      },
                      { name: "API", level: "skip", bar: "░░░░░", decisions: 2, color: "text-gray-500" },
                      { name: "UI", level: "medium", bar: "██░░░", decisions: 2 },
                    ].map((d) => (
                      <div key={d.name}>
                        <span className={d.active ? "text-orange-500" : "text-gray-600"}>
                          {d.active ? "> " : "  "}
                        </span>
                        <span className={d.active ? "text-white" : "text-gray-400"}>
                          {d.name.padEnd(18)}
                        </span>
                        <span className={d.color || "text-yellow-400"}>
                          {d.bar} {d.level}
                        </span>
                        <span className="text-gray-600">
                          {"   "}{d.decisions} decisions
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Decision 1 */}
              {(phase === "decision-1" || phase === "decision-1-pick") && (
                <div className="space-y-2">
                  <div>
                    <span className="text-orange-500 font-bold">1/6</span>
                    <span className="text-gray-500">{"  Stack  STACK-001"}</span>
                  </div>
                  <div className="text-white font-bold">
                    Backend language and framework?
                  </div>
                  <div className="space-y-0.5 mt-1">
                    {[
                      { key: "A", label: "Bun with Hono", picked: phase === "decision-1-pick" },
                      { key: "B", label: "Node.js with Express" },
                      { key: "C", label: "Deno with Fresh" },
                      { key: "D", label: "Choose for me", delegated: true },
                    ].map((o) => (
                      <div key={o.key}>
                        <span className={o.picked ? "text-orange-500" : "text-gray-500"}>
                          {o.picked ? " > " : "   "}
                        </span>
                        <span
                          className={
                            o.picked
                              ? "text-orange-500 font-bold"
                              : o.delegated
                                ? "text-purple-400"
                                : "text-white"
                          }
                        >
                          {o.key}) {o.label}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Decision 2 */}
              {(phase === "decision-2" || phase === "decision-2-pick") && (
                <div className="space-y-2">
                  <div>
                    <span className="text-orange-500 font-bold">2/6</span>
                    <span className="text-gray-500">{"  Stack  STACK-002"}</span>
                  </div>
                  <div className="text-white font-bold">Frontend framework?</div>
                  <div className="space-y-0.5 mt-1">
                    {[
                      { key: "A", label: "React with Next.js" },
                      { key: "B", label: "Svelte with SvelteKit", picked: phase === "decision-2-pick" },
                      { key: "C", label: "Choose for me", delegated: true },
                    ].map((o) => (
                      <div key={o.key}>
                        <span className={o.picked ? "text-orange-500" : "text-gray-500"}>
                          {o.picked ? " > " : "   "}
                        </span>
                        <span
                          className={
                            o.picked
                              ? "text-orange-500 font-bold"
                              : o.delegated
                                ? "text-purple-400"
                                : "text-white"
                          }
                        >
                          {o.key}) {o.label}
                        </span>
                      </div>
                    ))}
                  </div>
                  <div className="text-gray-600 mt-1 text-[10px]">
                    {"  "}
                    <span className="text-green-400">✓</span> STACK-001 Backend
                    → Bun with Hono
                  </div>
                </div>
              )}

              {/* Executing */}
              {phase === "executing" && (
                <div className="space-y-1">
                  <div className="text-green-400 font-bold">
                    All 6 decisions answered
                  </div>
                  <div className="mt-1 text-orange-500 animate-pulse">
                    Building...
                  </div>
                </div>
              )}

              {/* Summary with AI choices visible */}
              {phase === "summary" && (
                <div className="space-y-1">
                  <div className="text-white">
                    Created src/index.ts, src/routes/, src/db/
                  </div>
                  <div className="mt-2 text-gray-500 text-[10px] space-y-0.5">
                    <div>
                      {"  "}<span className="text-green-400">✓</span> STACK-001
                      Bun with Hono
                    </div>
                    <div>
                      {"  "}<span className="text-green-400">✓</span> STACK-002
                      Svelte with SvelteKit
                    </div>
                    <div>
                      {"  "}<span className="text-green-400">✓</span> DATA-001
                      SQLite with Drizzle ORM
                    </div>
                    <div>
                      {"  "}<span className="text-purple-400">◆</span> API-001
                      REST with /api prefix
                    </div>
                    <div>
                      {"  "}<span className="text-gray-500">▪</span>{" "}
                      <span className="text-gray-600">
                        NAMI-001 camelCase for routes
                      </span>
                    </div>
                    <div>
                      {"  "}<span className="text-gray-500">▪</span>{" "}
                      <span className="text-gray-600">
                        STRU-001 src/routes/todos.ts
                      </span>
                    </div>
                    <div>
                      {"  "}<span className="text-gray-500">▪</span>{" "}
                      <span className="text-gray-600">
                        ERRO-001 422 for validation errors
                      </span>
                    </div>
                  </div>
                  <div className="mt-1 text-gray-600 text-[10px]">
                    ✓ you decided{"  "}◆ you delegated{"  "}▪ AI chose
                  </div>
                </div>
              )}

              {/* Revisit: user selects an AI choice */}
              {phase === "revisit-select" && (
                <div className="space-y-1">
                  <div className="text-orange-500">
                    {">"} /revisit ERRO-001
                  </div>
                  <div className="mt-2">
                    <span className="text-orange-500 font-bold">ERRO-001</span>
                    <span className="text-gray-500">{"  error"}</span>
                  </div>
                  <div className="text-white font-bold">
                    422 for validation errors
                  </div>
                  <div className="text-gray-600 italic text-[10px]">
                    AI chose this. You can challenge it.
                  </div>
                </div>
              )}

              {/* Revisit: ask why */}
              {phase === "revisit-why" && (
                <div className="space-y-1">
                  <div>
                    <span className="text-orange-500 font-bold">ERRO-001</span>
                    <span className="text-gray-500">{"  error"}</span>
                  </div>
                  <div className="text-white font-bold">
                    422 for validation errors
                  </div>
                  <div className="text-gray-500 text-[10px] mt-1">
                    pressed w
                  </div>
                  <div className="text-gray-400 mt-1 italic">
                    422 is more precise (means &ldquo;I understood your request
                    but the data is wrong&rdquo;). 400 is broader (&ldquo;bad
                    request&rdquo;). Most modern APIs use 422 for validation.
                    Trade-off: some older clients only handle 400.
                  </div>
                </div>
              )}

              {/* Revisit: suggest more */}
              {phase === "revisit-suggest" && (
                <div className="space-y-1">
                  <div>
                    <span className="text-orange-500 font-bold">ERRO-001</span>
                    <span className="text-gray-500">
                      {"  error  "}
                    </span>
                    <span className="text-gray-500 text-[10px]">pressed s</span>
                  </div>
                  <div className="text-white font-bold">
                    HTTP status for validation errors?
                  </div>
                  <div className="space-y-0.5 mt-1">
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">A) 422 Unprocessable Entity</span>
                    </div>
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">B) 400 Bad Request</span>
                    </div>
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">
                        C) 400 with error codes in body
                      </span>
                    </div>
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">
                        D) RFC 7807 Problem Details
                      </span>
                    </div>
                  </div>
                </div>
              )}

              {/* Revisit: pick new option */}
              {phase === "revisit-pick" && (
                <div className="space-y-1">
                  <div>
                    <span className="text-orange-500 font-bold">ERRO-001</span>
                  </div>
                  <div className="text-white font-bold">
                    HTTP status for validation errors?
                  </div>
                  <div className="space-y-0.5 mt-1">
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">A) 422 Unprocessable Entity</span>
                    </div>
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">B) 400 Bad Request</span>
                    </div>
                    <div>
                      <span className="text-gray-500">{"   "}</span>
                      <span className="text-white">
                        C) 400 with error codes in body
                      </span>
                    </div>
                    <div>
                      <span className="text-orange-500">{" > "}</span>
                      <span className="text-orange-500 font-bold">
                        D) RFC 7807 Problem Details
                      </span>
                    </div>
                  </div>
                  <div className="mt-2 text-green-400 text-[10px]">
                    ✓ Updated. Adapting affected code...
                  </div>
                </div>
              )}

              {/* Done */}
              {phase === "done" && (
                <div className="space-y-1">
                  <div className="text-green-400">Done. $0.06</div>
                  <div className="text-gray-600 text-[10px] mt-1">
                    9 decisions: 4 yours, 2 delegated, 3 AI (1 revised)
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Status bar */}
          <div className="px-4 py-2 border-t border-border/50 flex justify-between">
            <span className="text-gray-600 text-[10px]">
              {phase === "decomposing" || phase === "boot"
                ? "thinking"
                : phase === "executing"
                  ? "executing"
                  : phase === "done"
                    ? "done | $0.06"
                    : phase.startsWith("revisit")
                      ? "revisiting ERRO-001"
                      : "asking"}
            </span>
            <span className="text-gray-600 text-[10px]">/help</span>
          </div>

          {/* Input */}
          <div className="px-4 py-2 border-t border-border/50">
            <span className="text-orange-500 font-bold text-xs">
              {"defer > "}
            </span>
            <span className="text-gray-600 animate-pulse">|</span>
          </div>
        </div>
      )}
    </div>
  );
}
