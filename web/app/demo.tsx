"use client";

import { useState, useEffect, useRef, useCallback } from "react";

type Phase =
  | "idle" | "typing" | "enter"
  | "decomposing" | "care-levels" | "pick-1" | "pick-2"
  | "detail" | "detail-why" | "detail-shuffle" | "detail-ask" | "detail-answer"
  | "chat-ref" | "chat-ref-response"
  | "executing" | "mid-pause" | "mid-pick" | "resuming" | "done";

const PHASE_ORDER: Phase[] = [
  "idle", "typing", "enter", "decomposing", "care-levels", "pick-1", "pick-2",
  "detail", "detail-why", "detail-shuffle", "detail-ask", "detail-answer",
  "chat-ref", "chat-ref-response",
  "executing", "mid-pause", "mid-pick", "resuming", "done",
];

function ord(current: Phase, target: Phase) {
  return PHASE_ORDER.indexOf(current) >= PHASE_ORDER.indexOf(target);
}

const PROMPT = "build a todo app with auth";
const SPIN = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const TOOLS_1 = ["Write(src/auth/jwt.ts)", "Write(src/db/schema.ts)", "Bash(npm install)"];
const TOOLS_2 = ["Write(src/auth/session.ts)", "Write(src/middleware/auth.ts)", "Bash(npm test)"];
const BTN = "px-3 py-1.5 text-[11px] font-mono bg-orange-500/15 text-orange-500 border border-orange-500/30 rounded hover:bg-orange-500/25 transition-colors cursor-pointer";
const BTN_SM = "px-2 py-1 text-[10px] font-mono bg-orange-500/15 text-orange-500 border border-orange-500/30 rounded hover:bg-orange-500/25 transition-colors cursor-pointer";

function ToolLine({ tool }: { tool: string }) {
  const [name, rest] = tool.split("(");
  return (
    <div className="text-gray-400">
      <span className="text-orange-500">{"● "}</span>
      <span className="font-bold text-white">{name}</span>
      <span className="text-gray-500">({rest}</span>
    </div>
  );
}

function OptionList({ options, delegate, onPick }: {
  options: string[]; delegate?: string; onPick: (i: number) => void;
}) {
  return (
    <div className="space-y-1 mt-1">
      {options.map((opt, i) => {
        const isDel = opt === delegate;
        return (
          <div key={opt} onClick={() => onPick(i)} className="cursor-pointer group">
            <span className="text-gray-500 group-hover:text-orange-500">{"   "}</span>
            <span className={`transition-colors ${isDel ? "text-purple-400 group-hover:text-purple-300" : "text-white group-hover:text-orange-500"}`}>
              {String.fromCharCode(65 + i)}) {opt}
            </span>
          </div>
        );
      })}
    </div>
  );
}

type CareLevel = "auto" | "review";
interface DecisionState { question: string; careLevel: CareLevel; domain: string }
const INIT_DECISIONS: DecisionState[] = [
  { question: "Backend framework?", careLevel: "auto", domain: "Stack" },
  { question: "Frontend framework?", careLevel: "auto", domain: "Stack" },
  { question: "Authentication method?", careLevel: "review", domain: "Auth" },
  { question: "Password storage?", careLevel: "review", domain: "Auth" },
  { question: "Database?", careLevel: "auto", domain: "Data" },
  { question: "Migration tool?", careLevel: "auto", domain: "Data" },
];

const AUTH_OPTS = ["JWT tokens", "Session-based", "OAuth2"];
const SHUFFLE_OPTS = ["Passport.js", "Auth0 SDK", "Lucia Auth"];

