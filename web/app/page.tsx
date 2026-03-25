import Link from "next/link";
import { prompts } from "./prompts";
import { PromptCard } from "./prompt-card";
import { CopyButton } from "./copy-button";

export default function Home() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <header className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-accent/8 via-transparent to-transparent" />
        <div className="relative max-w-3xl mx-auto px-6 pt-24 pb-20">
          <Link href="/" className="font-mono text-sm text-accent tracking-wider mb-8 inline-block">
            defer.sh
          </Link>

          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight mb-6 leading-[1.1]">
            Zero-Autonomy AI.
            <br />
            <span className="text-accent">Every Decision Is Yours.</span>
          </h1>

          <p className="text-lg text-muted leading-relaxed max-w-2xl mb-4">
            For when you want AI to do the job but don&apos;t want it to do it wrong.
            For when you don&apos;t want to describe everything in one prompt just to get started.
          </p>
          <p className="text-lg text-muted leading-relaxed max-w-2xl mb-10">
            Defer makes the AI ask you every question it needs answered before it writes a single line.{" "}
            <span className="text-foreground">
              You decide. It executes.
            </span>
          </p>

          <div className="flex flex-wrap gap-3">
            <a
              href="#prompts"
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-accent text-background font-medium rounded-lg hover:bg-accent/90 transition-colors text-sm"
            >
              Get the prompt
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
              <svg
                className="w-4 h-4"
                fill="currentColor"
                viewBox="0 0 24 24"
              >
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
              </svg>
            </a>
          </div>
        </div>
      </header>

      {/* Philosophy */}
      <section className="max-w-3xl mx-auto px-6 py-20">
        <h2 className="text-2xl font-bold mb-8">The Philosophy</h2>

        <div className="space-y-6 text-muted leading-relaxed">
          <p>
            Defer is a design philosophy where the AI makes{" "}
            <strong className="text-foreground">
              zero decisions that belong to the human
            </strong>
            . It doesn&apos;t guess. It doesn&apos;t assume. It doesn&apos;t
            &ldquo;pick the most reasonable default.&rdquo; It asks.
          </p>
          <p>
            The AI&apos;s job is not to decide. The AI&apos;s job is to{" "}
            <strong className="text-foreground">
              find every decision hidden in a task, surface it, and wait.
            </strong>
          </p>
        </div>

        <div className="grid sm:grid-cols-2 gap-4 mt-10">
          {[
            {
              title: "AI is bad at knowing what you care about",
              body: "It's great at identifying that a decision exists. It's terrible at knowing which decisions matter to you. So it surfaces all of them.",
            },
            {
              title: "Structured questions > blank prompts",
              body: "Humans engage far more with AI-generated questions than with open-ended input fields. Asking is a better interface than telling.",
            },
            {
              title: "Autonomy failures are silent",
              body: "When AI makes a wrong autonomous decision, you often don't notice until it's cascaded. When you answer wrong, at least you know where to look.",
            },
            {
              title: "Decisions are the product",
              body: "The code is the output. The real value is the decision record: an auditable trail of every choice that shaped the output.",
            },
          ].map((item) => (
            <div
              key={item.title}
              className="p-5 border border-border rounded-xl bg-surface"
            >
              <h3 className="font-semibold text-foreground text-sm mb-2">
                {item.title}
              </h3>
              <p className="text-sm text-muted leading-relaxed">{item.body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* The Spectrum */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-8">The Spectrum</h2>
        <div className="p-6 border border-border rounded-xl bg-surface font-mono text-sm">
          <div className="flex items-center justify-between mb-2 text-muted">
            <span>Full Autonomy</span>
            <span>Full Defer</span>
          </div>
          <div className="relative h-3 bg-black/40 rounded-full overflow-hidden mb-2">
            <div className="absolute inset-y-0 right-0 w-1/6 bg-accent/60 rounded-full" />
          </div>
          <div className="flex items-center justify-between text-xs text-muted">
            <span>&ldquo;Just do it&rdquo;</span>
            <span className="text-accent">
              &ldquo;Ask me everything&rdquo;
            </span>
          </div>
          <p className="mt-5 text-xs text-muted leading-relaxed font-sans">
            You can always move left from Defer (granting autonomy selectively),
            but you can never fully move right from autonomy. You don&apos;t
            know what the AI decided silently.
          </p>
        </div>
      </section>

      {/* What Defer is NOT */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-8">What Defer is NOT</h2>
        <div className="space-y-4">
          {[
            {
              not: "The AI being dumb or incapable",
              actually:
                "A Defer-mode AI needs to be more intelligent. It has to decompose a task into its full decision tree and sequence questions so they build on each other.",
            },
            {
              not: '"Asking permission"',
              actually:
                "It's requirements elicitation. The AI does the hard analytical work of identifying what needs to be decided. You do the deciding.",
            },
            {
              not: "Slow by default",
              actually:
                'After the first pass, you can say "use my previous answers as defaults" or "I trust you on all styling decisions." Autonomy becomes something you grant, not something you claw back.',
            },
          ].map((item) => (
            <div
              key={item.not}
              className="flex gap-4 p-5 border border-border rounded-xl bg-surface"
            >
              <span className="text-red-400/80 text-lg leading-none mt-0.5">
                &times;
              </span>
              <div>
                <p className="font-semibold text-foreground text-sm mb-1">
                  {item.not}
                </p>
                <p className="text-sm text-muted leading-relaxed">
                  {item.actually}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Prompts */}
      <section id="prompts" className="max-w-3xl mx-auto px-6 py-20">
        <div className="mb-10">
          <h2 className="text-2xl font-bold mb-3">Get the Prompt</h2>
          <p className="text-muted">
            Copy a prompt for your tool. Paste it in. Your AI
            now runs in Defer mode.
          </p>
        </div>

        <div className="space-y-4">
          {prompts.map((prompt) => (
            <PromptCard key={prompt.id} prompt={prompt} />
          ))}
        </div>
      </section>

      {/* CLI */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-3">CLI Companion</h2>
        <p className="text-muted mb-8">
          The prompt handles the AI behavior. The CLI handles the state: track
          decisions, revisit them, queue topics for decomposition.
        </p>

        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-sm space-y-4">
          <div>
            <p className="text-muted text-xs mb-2"># Scaffold Defer into your project</p>
            <p className="text-foreground">$ npx @defer/cli init</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2"># View your decision record</p>
            <p className="text-foreground">$ npx @defer/cli status</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2"># Revisit a previous decision</p>
            <p className="text-foreground">$ npx @defer/cli revisit D003</p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2"># Queue a topic for the AI to ask about</p>
            <p className="text-foreground">$ npx @defer/cli ask &quot;authentication&quot;</p>
          </div>
        </div>

        <div className="mt-6 flex flex-wrap gap-3">
          <CopyButton text="npx @defer/cli init" label="Copy install command" />
        </div>
      </section>

      {/* Decision Record */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-8">The Decision Record</h2>
        <div className="space-y-4 text-muted leading-relaxed">
          <p>
            The most underrated output of Defer is{" "}
            <code className="text-accent font-mono text-sm bg-accent/10 px-1.5 py-0.5 rounded">
              DECISIONS.md
            </code>
            . After a project is built, you have a complete, searchable log of
            why everything is the way it is.
          </p>
          <p>
            No more &ldquo;why did we use Redis here?&rdquo; six months later.
            The answer is in the record.
          </p>
        </div>

        <div className="mt-8 p-5 border border-border rounded-xl bg-surface">
          <p className="text-xs text-muted mb-3 font-mono">
            DECISIONS.md example
          </p>
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

        <div className="mt-6">
          <CopyButton
            text={`# DECISIONS.md

| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|
| D001 | | | | |`}
            label="Copy DECISIONS.md template"
          />
        </div>
      </section>

      {/* Usage Example */}
      <section className="max-w-3xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold mb-8">Usage Example</h2>
        <div className="p-5 border border-border rounded-xl bg-surface font-mono text-sm space-y-4">
          <div>
            <p className="text-muted text-xs mb-1"># You say:</p>
            <p className="text-foreground">
              Implement the user authentication flow from the spec in
              docs/auth-spec.md
            </p>
          </div>
          <div className="border-t border-border/50 pt-4">
            <p className="text-muted text-xs mb-2"># AI responds:</p>
            <div className="space-y-3 text-foreground/80">
              <p className="text-accent font-semibold">## Architecture</p>
              <p>
                <strong>Q1:</strong> Session-based auth or JWT tokens?
              </p>
              <p className="text-muted">
                Options: A) Session-based B) JWT C) Custom
              </p>
              <p>
                <strong>Q2:</strong> Where should tokens be stored?
              </p>
              <p className="text-muted">
                Options: A) httpOnly cookies B) localStorage C) Custom
              </p>
              <p className="text-accent font-semibold mt-4">
                ## Error Handling
              </p>
              <p>
                <strong>Q3:</strong> On failed login, generic or specific
                message?
              </p>
              <p className="text-muted">
                Options: A) &ldquo;Invalid credentials&rdquo; B) Specific
                &ldquo;wrong password&rdquo; C) Custom
              </p>
              <p className="text-accent font-semibold mt-4">## Data</p>
              <p>
                <strong>Q4:</strong> Password hashing algorithm?
              </p>
              <p className="text-muted">
                Options: A) argon2 B) bcrypt C) scrypt D) Custom
              </p>
            </div>
          </div>
          <div className="border-t border-border/50 pt-4 text-muted text-xs">
            # You answer → AI builds decision record → AI executes with full
            context
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="max-w-3xl mx-auto px-6 py-16 border-t border-border">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <span className="font-mono text-sm text-accent">
            defer.sh
          </span>
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
