"use client";

import { useState, useEffect, useRef } from "react";

const TYPING_SPEED = 20;
const LINE_DELAY = 300;
const PHASE_DELAY = 1200;

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
  | "assumption"
  | "done";

const MASCOT_THINKING = [
  "   ▄██████▄         ▄██████▄",
  "   ██ ▀▄ ██         ██ ▀▄ ██",
  "   ██ ▄▀ ██         ██ ▄▀ ██",
  "   ▀██████▀         ▀██████▀",
  "",
  "            ████████",
];

const MASCOT_ASKING = [
  "   ▄██████▄         ▄██████▄",
  "   ██ ▀▀ ██         ██ ▀▀ ██",
  "   ██ ▀  ██         ██ ▀  ██",
  "   ▀██████▀         ▀██████▀",
  "",
  "            ████████",
];

const MASCOT_DONE = [
  "   ▄██████▄         ▄██████▄",
  "   ██ ◆  ██         ██ ◆  ██",
  "   ▀██████▀         ▀██████▀",
  "",
  "            ████████",
];

export function Demo() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [frame, setFrame] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-advance phases
  useEffect(() => {
    if (phase === "idle") return;

    const delays: Partial<Record<Phase, number>> = {
      boot: 1500,
      decomposing: 2000,
      domains: 2500,
      "domain-adjust": 1500,
      "decision-1": 800,
      "decision-1-pick": 1200,
      "decision-2": 800,
      "decision-2-pick": 1200,
      executing: 2500,
      assumption: 2500,
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
      executing: "assumption",
      assumption: "done",
    };

    const delay = delays[phase];
    const next = nextPhase[phase];
    if (delay && next) {
      const timer = setTimeout(() => setPhase(next), delay);
      return () => clearTimeout(timer);
    }
  }, [phase]);

  // Mascot animation
  useEffect(() => {
    const interval = setInterval(() => setFrame((f) => f + 1), 200);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [phase]);

  if (phase === "idle") {
    return (
      <div className="border border-border rounded-xl bg-surface overflow-hidden">
        <div className="p-8 flex flex-col items-center justify-center gap-4">
          <p className="text-sm text-muted">
            Watch a full Defer session: decomposition, domain priorities,
            decision picking, and assumption tracking.
          </p>
          <button
            onClick={() => setPhase("boot")}
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent text-background font-medium rounded-lg hover:bg-accent/90 transition-colors text-sm cursor-pointer"
          >
            Run demo
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M8 5v14l11-7z" />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  const mascot =
    phase === "decomposing" || phase === "executing"
      ? MASCOT_THINKING
      : phase === "done"
        ? MASCOT_DONE
        : MASCOT_ASKING;

  return (
    <div className="border border-border rounded-xl bg-surface overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-black/20">
        <div className="w-2.5 h-2.5 rounded-full bg-red-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-green-500/50" />
        <span className="text-xs text-muted ml-2 font-mono">
          defer &quot;build a todo app&quot;
        </span>
      </div>
      <div
        ref={containerRef}
        className="font-mono text-xs max-h-[500px] overflow-y-auto"
      >
        {/* Mascot + content side by side */}
        <div className="flex p-4 gap-4">
          {/* Mascot */}
          <div className="text-cyan-400 whitespace-pre leading-tight shrink-0 hidden sm:block">
            {mascot.map((line, i) => (
              <div key={i}>{line}</div>
            ))}
          </div>

          {/* Content */}
          <div className="flex-1 space-y-2">
            <div>
              <span className="text-cyan-400 font-bold">defer</span>
              <span className="text-gray-500"> v0.1.0 | sonnet</span>
            </div>

            {/* Boot */}
            {phase === "boot" && (
              <div className="text-cyan-400">Decomposing task...</div>
            )}

            {/* Decomposing */}
            {phase === "decomposing" && (
              <div className="text-cyan-400 animate-pulse">
                Decomposing task...
              </div>
            )}

            {/* Domain priorities */}
            {(phase === "domains" || phase === "domain-adjust") && (
              <div className="space-y-1">
                <div className="text-cyan-400 font-bold">
                  How much do you care about each area?
                </div>
                <div className="text-gray-500 text-[10px]">
                  ←→ adjust, ↑↓ navigate, enter confirm
                </div>
                <div className="mt-2 space-y-0.5">
                  <div>
                    <span className="text-cyan-400">{"> "}</span>
                    <span className="text-white">
                      {"Stack             "}
                    </span>
                    <span className="text-yellow-400">{"██░░░  medium"}</span>
                    <span className="text-gray-500">{"    3 decisions"}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"  "}</span>
                    <span className="text-gray-400">
                      {"Data              "}
                    </span>
                    <span className={phase === "domain-adjust" ? "text-red-400" : "text-yellow-400"}>
                      {phase === "domain-adjust"
                        ? "█████  paranoid"
                        : "██░░░  medium"}
                    </span>
                    <span className="text-gray-500">{"  2 decisions"}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"  "}</span>
                    <span className="text-gray-400">
                      {"API               "}
                    </span>
                    <span className="text-gray-500">{"░░░░░  skip"}</span>
                    <span className="text-gray-500">{"      2 decisions"}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"  "}</span>
                    <span className="text-gray-400">
                      {"UI                "}
                    </span>
                    <span className="text-yellow-400">{"██░░░  medium"}</span>
                    <span className="text-gray-500">{"    2 decisions"}</span>
                  </div>
                </div>
              </div>
            )}

            {/* Decision 1 */}
            {(phase === "decision-1" || phase === "decision-1-pick") && (
              <div className="space-y-2">
                <div>
                  <span className="text-cyan-400 font-bold">1/6</span>
                  <span className="text-gray-500">
                    {"  Stack  STACK-001"}
                  </span>
                </div>
                <div className="text-white font-bold">
                  Backend language and framework?
                </div>
                <div className="text-gray-500 italic">
                  Determines ecosystem and deployment model
                </div>
                <div className="space-y-0.5 mt-1">
                  <div>
                    <span className={phase === "decision-1-pick" ? "text-cyan-400" : "text-gray-500"}>
                      {phase === "decision-1-pick" ? " > " : "   "}
                    </span>
                    <span className={phase === "decision-1-pick" ? "text-cyan-400 font-bold" : "text-white"}>
                      A) Bun with Hono
                    </span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"   "}</span>
                    <span className="text-white">B) Node.js with Express</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"   "}</span>
                    <span className="text-white">C) Deno with Fresh</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"   "}</span>
                    <span className="text-purple-400">
                      D) Choose for me
                    </span>
                  </div>
                </div>
              </div>
            )}

            {/* Decision 2 */}
            {(phase === "decision-2" || phase === "decision-2-pick") && (
              <div className="space-y-2">
                <div>
                  <span className="text-cyan-400 font-bold">2/6</span>
                  <span className="text-gray-500">
                    {"  Stack  STACK-002"}
                  </span>
                </div>
                <div className="text-white font-bold">
                  Frontend framework?
                </div>
                <div className="space-y-0.5 mt-1">
                  <div>
                    <span className="text-gray-500">{"   "}</span>
                    <span className="text-white">A) React with Next.js</span>
                  </div>
                  <div>
                    <span className={phase === "decision-2-pick" ? "text-cyan-400" : "text-gray-500"}>
                      {phase === "decision-2-pick" ? " > " : "   "}
                    </span>
                    <span className={phase === "decision-2-pick" ? "text-cyan-400 font-bold" : "text-white"}>
                      B) Svelte with SvelteKit
                    </span>
                  </div>
                  <div>
                    <span className="text-gray-500">{"   "}</span>
                    <span className="text-purple-400">
                      C) Choose for me
                    </span>
                  </div>
                </div>
                <div className="text-gray-500 mt-2 text-[10px]">
                  Recent:
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}
                  <span className="text-green-400">✓</span> STACK-001 Backend
                  language? → Bun with Hono
                </div>
              </div>
            )}

            {/* Executing */}
            {phase === "executing" && (
              <div className="space-y-1">
                <div className="text-green-400">
                  ✓ All 6 decisions answered
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}✓ STACK-001 Bun with Hono
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}✓ STACK-002 Svelte with SvelteKit
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}◆ API-001 delegated: REST with /api prefix
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}◆ API-002 delegated: offset pagination
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}✓ DATA-001 SQLite with Drizzle ORM
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}✓ UI-001 Tailwind CSS
                </div>
                <div className="mt-2 text-cyan-400 animate-pulse">
                  Building...
                </div>
              </div>
            )}

            {/* Assumptions */}
            {(phase === "assumption" || phase === "done") && (
              <div className="space-y-1">
                <div className="text-white">
                  Created src/index.ts, src/routes/, src/db/
                </div>
                <div className="mt-2 text-yellow-400 text-[10px]">
                  Assumptions made during execution:
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}
                  <span className="text-yellow-400">⚠</span> NAMI-001
                  camelCase for routes (framework convention)
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}
                  <span className="text-yellow-400">⚠</span> STRU-001
                  src/routes/todos.ts (Hono file-based routing)
                </div>
                <div className="text-gray-500 text-[10px]">
                  {"  "}
                  <span className="text-yellow-400">⚠</span> ERRO-001
                  422 for validation errors (semantically correct)
                </div>
                {phase === "done" && (
                  <div className="mt-2 text-green-400">Done. $0.04</div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Status bar */}
        <div className="px-4 py-2 border-t border-border/50 flex justify-between">
          <span className="text-gray-600 text-[10px]">
            {phase === "decomposing" || phase === "boot"
              ? "thinking"
              : phase === "executing" || phase === "assumption"
                ? "executing"
                : phase === "done"
                  ? "done"
                  : "asking"}
            {phase === "done" && " | 6/6 decisions | 3 assumptions | $0.04"}
          </span>
          <span className="text-gray-600 text-[10px]">/help</span>
        </div>

        {/* Input prompt */}
        <div className="px-4 py-2 border-t border-border/50">
          <span className="text-cyan-400 font-bold text-xs">
            {"defer > "}
          </span>
          <span className="text-gray-600 animate-pulse">|</span>
        </div>
      </div>
    </div>
  );
}
