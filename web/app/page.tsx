import Link from "next/link";
import { CopyButton } from "./copy-button";
import { Demo } from "./demo";
import { HeroMascot } from "./mascot";

export default function Home() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <header className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-accent/8 via-transparent to-transparent" />
        <div className="relative max-w-3xl mx-auto px-6 pt-16 pb-10">
          <div className="flex items-center gap-6 mb-8">
            <HeroMascot />
            <Link
              href="/"
              className="font-mono text-sm text-accent tracking-wider"
            >
              defer.sh
            </Link>
          </div>

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
            <CopyButton
              text='npx @defer/cli "build a todo app"'
              label="Copy install command"
              className="px-5 py-2.5 text-sm font-medium"
            />
            <a
              href="https://github.com/defer-ai/cli"
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
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">See it work</h2>
        <p className="text-muted mb-8">
          You give it a task. It decomposes it into decisions, asks you
          each one, then executes with full context.
        </p>
        <Demo />
      </section>

      {/* Install */}
      <section id="get-started" className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Get started</h2>
        <p className="text-muted mb-8">
          Defer wraps Claude Code. No API key needed, just your existing
          Claude Code installation.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-sm space-y-4">
          <div>
            <p className="text-muted text-xs mb-2">
              # Run directly with any task
            </p>
            <p className="text-foreground">
              $ npx @defer/cli &quot;build a REST API for a todo app&quot;
            </p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Or start interactive mode
            </p>
            <p className="text-foreground">$ npx @defer/cli</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Install globally
            </p>
            <p className="text-foreground">$ npm install -g @defer/cli</p>
            <p className="text-foreground">$ defer &quot;build auth&quot;</p>
          </div>
        </div>

        <p className="text-sm text-muted mt-4">
          Requires Claude Code (
          <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
            npm i -g @anthropic-ai/claude-code && claude login
          </code>
          ).
        </p>
      </section>

      {/* Features */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-8">How it works</h2>
        <div className="space-y-4">
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Domain priorities
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Before answering decisions, set how much you care about each
              domain on a scale from &ldquo;skip&rdquo; to &ldquo;paranoid.&rdquo;
              Skip auto-delegates. Paranoid generates sub-questions. You add
              domains the AI missed.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Assumption tracking
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              During execution, every choice the AI makes is tagged as an
              assumption with reasoning. Variable names, file paths, error
              messages, library versions. Nothing is invisible. Use{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /decisions
              </code>{" "}
              to see everything: your decisions (✓), delegated (◆), and
              assumptions (⚠).
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Profiles
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Save your decisions as a reusable template with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /profile save my-stack
              </code>
              . Next project, apply it with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /profile use my-stack
              </code>{" "}
              and only answer what&apos;s new.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Revisit and undo
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Change your mind with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /revisit STACK-001
              </code>
              . The AI adapts everything downstream. Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                u
              </code>{" "}
              to undo the last answer. Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                w
              </code>{" "}
              on any option to see tradeoffs before choosing.
            </p>
          </div>
        </div>
      </section>

      {/* Decision Record */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">The decision record</h2>
        <p className="text-muted mb-8">
          Every choice that shaped your project. Who decided, what they
          decided, and what the AI assumed on its own.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface">
          <p className="text-xs text-muted mb-3 font-mono">DECISIONS.md</p>
          <div className="overflow-x-auto">
            <p className="text-xs font-mono text-accent mb-2">
              ## Decisions
            </p>
            <table className="w-full text-xs font-mono mb-4">
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
                  <td className="py-2 pr-4 text-accent">STACK-001</td>
                  <td className="py-2 pr-4">Stack</td>
                  <td className="py-2 pr-4">Backend language?</td>
                  <td className="py-2 pr-4">Node.js (TypeScript)</td>
                </tr>
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-accent">DATA-001</td>
                  <td className="py-2 pr-4">Data</td>
                  <td className="py-2 pr-4">Database?</td>
                  <td className="py-2 pr-4 italic text-muted">
                    DELEGATED: PostgreSQL
                  </td>
                </tr>
              </tbody>
            </table>
            <p className="text-xs font-mono text-yellow-400 mb-2">
              ## Assumptions
            </p>
            <table className="w-full text-xs font-mono">
              <thead>
                <tr className="text-left text-muted border-b border-border">
                  <th className="pb-2 pr-4">ID</th>
                  <th className="pb-2 pr-4">Category</th>
                  <th className="pb-2 pr-4">What was decided</th>
                  <th className="pb-2 pr-4">Reasoning</th>
                </tr>
              </thead>
              <tbody className="text-foreground/70">
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-yellow-400">NAMI-001</td>
                  <td className="py-2 pr-4">naming</td>
                  <td className="py-2 pr-4">camelCase for routes</td>
                  <td className="py-2 pr-4 text-muted">
                    framework convention
                  </td>
                </tr>
                <tr>
                  <td className="py-2 pr-4 text-yellow-400">ERRO-001</td>
                  <td className="py-2 pr-4">error</td>
                  <td className="py-2 pr-4">422 for validation</td>
                  <td className="py-2 pr-4 text-muted">
                    more semantically correct
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Commands */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Commands</h2>
        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-xs space-y-1">
          <p className="text-muted"># Slash commands</p>
          <p>
            <span className="text-accent">/decisions</span>
            <span className="text-muted">
              {"          "}View all decisions and assumptions
            </span>
          </p>
          <p>
            <span className="text-accent">/revisit STACK-001</span>
            <span className="text-muted">
              {"  "}Change a specific decision
            </span>
          </p>
          <p>
            <span className="text-accent">/profile save my-stack</span>
            <span className="text-muted">
              {"  "}Save as reusable template
            </span>
          </p>
          <p>
            <span className="text-accent">/profile use my-stack</span>
            <span className="text-muted">
              {"   "}Pre-fill from template
            </span>
          </p>
          <p>
            <span className="text-accent">/model opus</span>
            <span className="text-muted">
              {"            "}Switch model
            </span>
          </p>
          <p>
            <span className="text-accent">/export</span>
            <span className="text-muted">
              {"               "}Markdown table for PRs
            </span>
          </p>
          <p>
            <span className="text-accent">/cost</span>
            <span className="text-muted">
              {"                 "}Session cost and tokens
            </span>
          </p>
          <p>
            <span className="text-accent">/history</span>
            <span className="text-muted">
              {"              "}Previous sessions
            </span>
          </p>
          <p className="mt-2 text-muted"># In decision view</p>
          <p>
            <span className="text-accent">↑↓ enter</span>
            <span className="text-muted">
              {"  "}Pick an option
            </span>
            {"  "}
            <span className="text-accent">u</span>
            <span className="text-muted"> Undo</span>
            {"  "}
            <span className="text-accent">w</span>
            <span className="text-muted"> Explain tradeoffs</span>
          </p>
          <p>
            <span className="text-accent">t</span>
            <span className="text-muted">
              {"        "}Type custom answer
            </span>
            {"  "}
            <span className="text-accent">a</span>
            <span className="text-muted"> Ask about</span>
            {"  "}
            <span className="text-accent">c</span>
            <span className="text-muted"> Change answered</span>
          </p>
        </div>
      </section>

      {/* Footer */}
      <footer className="max-w-3xl mx-auto px-6 py-10 border-t border-border">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <span className="font-mono text-sm text-accent">defer.sh</span>
          <div className="flex gap-6 text-sm text-muted">
            <a
              href="https://github.com/defer-ai/cli"
              className="hover:text-foreground transition-colors"
            >
              GitHub
            </a>
            <a
              href="https://github.com/defer-ai/cli/blob/main/LICENSE"
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
