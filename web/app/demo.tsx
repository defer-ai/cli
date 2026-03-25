"use client";

import { useState, useEffect, useRef } from "react";

const TYPING_SPEED = 25;
const LINE_DELAY = 400;

interface DemoLine {
  text: string;
  className: string;
  delay?: number;
}

const userPrompt: DemoLine = {
  text: "> Build user authentication for my app",
  className: "text-foreground font-semibold",
  delay: 0,
};

const aiLines: DemoLine[] = [
  {
    text: "Before writing any code, I need to understand your requirements.",
    className: "text-muted",
  },
  { text: "", className: "", delay: 200 },
  { text: "## Architecture", className: "text-accent font-semibold" },
  {
    text: "Q1: Session-based auth or JWT tokens?",
    className: "text-foreground",
  },
  {
    text: "    Options: A) Session-based  B) JWT  C) Choose for me",
    className: "text-muted",
  },
  {
    text: "    Context: Affects your entire auth infrastructure and scaling model.",
    className: "text-muted/60 italic",
  },
  { text: "", className: "", delay: 200 },
  {
    text: "Q2: Where should tokens/sessions be stored?",
    className: "text-foreground",
  },
  {
    text: "    Options: A) httpOnly cookies  B) localStorage  C) Choose for me",
    className: "text-muted",
  },
  { text: "", className: "", delay: 200 },
  { text: "## Security", className: "text-accent font-semibold" },
  {
    text: "Q3: Password hashing algorithm?",
    className: "text-foreground",
  },
  {
    text: "    Options: A) argon2  B) bcrypt  C) scrypt  D) Choose for me",
    className: "text-muted",
  },
  { text: "", className: "", delay: 200 },
  { text: "## Error Handling", className: "text-accent font-semibold" },
  {
    text: "Q4: Failed login message: generic or specific?",
    className: "text-foreground",
  },
  {
    text: '    Options: A) Generic "invalid credentials"  B) Specific "wrong password"  C) Choose for me',
    className: "text-muted",
  },
  {
    text: "    Context: Generic is more secure. Specific is better UX.",
    className: "text-muted/60 italic",
  },
  { text: "", className: "", delay: 200 },
  {
    text: "Answer these and I'll show the full decision record before writing any code.",
    className: "text-muted italic",
  },
];

export function Demo() {
  const [visibleLines, setVisibleLines] = useState(0);
  const [started, setStarted] = useState(false);
  const [typingLine, setTypingLine] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const allLines = [userPrompt, { text: "", className: "", delay: 300 }, ...aiLines];

  useEffect(() => {
    if (!started) return;

    if (visibleLines >= allLines.length) return;

    const currentLine = allLines[visibleLines];
    const delay = currentLine.delay ?? LINE_DELAY;

    if (currentLine.text === "") {
      const timer = setTimeout(() => {
        setVisibleLines((v) => v + 1);
      }, delay);
      return () => clearTimeout(timer);
    }

    setIsTyping(true);
    let charIndex = 0;
    setTypingLine("");

    const typeChar = () => {
      if (charIndex < currentLine.text.length) {
        setTypingLine(currentLine.text.slice(0, charIndex + 1));
        charIndex++;
        setTimeout(typeChar, TYPING_SPEED);
      } else {
        setIsTyping(false);
        setTimeout(() => {
          setVisibleLines((v) => v + 1);
          setTypingLine("");
        }, delay);
      }
    };

    const startTimer = setTimeout(typeChar, visibleLines === 0 ? 0 : 100);
    return () => clearTimeout(startTimer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [started, visibleLines]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [visibleLines, typingLine]);

  if (!started) {
    return (
      <div className="border border-border rounded-xl bg-surface overflow-hidden">
        <div className="p-8 flex flex-col items-center justify-center gap-4">
          <p className="text-sm text-muted">
            Watch the AI decompose a task into decisions before acting.
          </p>
          <button
            onClick={() => setStarted(true)}
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent text-background font-medium rounded-lg hover:bg-accent/90 transition-colors text-sm cursor-pointer"
          >
            Run demo
            <svg
              className="w-4 h-4"
              fill="currentColor"
              viewBox="0 0 24 24"
            >
              <path d="M8 5v14l11-7z" />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-xl bg-surface overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-black/20">
        <div className="w-2.5 h-2.5 rounded-full bg-red-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/50" />
        <div className="w-2.5 h-2.5 rounded-full bg-green-500/50" />
        <span className="text-xs text-muted ml-2 font-mono">
          claude &ldquo;Build user auth&rdquo;
        </span>
      </div>
      <div
        ref={containerRef}
        className="p-5 font-mono text-sm max-h-96 overflow-y-auto space-y-1"
      >
        {allLines.slice(0, visibleLines).map((line, i) => (
          <p key={i} className={line.className}>
            {line.text || "\u00A0"}
          </p>
        ))}
        {isTyping && visibleLines < allLines.length && (
          <p className={allLines[visibleLines].className}>
            {typingLine}
            <span className="animate-pulse">|</span>
          </p>
        )}
      </div>
    </div>
  );
}
