import React from "react";
import { Box, Text } from "ink";
import { Mascot, MiniMascot, type MascotMood } from "./Mascot.js";

const VERSION = "0.1.0";

export function Banner({
  model,
  cwd,
  mood,
}: {
  model: string;
  cwd: string;
  mood: MascotMood;
}) {
  const dir = cwd.replace(process.env.HOME || "", "~");

  return (
    <Box flexDirection="column" paddingX={1} paddingTop={1}>
      <Box flexDirection="row">
        <Box marginRight={3}>
          <Mascot mood={mood} />
        </Box>
        <Box flexDirection="column" paddingTop={2}>
          <Text bold>
            <Text color="cyan">defer</Text>
            <Text color="gray" dimColor>
              {" "}
              v{VERSION}
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

export function Header({
  model,
  mood,
}: {
  model: string;
  mood: MascotMood;
}) {
  return (
    <Box paddingX={1}>
      <MiniMascot mood={mood} />
      <Text color="cyan" bold>
        {" "}
        defer
      </Text>
      <Text color="gray" dimColor>
        {" "}
        v{VERSION} | {model}
      </Text>
    </Box>
  );
}
