import { CopyButton } from "./copy-button";
import { Demo } from "./demo";
import { MascotLogo } from "./mascot";

export default function Home() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <header className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-accent/8 via-transparent to-transparent" />
        <div className="relative max-w-3xl mx-auto px-6 pt-16 pb-10">
          <div className="mb-10 flex justify-center">
            <MascotLogo />
          </div>

          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight mb-6 leading-[1.1]">
            Zero-Autonomy AI.
            <br />
            <span className="text-accent">Every Decision Is Yours.</span>
          </h1>

          <p className="text-lg text-muted leading-relaxed max-w-2xl mb-4">
            For when you want AI to do the work, not make the calls.
          </p>
          <p className="text-lg leading-relaxed max-w-2xl mb-10">
            <span className="text-foreground">
              Defer decomposes your task into decisions, lets you set how much you care about each domain, then implements everything while you watch and challenge in real-time.
            </span>{" "}
            <span className="text-muted">
              Every decision recorded. Every AI choice visible. Nothing hidden.
            </span>
          </p>

          <div className="flex flex-wrap gap-3">
            <CopyButton
              text="go install github.com/defer-ai/cli@latest"
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
          Works with Claude Code, OpenAI, Groq, Mistral, Together, Ollama, and any OpenAI-compatible provider.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-sm space-y-4">
          <div>
            <p className="text-muted text-xs mb-2">
              # Install
            </p>
            <p className="text-foreground">
              $ go install github.com/defer-ai/cli@latest
            </p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Start a project
            </p>
            <p className="text-foreground">
              $ defer &quot;build a REST API for a todo app&quot;
            </p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Or start conversational mode
            </p>
            <p className="text-foreground">$ defer</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Onboard an existing project
            </p>
            <p className="text-foreground">$ defer scan</p>
          </div>
        </div>

      </section>

      {/* Features */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-8">How it works</h2>
        <div className="space-y-4">
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Natural language in, decisions out
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Describe your project. The agent decomposes it into decisions
              with concrete options, grouped by domain. You set care levels,
              inspect tradeoffs, and override anything. Reference decisions with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                @STA-0001
              </code>{" "}
              or features with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                #auth
              </code>
              .
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Care levels
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Set how much you care about each domain on a scale from
              &ldquo;skip&rdquo; to &ldquo;paranoid.&rdquo; Skip auto-delegates
              everything. Paranoid shows every sub-decision. Same number of
              decisions either way -- you just choose which ones you see.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Full visibility
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Every choice is recorded, whether you made it or the AI did.
              Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                tab
              </code>{" "}
              to see the decision tree. Icons show status:{" "}
              <span className="text-green-400">+</span> yours,{" "}
              <span className="text-gray-400">*</span> auto-decided,{" "}
              <span className="text-yellow-400">o</span> pending.
              IDs colored by impact.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Change anything, anytime
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Change a decision mid-execution and the agent re-implements.
              High-impact changes cascade: switch from Go to Python and
              every Go-specific decision gets invalidated and re-evaluated.
              Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                w
              </code>{" "}
              on any option to see tradeoffs before choosing.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Feature tagging
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Tag decisions with features like &ldquo;auth&rdquo; or
              &ldquo;messaging.&rdquo; Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                g
              </code>{" "}
              to switch between grouping by domain and grouping by feature.
              Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                f
              </code>{" "}
              to find and jump to any decision, category, or feature.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Portable
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Don&apos;t need the CLI? Run{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                defer init cursor
              </code>{" "}
              to drop the defer philosophy into your tool&apos;s config file.
              Works with Claude Code, Cursor, Copilot, and Codex.
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
                  <td className="py-2 pr-4 text-accent">@STA-0001</td>
                  <td className="py-2 pr-4">Stack</td>
                  <td className="py-2 pr-4">Backend language?</td>
                  <td className="py-2 pr-4">Node.js (TypeScript)</td>
                </tr>
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-accent">@DAT-0001</td>
                  <td className="py-2 pr-4">Data</td>
                  <td className="py-2 pr-4">Database?</td>
                  <td className="py-2 pr-4 italic text-muted">
                    DELEGATED: PostgreSQL
                  </td>
                </tr>
              </tbody>
            </table>
            <p className="text-xs font-mono text-gray-500 mb-2">
              ## AI Choices
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
              <tbody className="text-foreground/50">
                <tr className="border-b border-border/50">
                  <td className="py-2 pr-4 text-gray-500">@NAM-0001</td>
                  <td className="py-2 pr-4">Naming</td>
                  <td className="py-2 pr-4">camelCase for routes</td>
                  <td className="py-2 pr-4 text-muted">
                    framework convention
                  </td>
                </tr>
                <tr>
                  <td className="py-2 pr-4 text-gray-500">@ERR-0001</td>
                  <td className="py-2 pr-4">Error</td>
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

      {/* Keybindings */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Shortcuts</h2>
        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-xs space-y-1">
          <p className="text-muted"># Chat</p>
          <p>
            <span className="text-accent">tab</span>
            <span className="text-muted">
              {"      "}Switch between chat and tree
            </span>
          </p>
          <p>
            <span className="text-accent">@ID</span>
            <span className="text-muted">
              {"      "}Reference a decision
            </span>
            {"  "}
            <span className="text-accent">#TAG</span>
            <span className="text-muted"> Reference a feature</span>
          </p>
          <p>
            <span className="text-accent">ctrl+o</span>
            <span className="text-muted">
              {"   "}Expand/collapse message
            </span>
            {"  "}
            <span className="text-accent">ctrl+q</span>
            <span className="text-muted"> Quit</span>
          </p>
          <p className="mt-2 text-muted"># Decision tree</p>
          <p>
            <span className="text-accent">j/k</span>
            <span className="text-muted">
              {"      "}Navigate
            </span>
            {"  "}
            <span className="text-accent">enter</span>
            <span className="text-muted"> Inspect</span>
            {"  "}
            <span className="text-accent">/</span>
            <span className="text-muted"> Filter</span>
            {"  "}
            <span className="text-accent">f</span>
            <span className="text-muted"> Find</span>
            {"  "}
            <span className="text-accent">g</span>
            <span className="text-muted"> Group by feature</span>
          </p>
          <p className="mt-2 text-muted"># Decision detail</p>
          <p>
            <span className="text-accent">enter</span>
            <span className="text-muted">
              {"    "}Confirm
            </span>
            {"  "}
            <span className="text-accent">c</span>
            <span className="text-muted"> Custom answer</span>
            {"  "}
            <span className="text-accent">s</span>
            <span className="text-muted"> Shuffle options</span>
            {"  "}
            <span className="text-accent">w</span>
            <span className="text-muted"> Why?</span>
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
