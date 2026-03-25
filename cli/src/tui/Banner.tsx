import React from "react";
import { Box, Text } from "ink";

const BANNER = `
     ██████╗ ███████╗███████╗███████╗██████╗
     ██╔══██╗██╔════╝██╔════╝██╔════╝██╔══██╗
     ██║  ██║█████╗  █████╗  █████╗  ██████╔╝
     ██║  ██║██╔══╝  ██╔══╝  ██╔══╝  ██╔══██╗
     ██████╔╝███████╗██║     ███████╗██║  ██║
     ╚═════╝ ╚══════╝╚═╝     ╚══════╝╚═╝  ╚═╝
`;

export function Banner({ model, cwd }: { model: string; cwd: string }) {
  const dir = cwd.replace(process.env.HOME || "", "~");

  return (
    <Box flexDirection="column" paddingX={2}>
      <Text color="cyan">{BANNER}</Text>
      <Box>
        <Text color="gray">     Zero-autonomy AI. Every decision is yours.</Text>
      </Box>
      <Box marginTop={1}>
        <Text color="gray">     model: </Text>
        <Text color="white">{model}</Text>
        <Text color="gray">  cwd: </Text>
        <Text color="white">{dir}</Text>
      </Box>
      <Box marginTop={1}>
        <Text color="gray">     Type a task to start. /help for commands.</Text>
      </Box>
    </Box>
  );
}
