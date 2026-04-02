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
            <span className="text-accent">Chat First. You Decide.</span>
          </h1>

          <p className="text-lg text-muted leading-relaxed max-w-2xl mb-4">
            A conversation-driven coding agent that asks before it acts.
          </p>
          <p className="text-lg leading-relaxed max-w-2xl mb-10">
            <span className="text-foreground">
              Talk to the AI naturally. Every tool call needs your approval. Tag decisions by feature. Switch to the decision tree when you need the big picture.
            </span>{" "}
            <span className="text-muted">
              9+ providers. Custom skills. Works inside Claude Code, Cursor, Copilot, and Codex.
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
          You chat with the AI. It proposes tool calls. You approve or reject
          each one. Decisions get tagged by feature so you can trace every
          choice.
        </p>
        <Demo />
      </section>

      {/* How it works */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-8">How it works</h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="p-5 border border-border rounded-xl bg-surface">
            <div className="text-accent font-mono text-sm mb-3">01</div>
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Chat naturally
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Describe what you want in plain language. The AI responds in a
              full-screen conversation. Reference previous decisions with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                @STACK-001
              </code>{" "}
              or filter by feature with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                #auth
              </code>
              .
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <div className="text-accent font-mono text-sm mb-3">02</div>
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Approve every action
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              When the AI wants to run a tool&mdash;file write, shell command,
              API call&mdash;a permission overlay appears. You approve, reject,
              or edit before anything executes. Nothing is hidden.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <div className="text-accent font-mono text-sm mb-3">03</div>
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Review the tree
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                tab
              </code>{" "}
              to switch to the decision tree. See every choice organized by
              domain. Press{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                g
              </code>{" "}
              to group by feature. Revisit anything.
            </p>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-8">Features</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Permission system
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Every tool call shows a full approval overlay before execution.
              See the exact command, file path, or API request. Approve, reject,
              or edit inline. The AI never acts without your sign-off.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Feature tagging
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Tag decisions with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                #auth
              </code>
              ,{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                #api
              </code>
              ,{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                #db
              </code>{" "}
              in chat. Reference any decision by ID with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                @STACK-001
              </code>
              . Group the tree view by feature with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                g
              </code>
              .
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              9+ AI providers
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Anthropic, OpenAI, Google, Groq, Mistral, Ollama, OpenRouter,
              xAI, and DeepSeek. Switch models mid-session with{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /model
              </code>
              . Bring your own API key or use local models.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Custom skills and hooks
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Define custom slash commands as skills. Attach lifecycle hooks
              that run before or after tool execution. Extend the agent without
              touching its source.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Portable mode
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Run{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                defer init
              </code>{" "}
              to generate configuration for Claude Code, Cursor, GitHub
              Copilot, or OpenAI Codex. Same zero-autonomy philosophy, native
              to each tool.
            </p>
          </div>
          <div className="p-5 border border-border rounded-xl bg-surface">
            <h3 className="font-semibold text-foreground text-sm mb-2">
              Full visibility
            </h3>
            <p className="text-sm text-muted leading-relaxed">
              Every choice is recorded&mdash;who decided, what they decided,
              and what the AI assumed on its own. Use{" "}
              <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
                /decisions
              </code>{" "}
              to see everything: yours, delegated, and AI-chosen.
            </p>
          </div>
        </div>
      </section>

      {/* Keybindings */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Keyboard-driven</h2>
        <p className="text-muted mb-8">
          Everything has a shortcut. The conversation is the primary view;
          the decision tree is one tab away.
        </p>
        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-xs space-y-1">
          <p className="text-muted"># Navigation</p>
          <p>
            <span className="text-accent">tab</span>
            <span className="text-muted">
              {"              "}Switch between chat and decision tree
            </span>
          </p>
          <p>
            <span className="text-accent">ctrl+q</span>
            <span className="text-muted">
              {"           "}Quit
            </span>
          </p>
          <p>
            <span className="text-accent">f</span>
            <span className="text-muted">
              {"                "}Find / search
            </span>
          </p>
          <p>
            <span className="text-accent">g</span>
            <span className="text-muted">
              {"                "}Group by feature
            </span>
          </p>
          <p className="mt-2 text-muted"># In decision tree</p>
          <p>
            <span className="text-accent">enter</span>
            <span className="text-muted">
              {"            "}Pick an option
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
              {"                "}Type custom answer
            </span>
            {"  "}
            <span className="text-accent">c</span>
            <span className="text-muted"> Change answered</span>
          </p>
          <p className="mt-2 text-muted"># Chat references</p>
          <p>
            <span className="text-accent">@STACK-001</span>
            <span className="text-muted">
              {"        "}Reference a decision by ID
            </span>
          </p>
          <p>
            <span className="text-accent">#auth</span>
            <span className="text-muted">
              {"             "}Tag or filter by feature
            </span>
          </p>
        </div>
      </section>

      {/* Providers */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Works with your provider</h2>
        <p className="text-muted mb-8">
          Swap models mid-session. Use cloud APIs or local inference.
        </p>
        <div className="flex flex-wrap gap-3">
          {[
            "Anthropic",
            "OpenAI",
            "Google",
            "Groq",
            "Mistral",
            "Ollama",
            "OpenRouter",
            "xAI",
            "DeepSeek",
          ].map((name) => (
            <span
              key={name}
              className="px-4 py-2 border border-border rounded-lg bg-surface text-sm text-foreground font-mono"
            >
              {name}
            </span>
          ))}
        </div>
      </section>

      {/* Portable */}
      <section className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Portable philosophy</h2>
        <p className="text-muted mb-8">
          Not everyone uses the same agent. Run{" "}
          <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
            defer init
          </code>{" "}
          to generate zero-autonomy rules for your tool of choice.
        </p>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[
            { name: "Claude Code", cmd: "defer init claude" },
            { name: "Cursor", cmd: "defer init cursor" },
            { name: "Copilot", cmd: "defer init copilot" },
            { name: "Codex", cmd: "defer init codex" },
          ].map((tool) => (
            <div
              key={tool.name}
              className="p-4 border border-border rounded-xl bg-surface text-center"
            >
              <p className="text-sm font-semibold text-foreground mb-1">
                {tool.name}
              </p>
              <p className="text-xs font-mono text-muted">{tool.cmd}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Install */}
      <section id="get-started" className="max-w-3xl mx-auto px-6 py-10">
        <h2 className="text-2xl font-bold mb-3">Get started</h2>
        <p className="text-muted mb-8">
          Defer is a single Go binary. Install it and point it at a task.
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
              # Start a conversation
            </p>
            <p className="text-foreground">
              $ defer &quot;build a REST API for a todo app&quot;
            </p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Or start interactive mode
            </p>
            <p className="text-foreground">$ defer</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2">
              # Generate rules for another tool
            </p>
            <p className="text-foreground">$ defer init cursor</p>
          </div>
        </div>

        <p className="text-sm text-muted mt-4">
          Requires an API key for at least one supported provider. Set{" "}
          <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
            ANTHROPIC_API_KEY
          </code>
          ,{" "}
          <code className="text-accent font-mono text-xs bg-accent/10 px-1 py-0.5 rounded">
            OPENAI_API_KEY
          </code>
          , or any other supported key.
        </p>
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
