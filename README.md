# Defer

## Zero-Autonomy AI. Every Decision Is Yours.

The entire AI industry is racing toward autonomy. More agentic. More autonomous. Less human involvement.

**Defer runs in the opposite direction.**

Defer is a design philosophy where the AI makes **zero decisions that belong to the human**. It doesn't guess. It doesn't assume. It doesn't "pick the most reasonable default." It asks.

The AI's job is not to decide. The AI's job is to **find every decision hidden in a task, surface it, and wait.**

## Quick Start

Go to [defer.sh](https://defer.sh), pick your AI tool, copy the prompt, paste it in. Done.

Or grab one directly:

### Universal (any AI tool)

Copy the [Universal Defer Prompt](https://defer.sh#prompts) and paste it into your AI tool's system instructions or at the start of a conversation.

### Claude Code

Create a `CLAUDE.md` in your project root with the [Claude Code prompt](https://defer.sh#prompts).

### Cursor

Save the [Cursor prompt](https://defer.sh#prompts) as `.cursor/rules/defer.mdc` in your project.

### ChatGPT

Paste the [ChatGPT prompt](https://defer.sh#prompts) into Settings → Personalization → Custom Instructions.

### API / System Prompt

Use the [API prompt](https://defer.sh#prompts) as the `system` parameter in your LLM API calls.

## Why This Works

1. **AI is bad at knowing what you care about.** It's great at identifying *that* a decision exists. It's terrible at knowing which ones matter to you.

2. **Structured questions > blank prompts.** Humans engage far more with AI-generated questions than with open-ended input fields. Asking is a better interface than telling.

3. **Autonomy failures are silent.** When AI makes a wrong autonomous decision, you often don't notice until it's cascaded. When you answer wrong, at least *you* know where to look.

4. **Decisions are the product.** The code is the output. The real value is the decision record — an auditable trail of every choice that shaped the output.

## The Spectrum

```
Full Autonomy ←————————————————→ Full Defer
"Just do it"                     "Ask me everything"
```

You can always move left from Defer (granting autonomy selectively), but you can never fully move right from autonomy (you don't know what the AI decided silently).

## What Defer is NOT

- **Not the AI being dumb.** A Defer-mode AI needs to be *more* intelligent — it has to decompose a task into its full decision tree.
- **Not "asking permission."** It's requirements elicitation. The AI does the analysis. You do the deciding.
- **Not slow by default.** After the first pass, grant autonomy selectively: "You decide naming conventions." "Use your judgment on error handling."

## The Decision Record

The most underrated output of Defer is `DECISIONS.md` — a complete, searchable log of why everything is the way it is.

| ID | Category | Question | Answer | Date |
|----|----------|----------|--------|------|
| D001 | Auth | Session-based or JWT? | JWT | 2026-03-25 |
| D002 | Auth | Token storage? | httpOnly cookies | 2026-03-25 |
| D003 | Security | Password hashing? | argon2 | 2026-03-25 |
| D004 | UX | Failed login message? | DELEGATED — generic | 2026-03-25 |

## Website

The website is a Next.js app in the `web/` directory, deployed to [defer.sh](https://defer.sh) via Vercel.

```bash
cd web
npm install
npm run dev
```

## License

MIT
