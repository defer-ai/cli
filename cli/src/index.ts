#!/usr/bin/env node

import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { statusCommand } from "./commands/status.js";
import { revisitCommand } from "./commands/revisit.js";
import { askCommand } from "./commands/ask.js";

const program = new Command();

program
  .name("defer")
  .description("Zero-Autonomy AI. Every decision is yours.")
  .version("0.1.0");

program
  .command("init")
  .description("Scaffold Defer mode config files into your project")
  .argument("[target]", "Target tool: claude-code, cursor, chatgpt, universal, api")
  .action(initCommand);

program
  .command("status")
  .description("View and navigate your decision record")
  .action(statusCommand);

program
  .command("revisit")
  .description("Revisit and change a previous decision")
  .argument("[id]", "Decision ID to revisit (e.g. D001)")
  .action(revisitCommand);

program
  .command("ask")
  .description("Queue a topic for the AI to decompose into decisions")
  .argument("<topic>", "Topic to generate questions about")
  .action(askCommand);

program.parse();
