"use client";

import { useState } from "react";
import { DeferPrompt } from "./prompts";
import { CopyButton } from "./copy-button";

const targetIcons: Record<string, string> = {
  universal: "~",
  "claude-code": ">_",
  chatgpt: "G",
  cursor: "{}",
  system: "//",
};

const targetColors: Record<string, string> = {
  universal: "text-cyan-400",
  "claude-code": "text-orange-400",
  chatgpt: "text-emerald-400",
  cursor: "text-purple-400",
  system: "text-yellow-400",
};

export function PromptCard({
  prompt,
  defaultExpanded = false,
}: {
  prompt: DeferPrompt;
  defaultExpanded?: boolean;
}) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div className="group border border-border rounded-xl bg-surface hover:bg-surface-hover hover:border-border transition-all">
      <div className="p-5">
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="flex items-center gap-3">
            <span
              className={`font-mono text-xs font-bold ${targetColors[prompt.target]} bg-white/5 px-2 py-1 rounded`}
            >
              {targetIcons[prompt.target]}
            </span>
            <h3 className="font-semibold text-foreground">{prompt.name}</h3>
          </div>
          <CopyButton text={prompt.prompt} label="Copy" />
        </div>
        <p className="text-sm text-muted mb-3">{prompt.description}</p>
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-accent/70 hover:text-accent transition-colors cursor-pointer"
        >
          {expanded ? "Hide prompt" : "View prompt"}
        </button>
      </div>
      {expanded && (
        <div className="border-t border-border">
          <div className="p-4 bg-black/30 rounded-b-xl">
            <p className="text-xs text-muted mb-3 italic">
              {prompt.instructions}
            </p>
            <pre className="text-xs text-foreground/80 font-mono whitespace-pre-wrap leading-relaxed max-h-80 overflow-y-auto">
              {prompt.prompt}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
