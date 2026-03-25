import React, { useState } from "react";
import { Box, Text, useInput } from "ink";

export interface ParsedOption {
  label: string;
  value: string;
}

interface Props {
  options?: ParsedOption[];
  onSubmit: (value: string) => void;
  onCancel: () => void;
}

export function InputBar({ options, onSubmit, onCancel }: Props) {
  const [mode, setMode] = useState<"select" | "text">(
    options && options.length > 0 ? "select" : "text"
  );
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [textValue, setTextValue] = useState("");

  useInput((input, key) => {
    if (key.escape) {
      if (mode === "text" && options && options.length > 0) {
        setMode("select");
        setTextValue("");
        return;
      }
      onCancel();
      return;
    }

    if (mode === "select") {
      if (input === "j" || key.downArrow) {
        setSelectedIdx((i) =>
          Math.min(i + 1, (options?.length || 1) - 1)
        );
        return;
      }
      if (input === "k" || key.upArrow) {
        setSelectedIdx((i) => Math.max(i - 1, 0));
        return;
      }
      if (key.return) {
        const opt = options?.[selectedIdx];
        if (opt) {
          onSubmit(opt.value);
        }
        return;
      }
      // Any other key switches to text mode
      if (input === "t" || input === "/") {
        setMode("text");
        return;
      }
    }

    if (mode === "text") {
      if (key.return) {
        onSubmit(textValue);
        return;
      }
      if (key.backspace || key.delete) {
        setTextValue((v) => v.slice(0, -1));
        return;
      }
      if (input && !key.ctrl && !key.meta) {
        setTextValue((v) => v + input);
      }
    }
  });

  if (mode === "select" && options && options.length > 0) {
    return (
      <Box
        borderStyle="single"
        borderColor="yellow"
        flexDirection="column"
        paddingX={1}
      >
        {options.map((opt, i) => (
          <Box key={i}>
            <Text color={i === selectedIdx ? "cyan" : "white"}>
              {i === selectedIdx ? "> " : "  "}
            </Text>
            <Text color={i === selectedIdx ? "cyan" : "white"}>
              {opt.label}
            </Text>
          </Box>
        ))}
        <Box marginTop={1}>
          <Text color="gray" dimColor>
            j/k:move enter:select t:type custom esc:cancel
          </Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box borderStyle="single" borderColor="yellow" paddingX={1}>
      <Text color="yellow">{"> "}</Text>
      <Text>{textValue}</Text>
      <Text color="gray">|</Text>
      <Box flexGrow={1} />
      <Text color="gray" dimColor>
        enter:send esc:{options ? "back" : "cancel"}
      </Text>
    </Box>
  );
}