export function Demo() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [typed, setTyped] = useState("");
  const [spinIdx, setSpinIdx] = useState(0);
  const [decisions, setDecisions] = useState<DecisionState[]>(INIT_DECISIONS);
  const [pick1, setPick1] = useState<number | null>(null);
  const [pick2, setPick2] = useState<number | null>(null);
  const [midPick, setMidPick] = useState<number | null>(null);
  const [toolIdx, setToolIdx] = useState(0);
  const [toolIdx2, setToolIdx2] = useState(0);
  const [decisionVisible, setDecisionVisible] = useState(0);
  const [shuffled, setShuffled] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const typingRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Spinner
  useEffect(() => {
    if (phase === "decomposing" || phase === "executing" || phase === "resuming") {
      const iv = setInterval(() => setSpinIdx((i) => (i + 1) % SPIN.length), 80);
      return () => clearInterval(iv);
    }
  }, [phase]);

  // Auto-advance decomposing → care-levels
  useEffect(() => {
    if (phase === "decomposing") {
      const t = setTimeout(() => setPhase("care-levels"), 3000);
      return () => clearTimeout(t);
    }
  }, [phase]);

  // Reveal decisions one by one during care-levels
  const totalDecisions = decisions.length;
  useEffect(() => {
    if (phase !== "care-levels") return;
    setDecisionVisible(0);
    let count = 0;
    const iv = setInterval(() => {
      count++;
      setDecisionVisible(count);
      if (count >= totalDecisions) clearInterval(iv);
    }, 250);
    return () => clearInterval(iv);
  }, [phase, totalDecisions]);

  // Tool call animations — first batch
  useEffect(() => {
    if (phase !== "executing") return;
    setToolIdx(0); let i = 0;
    const iv = setInterval(() => { i++; if (i < TOOLS_1.length) setToolIdx(i); else { clearInterval(iv); setTimeout(() => setPhase("mid-pause"), 800); } }, 700);
    return () => clearInterval(iv);
  }, [phase]);

  // Tool call animations — second batch (after mid-pause resolution)
  useEffect(() => {
    if (phase !== "resuming") return;
    setToolIdx2(0); let i = 0;
    const iv = setInterval(() => { i++; if (i < TOOLS_2.length) setToolIdx2(i); else { clearInterval(iv); setTimeout(() => setPhase("done"), 800); } }, 700);
    return () => clearInterval(iv);
  }, [phase]);

  // Scroll
  useEffect(() => {
    requestAnimationFrame(() => containerRef.current?.scrollTo({ top: containerRef.current.scrollHeight, behavior: "smooth" }));
  }, [phase, typed, toolIdx, toolIdx2, pick1, pick2, midPick, decisionVisible, shuffled]);

  // Typing
  const startTyping = useCallback(() => {
    setPhase("typing"); setTyped("");
    let i = 0;
    function next() {
      if (i < PROMPT.length) { setTyped(PROMPT.slice(0, i + 1)); i++; typingRef.current = setTimeout(next, 40 + Math.random() * 40); }
      else setPhase("enter");
    }
    typingRef.current = setTimeout(next, 300);
  }, []);

  useEffect(() => () => { if (typingRef.current) clearTimeout(typingRef.current); }, []);

  const toggleCareLevel = (idx: number) => setDecisions((ds) => ds.map((d, i) => i === idx ? { ...d, careLevel: d.careLevel === "auto" ? "review" : "auto" } : d));
  const reviewDecisions = decisions.filter((d) => d.careLevel === "review");
  const reviewCount = reviewDecisions.length;

  const reset = () => {
    setPhase("idle"); setTyped(""); setPick1(null); setPick2(null);
    setMidPick(null); setToolIdx(0); setToolIdx2(0); setDecisionVisible(0);
    setShuffled(false); setDecisions(INIT_DECISIONS);
  };

  const active = phase !== "idle";
  const isDetail = phase.startsWith("detail");

  // Auto-resolved answers for "auto" decisions
  const autoAnswers: Record<string, string> = {
    "Backend framework?": "Bun with Hono",
    "Frontend framework?": "React with Vite",
    "Database?": "SQLite with Drizzle",
    "Migration tool?": "Drizzle Kit",
  };

  // Group decisions by domain for display
  const domainGroups: { domain: string; items: (DecisionState & { globalIdx: number })[] }[] = [];
  decisions.forEach((d, i) => {
    const last = domainGroups[domainGroups.length - 1];
    if (last && last.domain === d.domain) {
      last.items.push({ ...d, globalIdx: i });
    } else {
      domainGroups.push({ domain: d.domain, items: [{ ...d, globalIdx: i }] });
    }
  });

  return (
    <div className="border border-border rounded-xl bg-surface overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-black/20">
        <span className="text-xs text-muted font-mono flex-1">defer interactive demo</span>
        {phase === "done" && <button onClick={reset} className="text-[10px] text-muted hover:text-foreground transition-colors cursor-pointer font-mono">replay</button>}
      </div>

      <div ref={containerRef} className="font-mono text-xs max-h-[600px] overflow-y-auto">
        <div className="p-4 space-y-3">
          {/* Header line */}
          <div>
            <span className="text-orange-500 font-bold">defer</span>
            <span className="text-gray-500"> | sonnet</span>
          </div>

          {/* Layout hint */}
          <div className="text-gray-600 text-[10px]">
            tree (left) · chat (right top) · resolver (right bottom)
          </div>

          {/* Input line */}
          <div>
            <span className="text-orange-500">{"> "}</span>
            {active ? (
              <><span className="text-white">{typed}</span>{phase === "typing" && <span className="text-gray-500 animate-pulse">|</span>}</>
            ) : <span className="text-gray-500 animate-pulse">|</span>}
          </div>

          {phase === "idle" && <div><button onClick={startTyping} className={BTN}>type command</button></div>}
          {phase === "enter" && <div><button onClick={() => setPhase("decomposing")} className={BTN}>press enter</button></div>}

          {/* Decomposing */}
          {ord(phase, "decomposing") && (
            <div className="space-y-1 border-t border-border/30 pt-3">
              <div>{phase === "decomposing"
                ? <span className="text-orange-500">{SPIN[spinIdx]} Decomposing task into decisions...</span>
                : <span className="text-gray-600">{"  "}Decomposed into {totalDecisions} decisions</span>}
              </div>
            </div>
          )}

          {/* Care levels + decision list */}
          {ord(phase, "care-levels") && (
            <div className="space-y-1 border-t border-border/30 pt-3">
              <div className="text-white font-bold text-[11px] mb-2">
                Decisions <span className="font-normal text-gray-500">— {totalDecisions} found, set care levels</span>
              </div>

              {domainGroups.map((group) => (
                <div key={group.domain} className="mb-2">
                  <div className="text-gray-500 text-[10px] mb-0.5">{group.domain}</div>
                  {group.items.map((d) => {
                    const visible = d.globalIdx < decisionVisible || ord(phase, "pick-1");
                    const isReview = d.careLevel === "review";
                    const autoAnswer = autoAnswers[d.question];
                    const answered = !isReview && autoAnswer;
                    // For review decisions, show answer after picked
                    const reviewAnswer = d.question === "Authentication method?" && pick1 !== null ? AUTH_OPTS[pick1]
                      : d.question === "Password storage?" && pick2 !== null ? ["bcrypt", "argon2"][pick2]
                      : null;

                    return (
                      <div key={d.question} className={`transition-opacity duration-500 flex items-center gap-2 ${visible ? "opacity-100" : "opacity-0"}`}>
                        <div className="pl-2 flex-1">
                          <span className={isReview ? "text-yellow-400" : "text-gray-500"}>{isReview ? "○ " : "▪ "}</span>
                          <span className={isReview ? (reviewAnswer ? "text-green-400" : "text-yellow-400") : "text-gray-500"}>
                            {d.question}
                          </span>
                          {answered && <span className="text-gray-600"> {autoAnswer}</span>}
                          {reviewAnswer && <span className="text-gray-600"> {reviewAnswer}</span>}
                        </div>
                        {phase === "care-levels" && visible && (
                          <div className="flex gap-1 shrink-0">
                            <button onClick={() => toggleCareLevel(d.globalIdx)} className={`px-2 py-0.5 text-[10px] rounded font-mono cursor-pointer transition-colors ${!isReview ? "bg-gray-700 text-white" : "bg-transparent text-gray-600 hover:text-gray-400"}`}>auto</button>
                            <button onClick={() => toggleCareLevel(d.globalIdx)} className={`px-2 py-0.5 text-[10px] rounded font-mono cursor-pointer transition-colors ${isReview ? "bg-orange-500/20 text-orange-500" : "bg-transparent text-gray-600 hover:text-gray-400"}`}>review</button>
                          </div>
                        )}
                        {phase !== "care-levels" && visible && (
                          <span className={`text-[10px] shrink-0 ${isReview ? "text-orange-500" : "text-gray-600"}`}>{d.careLevel}</span>
                        )}
                      </div>
                    );
                  })}
                </div>
              ))}

              {phase === "care-levels" && decisionVisible >= totalDecisions && (
                <button onClick={() => reviewCount > 0 ? setPhase("pick-1") : setPhase("executing")} className={BTN + " mt-2"}>
                  confirm care levels{reviewCount > 0 && ` (${reviewCount} to review)`}
                </button>
              )}
            </div>
          )}

          {/* Pick auth method (resolver) */}
          {ord(phase, "pick-1") && pick1 === null && (
            <div className="space-y-2 border-t border-border/30 pt-3">
              <div className="text-gray-500 text-[10px]">resolver</div>
              <div><span className="text-orange-500 font-bold">1/{reviewCount}</span><span className="text-gray-500">{"  Auth"}</span></div>
              <div className="text-white font-bold">Authentication method?</div>
              <OptionList options={["JWT tokens", "Session-based", "OAuth2", "Choose for me"]} delegate="Choose for me" onPick={(i) => setPick1(i === 3 ? 0 : i)} />
            </div>
          )}

          {/* Pick password storage (resolver) */}
          {ord(phase, "pick-1") && pick1 !== null && pick2 === null && !isDetail && phase !== "executing" && (
            <div className="space-y-2 border-t border-border/30 pt-3">
              <div className="text-green-400 text-[10px] mb-1">{"✓ "}Authentication method: {AUTH_OPTS[pick1]}</div>
              <div className="text-gray-500 text-[10px]">resolver</div>
              <div><span className="text-orange-500 font-bold">2/{reviewCount}</span><span className="text-gray-500">{"  Auth"}</span></div>
              <div className="text-white font-bold">Password storage?</div>
              <OptionList options={["bcrypt", "argon2", "Choose for me"]} delegate="Choose for me" onPick={(i) => { setPick2(i === 2 ? 1 : i); setPhase("detail"); }} />
            </div>
          )}

          {/* Decision detail view */}
          {isDetail && (
            <div className="space-y-3 border-t border-border/30 pt-3">
              <div className="text-green-400 text-[10px]">{"✓ "}Password storage: {["bcrypt", "argon2"][pick2!]}</div>
              <div className="text-white font-bold text-[11px]">Decision detail: Authentication method</div>
              <div className="bg-black/30 rounded p-3 space-y-1.5 border border-border/20">
                <div className="text-gray-400">Domain: <span className="text-white">Auth</span></div>
                <div className="text-gray-400">Question: <span className="text-white">Authentication method?</span></div>
                <div className="text-gray-400">Selected: <span className="text-green-400">{AUTH_OPTS[pick1!]}</span></div>
                <div className="text-gray-400">Confidence: <span className="text-orange-500">medium</span> — depends on project requirements</div>
              </div>

              {/* Action buttons */}
              {phase === "detail" && (
                <div className="flex gap-2">
                  <button onClick={() => setPhase("detail-why")} className={BTN_SM}>[w] why?</button>
                  <button onClick={() => { setShuffled(false); setPhase("detail-shuffle"); }} className={BTN_SM}>[s] shuffle</button>
                  <button onClick={() => setPhase("detail-ask")} className={BTN_SM}>[a] ask</button>
                  <button onClick={() => setPhase("chat-ref")} className={BTN_SM + " ml-auto"}>continue</button>
                </div>
              )}

              {/* Why explanation */}
              {phase === "detail-why" && (
                <div className="space-y-2">
                  <div className="bg-black/30 rounded p-3 border border-border/20 space-y-1">
                    <div className="text-orange-500 font-bold text-[10px]">Tradeoffs</div>
                    <div className="text-gray-400">
                      <span className="text-white">JWT</span> — stateless, scales horizontally, but hard to revoke.
                    </div>
                    <div className="text-gray-400">
                      <span className="text-white">Session</span> — server-side control, easy revoke, but needs shared store.
                    </div>
                    <div className="text-gray-400">
                      <span className="text-white">OAuth2</span> — delegated auth, good for third-party, more complex setup.
                    </div>
                  </div>
                  <button onClick={() => setPhase("chat-ref")} className={BTN}>continue</button>
                </div>
              )}

              {/* Shuffle */}
              {phase === "detail-shuffle" && (
                <div className="space-y-2">
                  <div className="text-gray-500 text-[10px]">Alternative options:</div>
                  {SHUFFLE_OPTS.map((opt, i) => (
                    <div key={opt} className="text-white pl-2 animate-[fadeSlide_0.3s_ease-out_both]" style={{ animationDelay: `${i * 150}ms` }}>
                      <span className="text-orange-500">{String.fromCharCode(65 + i)})</span> {opt}
                    </div>
                  ))}
                  <button onClick={() => setPhase("chat-ref")} className={BTN}>continue</button>
                </div>
              )}

              {/* Ask */}
              {phase === "detail-ask" && (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <span className="text-orange-500">?</span>
                    <span className="text-gray-500">type a question:</span>
                  </div>
                  <div className="bg-black/30 rounded px-3 py-2 border border-border/20 text-white cursor-pointer" onClick={() => setPhase("detail-answer")}>
                    Can I switch to sessions later?<span className="text-gray-500 animate-pulse">|</span>
                  </div>
                  <button onClick={() => setPhase("detail-answer")} className={BTN_SM}>send</button>
                </div>
              )}

              {phase === "detail-answer" && (
                <div className="space-y-2">
                  <div className="text-gray-500 text-[10px]">Q: Can I switch to sessions later?</div>
                  <div className="bg-black/30 rounded p-3 border border-border/20 text-gray-300">
                    Yes. The auth module is isolated in <span className="text-orange-500">src/auth/</span>. Switching from JWT to sessions means replacing the token verify middleware with a session lookup. The route handlers stay the same.
                  </div>
                  <button onClick={() => setPhase("chat-ref")} className={BTN}>continue</button>
                </div>
              )}
            </div>
          )}

          {/* Chat references */}
          {phase === "chat-ref" && (
            <div className="space-y-2 border-t border-border/30 pt-3">
              <div className="text-gray-500 text-[10px]">chat — reference decisions with @ and features with #</div>
              <div className="bg-black/30 rounded p-3 border border-border/20 space-y-2">
                <div>
                  <span className="text-white font-bold">{"> "}</span>
                  <span className="text-white">what did we pick for </span>
                  <span className="text-orange-500 font-bold">@AUT-0001</span>
                  <span className="text-white">? how does </span>
                  <span className="text-orange-500 font-bold">#auth</span>
                  <span className="text-white"> affect the data layer?</span>
                </div>
              </div>
              <button onClick={() => setPhase("chat-ref-response")} className={BTN}>send</button>
            </div>
          )}
          {phase === "chat-ref-response" && (
            <div className="space-y-2 border-t border-border/30 pt-3">
              <div className="bg-black/30 rounded p-3 border border-border/20 space-y-2">
                <div>
                  <span className="text-white font-bold">{"> "}</span>
                  <span className="text-white">what did we pick for </span>
                  <span className="text-orange-500 font-bold">@AUT-0001</span>
                  <span className="text-white">? how does </span>
                  <span className="text-orange-500 font-bold">#auth</span>
                  <span className="text-white"> affect the data layer?</span>
                </div>
                <div className="text-gray-400 mt-2">
                  <span className="text-orange-500">@AUT-0001</span> is set to <span className="text-white">JWT tokens</span>.
                  The <span className="text-orange-500">#auth</span> feature affects 2 data decisions:
                  the users table needs a <code className="text-orange-500 bg-orange-500/10 px-1 rounded text-[10px]">password_hash</code> column (bcrypt),
                  and sessions are stateless (no session table needed).
                </div>
              </div>
              <button onClick={() => setPhase("executing")} className={BTN}>start implementation</button>
            </div>
          )}

          {/* Execution — agent implementing */}
          {ord(phase, "executing") && pick2 !== null && !isDetail && (
            <div className="space-y-1 border-t border-border/30 pt-3">
              <div className="text-green-400 font-bold">All initial decisions resolved. Agent implementing...</div>
            </div>
          )}
          {ord(phase, "executing") && !isDetail && (
            <div className="space-y-1 mt-1">{TOOLS_1.slice(0, toolIdx + 1).map((t, i) => <ToolLine key={i} tool={t} />)}</div>
          )}

          {/* Mid-execution: new decision discovered inline */}
          {ord(phase, "mid-pause") && (
            <div className="mt-2">
              <div className="text-yellow-400 font-bold">{"● "}Paused — new decision discovered during implementation</div>
              <div className="text-gray-500 text-[10px] mt-1">resolver</div>
            </div>
          )}
          {phase === "mid-pause" && (
            <div className="space-y-2 mt-1">
              <div className="text-white font-bold">Error response format?</div>
              <OptionList options={["JSON {error, message}", "RFC 7807 Problem Details", "Choose for me"]} delegate="Choose for me" onPick={(i) => { setMidPick(i === 2 ? 0 : i); setPhase("resuming"); }} />
            </div>
          )}

          {/* Resume after mid-pause resolution */}
          {ord(phase, "resuming") && midPick !== null && (
            <div className="space-y-1 mt-1">
              <div className="text-green-400 text-[10px]">{"✓ "}Error response format: {["JSON {error, message}", "RFC 7807 Problem Details"][midPick]}</div>
              <div className="text-green-400 font-bold">Continuing implementation...</div>
            </div>
          )}
          {ord(phase, "resuming") && (
            <div className="space-y-1 mt-1">{TOOLS_2.slice(0, toolIdx2 + 1).map((t, i) => <ToolLine key={i} tool={t} />)}</div>
          )}

          {/* Done */}
          {phase === "done" && (
            <div className="space-y-2 border-t border-border/30 pt-3">
              <div className="text-green-400 font-bold">{"✓ "}Implementation complete</div>
              <div className="text-gray-500 text-[10px]">tree (left) shows all decisions · chat (right top) for follow-ups · resolver (right bottom) when pending</div>
              <button onClick={reset} className={BTN + " mt-1"}>replay</button>
            </div>
          )}
        </div>
      </div>

      {/* Status bar */}
      <div className="px-4 py-2 border-t border-border/50 flex justify-between">
        <span className="text-gray-600 text-[10px] font-mono">
          {phase === "idle" ? "ready"
            : phase === "typing" || phase === "enter" ? "input"
            : phase === "decomposing" ? "decomposing"
            : phase === "executing" || phase === "resuming" ? "executing"
            : phase === "done" ? "done"
            : phase === "mid-pause" ? "waiting"
            : isDetail ? "inspecting"
            : ord(phase, "pick-1") && (pick1 === null || pick2 === null) ? "waiting"
            : "resolving"}
        </span>
        <span className="text-gray-600 text-[10px] font-mono">←→ cycle pending | click to interact</span>
      </div>

      <style jsx>{`
        @keyframes fadeSlide {
          from { opacity: 0; transform: translateX(-8px); }
          to { opacity: 1; transform: translateX(0); }
        }
      `}</style>
    </div>
  );
}
