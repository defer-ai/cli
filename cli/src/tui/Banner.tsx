import React from "react";
import { Box, Text } from "ink";

// Unique mascot: a small diamond-eyed listener with a thought bubble
// Represents "I'm here to listen and ask, not decide"
const MASCOT = [
  `        ○`,
  `       ╱│╲`,
  `      ╱ │ ╲`,
  `     ◇  │  ◇`,
  `      ╲ │ ╱`,
  `       ╲│╱`,
  `        │`,
];

const VERSION = "0.1.0";

export function Banner({ model, cwd }: { model: string; cwd: string }) {
  const dir = cwd.replace(process.env.HOME || "", "~");

  return (
    <Box flexDirection="column" paddingX={2} paddingTop={1}>
      <Box flexDirection="row">
        <Box flexDirection="column" marginRight={3}>
          {MASCOT.map((line, i) => (
            <Text key={i} color="cyan">
              {line}
            </Text>
          ))}
        </Box>
        <Box flexDirection="column" paddingTop={1}>
          <Text bold>
            <Text color="cyan">defer</Text>
            <Text color="gray" dimColor>
              {" "}v{VERSION}
            </Text>
          </Text>
          <Text color="gray" dimColor>
            Zero-autonomy AI
          </Text>
          <Box marginTop={1}>
            <Text color="gray" dimColor>
              model{" "}
            </Text>
            <Text color="white">{model}</Text>
          </Box>
          <Box>
            <Text color="gray" dimColor>
              cwd   {dir}
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text color="gray" dimColor>
              /help for commands, tab to switch views
            </Text>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}

/** Compact header shown at the top of every view */
export function Header({ model }: { model: string }) {
  return (
    <Box paddingX={1}>
      <Text color="cyan" bold>
        defer
      </Text>
      <Text color="gray" dimColor>
        {" "}v{VERSION} | {model}
      </Text>
    </Box>
  );
}
