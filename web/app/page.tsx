import Link from "next/link";
import { prompts } from "./prompts";
import { PromptCard } from "./prompt-card";
import { CopyButton } from "./copy-button";
import { Demo } from "./demo";

export default function Home() {
  const universalPrompt = prompts.find((p) => p.id === "universal")!;
  const otherPrompts = prompts.filter((p) => p.id !== "universal");

  return (
    <div className="min-h-screen">
      {/* Hero */}
      <header className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-accent/8 via-transparent to-transparent" />
        <div className="relative max-w-3xl mx-auto px-6 pt-24 pb-16">
          <Link
            href="/"
            className="font-mono text-sm text-accent tracking-wider mb-8 inline-block"
          >
            defer.sh
          </Link>

          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight mb-6 leading-[1.1]">
            Zero-Autonomy AI.
            <br />
            <span className="text-accent">Every Decision Is Yours.</span>
          </h1>

          <p className="text-lg text-muted leading-relaxed max-w-2xl mb-4">
            AI keeps making choices you didn&apos;t ask for. It picks your tech
            stack, your file structure, your error messages. You don&apos;t find
            out until the code is wrong.
          </p>
          <p className="text-lg leading-relaxed max-w-2xl mb-10">
            <span className="text-foreground">
              Defer makes the AI ask first, then execute.
            </span>{" "}
            <span className="text-muted">
              Slow upfront. Zero rework later.
            </span>
          </p>

          <div className="flex flex-wrap gap-3">
            <a
              href="#get-started"
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent text-background font-medium rounded-lg hover:bg-accent/90 transition-colors text-sm"
            >
              Get started
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </a>
            <a
              href="https://github.com/gabrielmanhaes/defer"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-5 py-2.5 border border-border text-foreground font-medium rounded-lg hover:bg-surface hover:border-border transition-colors text-sm"
            >
              GitHub
            </a>
          </div>
        </div>
      </header>

      {/* Demo */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-3">See it work</h2>
        <p className="text-muted mb-8">
          You say: &ldquo;Build user auth.&rdquo; Without Defer, the AI just
          starts building. With Defer, it asks first.
        </p>
        <Demo />
      </section>

      {/* Why */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-8">Why this works</h2>
        <div className="space-y-4">
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              You can always grant autonomy. You can&apos;t claw back silent decisions.
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Start with the AI asking everything. Then tell it: &ldquo;You decide
              naming conventions.&rdquo; &ldquo;Skip styling questions.&rdquo;
              Autonomy you grant is autonomy you understand. Autonomy the AI assumes
              is a bug you find later.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Decisions are the product
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              The code is the output. The decision record is the value.{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                DECISIONS.md
              </code>{" "}
              gives you a searchable log of why everything is the way it is. No
              more &ldquo;why did we use Redis?&rdquo; six months later.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Structured questions beat blank prompts
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              You shouldn&apos;t have to describe your entire system in one prompt
              just to get started. Give the AI a task. It figures out what it needs
              to know and asks.
            </p>
          </div>
        </div>
      </section>

      {/* Get Started */}
      <section id="get-started" className="max-w-3xl mx-auto px-6 py-20">
        <h2 className="text-2xl font-bold mb-3">Get started</h2>
        <p className="text-muted mb-8">
          Copy the prompt and paste it into your AI tool. Or use the CLI for
          decision tracking across sessions.
        </p>

        {/* Universal prompt, prominent */}
        <div className="mb-6">
          <PromptCard prompt={universalPrompt} defaultExpanded />
        </div>

        {/* Other tools, collapsed */}
        <details className="group">
          <summary className="text-sm text-muted cursor-pointer hover:text-foreground transition-colors mb-4 list-none">
            <span className="group-open:hidden">+ Show prompts for Claude Code, ChatGPT, Cursor, API</span>
            <span className="hidden group-open:inline">- Hide tool-specific prompts</span>
          </summary>
          <div className="space-y-4">
            {otherPrompts.map((prompt) => (
              <PromptCard key={prompt.id} prompt={prompt} />
            ))}
          </div>
        </details>
      </section>

      {/* CLI */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-3">CLI</h2>
        <p className="text-muted mb-8">
          The prompt handles AI behavior. The CLI handles everything else:
          scaffolding, decision tracking, revisiting choices, git integration.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-sm space-y-4">
          <div>
            <p className="text-muted text-xs mb-2">
              # Scaffold Defer into your project
            </p>
            <p className="text-foreground">$ npx @defer/cli init</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # View decisions, grouped by category
            </p>
            <p className="text-foreground">$ npx @defer/cli status</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Change a previous decision
            </p>
            <p className="text-foreground">$ npx @defer/cli revisit D003</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # See what changed since your last decision review
            </p>
            <p className="text-foreground">$ npx @defer/cli diff</p>
          </div>
        </div>

        <div className="mt-6">
          <CopyButton text="npx @defer/cli init" label="Copy install command" />
        </div>
      </section>

      {/* Decision Record */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-3">The decision record</h2>
        <p className="text-muted mb-8">
          Every choice that shaped your project, in one file. Who decided, what
          they decided, and when.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface">
          <p className="text-xs text-muted mb-3 font-mono">DECISIONS.md</p>
          <div className="overflow-x-auto">
            <table className="w-full text-xs font-mono">
              <thead>
                <tr className="text-left text-muted border-b border-border">
                  <th className="pb-2 pr-4">ID</th>
                  <th className="pb-2 pr-4">Category</th>
                  <th className="pb-2 pr-4">Question</th>
                  <th className="pb-2 pr-4">Answer</th>
                </tr>
              </thead>
              <tbody className="text-foreground/70">
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-accent">D001</td>
                  <td className="py-2 pr-4">Auth</td>
                  <td className="py-2 pr-4">Session-based or JWT?</td>
                  <td className="py-2 pr-4">JWT</td>
                </tr>
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-accent">D002</td>
                  <td className="py-2 pr-4">Auth</td>
                  <td className="py-2 pr-4">Token storage?</td>
                  <td className="py-2 pr-4">httpOnly cookies</td>
                </tr>
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-accent">D003</td>
                  <td className="py-2 pr-4">Security</td>
                  <td className="py-2 pr-4">Password hashing?</td>
                  <td className="py-2 pr-4">argon2</td>
                </tr>
                <tr>
                  <td className="py-2 pr-4 text-accent">D004</td>
                  <td className="py-2 pr-4">UX</td>
                  <td className="py-2 pr-4">Failed login message?</td>
                  <td className="py-2 pr-4 italic text-muted">
                    DELEGATED: generic &ldquo;invalid credentials&rdquo;
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="max-w-3xl mx-auto px-6 py-16 border-t border-border">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <span className="font-mono text-sm text-accent">defer.sh</span>
          <div className="flex gap-6 text-sm text-muted">
            <a
              href="https://github.com/gabrielmanhaes/defer"
              className="hover:text-foreground transition-colors"
            >
              GitHub
            </a>
            <a
              href="https://github.com/gabrielmanhaes/defer/blob/main/LICENSE"
              className="hover:text-foreground transition-colors"
            >
              MIT License
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
