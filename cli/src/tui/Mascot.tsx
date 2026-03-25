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

// Pixel-art faces using block characters
// Each face is 16 chars wide, 7 lines tall
const FACES: Record<MascotMood, string[]> = {
  idle: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ▄▄▄  ▄▄▄   █ ",
    " █              █ ",
    " █    ══════    █ ",
    " █              █ ",
    "  ▀████████████▀  ",
  ],
  thinking: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ████  ████  █ ",
    " █              █ ",
    " █      ██      █ ",
    " █              █ ",
    "  ▀████████████▀  ",
  ],
  asking: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ████  ████  █ ",
    " █  ████  ████  █ ",
    " █     ▄▄▄▄    █ ",
    " █     ▀▀      █ ",
    "  ▀████████████▀  ",
  ],
  answering: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ▀▀▀  ▀▀▀   █ ",
    " █              █ ",
    " █   ▄██████▄   █ ",
    " █              █ ",
    "  ▀████████████▀  ",
  ],
  executing: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ▀██▀ ▀██▀  █ ",
    " █              █ ",
    " █   ████████   █ ",
    " █              █ ",
    "  ▀████████████▀  ",
  ],
  done: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ▀▀▀  ▀▀▀   █ ",
    " █              █ ",
    " █  ▄████████▄  █ ",
    " █  ▀▀▀▀▀▀▀▀▀▀  █ ",
    "  ▀████████████▀  ",
  ],
  error: [
    "  ▄████████████▄  ",
    " █              █ ",
    " █  ▀██▄ ▄██▀  █ ",
    " █   ▄██▄██▄   █ ",
    " █              █ ",
    " █   ▄▀▀▀▀▄    █ ",
    "  ▀████████████▀  ",
  ],
};

const LABELS: Record<MascotMood, string> = {
  idle: "",
  thinking: "thinking...",
  asking: "your turn",
  answering: "got it",
  executing: "working...",
  done: "all done",
  error: "uh oh",
};

export function Mascot({ mood }: { mood: MascotMood }) {
  const face = FACES[mood];
  const label = LABELS[mood];

  return (
    <Box flexDirection="column">
      {face.map((line, i) => (
        <Text key={i} color="cyan">
          {line}
        </Text>
      ))}
      {label ? (
        <Box justifyContent="center">
          <Text color="gray" dimColor>
            {label}
          </Text>
        </Box>
      ) : null}
    </Box>
  );
}

// Inline mini face for the header (single line)
export function MiniMascot({ mood }: { mood: MascotMood }) {
  const mini: Record<MascotMood, string> = {
    idle: "[· ·]",
    thinking: "[◠ ◠]",
    asking: "[◉ ◉]",
    answering: "[◠‿◠]",
    executing: "[▪ ▪]",
    done: "[^ ^]",
    error: "[x x]",
  };

  return <Text color="cyan">{mini[mood]}</Text>;
}

export function statusToMood(
  status: string,
  _phase?: string
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
