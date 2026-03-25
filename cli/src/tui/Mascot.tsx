import React from "react";
import { Box, Text } from "ink";

export type MascotMood =
  | "idle"
  | "thinking"
  | "asking"
  | "answering"
  | "executing"
  | "done"
  | "error";

interface Face {
  eyes: string;
  mouth: string;
  blush?: boolean;
}

const FACES: Record<MascotMood, Face> = {
  idle: { eyes: "-   -", mouth: "  ~  " },
  thinking: { eyes: "◠   ◠", mouth: "  o  " },
  asking: { eyes: "◉   ◉", mouth: "  ?  " },
  answering: { eyes: "◠   ◠", mouth: "  ◡  ", blush: true },
  executing: { eyes: "▪   ▪", mouth: " ─── " },
  done: { eyes: "^   ^", mouth: "  ◡  ", blush: true },
  error: { eyes: "x   x", mouth: "  _  " },
};

const LABELS: Record<MascotMood, string> = {
  idle: "",
  thinking: "thinking...",
  asking: "your turn",
  answering: "got it!",
  executing: "working...",
  done: "all done",
  error: "uh oh",
};

export function Mascot({ mood }: { mood: MascotMood }) {
  const face = FACES[mood];
  const label = LABELS[mood];

  return (
    <Box flexDirection="column">
      <Text color="cyan">{"┌───────┐"}</Text>
      <Text>
        <Text color="cyan">{"│ "}</Text>
        <Text color="white">{face.eyes}</Text>
        <Text color="cyan">{" │"}</Text>
      </Text>
      <Text>
        <Text color="cyan">{"│ "}</Text>
        <Text color={face.blush ? "magenta" : "white"}>{face.mouth}</Text>
        <Text color="cyan">{" │"}</Text>
      </Text>
      <Text color="cyan">{"└───────┘"}</Text>
      {label ? (
        <Text color="gray" dimColor>
          {"  "}{label}
        </Text>
      ) : null}
    </Box>
  );
}

/** Map agent status to mascot mood */
export function statusToMood(
  status: string,
  phase?: string
): MascotMood {
  switch (status) {
    case "thinking":
      return "thinking";
    case "asking":
      return "asking";
    case "executing":
      return "executing";
    case "done":
      return "done";
    case "error":
      return "error";
    default:
      return "idle";
  }
}
