import React, { useState, useMemo } from "react";
import { Box, Text, useInput } from "ink";
import { MiniMascot } from "./Mascot.js";
import type { Decision } from "../decisions.js";

export type CareLevel = "skip" | "low" | "medium" | "high" | "paranoid";

const LEVELS: { key: CareLevel; label: string; color: string; description: string }[] = [
  { key: "skip", label: "skip", color: "gray", description: "delegate everything" },
  { key: "low", label: "low", color: "blue", description: "only the key question" },
  { key: "medium", label: "medium", color: "yellow", description: "important decisions" },
  { key: "high", label: "high", color: "green", description: "ask me everything" },
  { key: "paranoid", label: "paranoid", color: "red", description: "deep dive, sub-questions" },
];

const LEVEL_BAR: Record<CareLevel, string> = {
  skip: "░░░░░",
  low: "█░░░░",
  medium: "██░░░",
  high: "████░",
  paranoid: "█████",
};

interface Props {
  decisions: Decision[];
  onComplete: (priorities: Record<string, CareLevel>) => void;
  rows: number;
}

export function DomainPriority({ decisions, onComplete, rows }: Props) {
  // Extract unique categories
  const categories = useMemo(() => {
    const cats: string[] = [];
    for (const d of decisions) {
      if (!cats.includes(d.category)) cats.push(d.category);
    }
    return cats;
  }, [decisions]);

  const [cursorIdx, setCursorIdx] = useState(0);
  const [priorities, setPriorities] = useState<Record<string, CareLevel>>(
    () => {
      const initial: Record<string, CareLevel> = {};
      for (const cat of categories) {
        initial[cat] = "medium";
      }
      return initial;
    }
  );

  const currentCat = categories[cursorIdx];
  const currentLevel = priorities[currentCat];
  const currentLevelIdx = LEVELS.findIndex((l) => l.key === currentLevel);
  const decisionsInCat = decisions.filter((d) => d.category === currentCat);

  const [addingCategory, setAddingCategory] = useState(false);
  const [newCatValue, setNewCatValue] = useState("");

  useInput((input, key) => {
    // Adding a new category
    if (addingCategory) {
      if (key.escape) {
        setAddingCategory(false);
        setNewCatValue("");
        return;
      }
      if (key.return && newCatValue.trim()) {
        const name = newCatValue.trim();
        if (!categories.includes(name)) {
          categories.push(name);
          setPriorities((prev) => ({ ...prev, [name]: "high" }));
          setCursorIdx(categories.length - 1);
        }
        setAddingCategory(false);
        setNewCatValue("");
        return;
      }
      if (key.backspace || key.delete) {
        setNewCatValue((v) => v.slice(0, -1));
        return;
      }
      if (input && !key.ctrl && !key.meta) {
        setNewCatValue((v) => v + input);
      }
      return;
    }

    // Navigate categories
    if (input === "j" || key.downArrow) {
      setCursorIdx((i) => Math.min(i + 1, categories.length - 1));
    }
    if (input === "k" || key.upArrow) {
      setCursorIdx((i) => Math.max(i - 1, 0));
    }

    // Adjust care level with left/right
    if ((input === "h" || key.leftArrow) && currentLevelIdx > 0) {
      setPriorities((prev) => ({
        ...prev,
        [currentCat]: LEVELS[currentLevelIdx - 1].key,
      }));
    }
    if (
      (input === "l" || key.rightArrow) &&
      currentLevelIdx < LEVELS.length - 1
    ) {
      setPriorities((prev) => ({
        ...prev,
        [currentCat]: LEVELS[currentLevelIdx + 1].key,
      }));
    }

    // Add new category
    if (input === "n") {
      setAddingCategory(true);
      setNewCatValue("");
    }

    // Confirm
    if (key.return) {
      onComplete(priorities);
    }

    // Escape: confirm with current settings
    if (key.escape) {
      onComplete(priorities);
    }
  });

  return (
    <Box flexDirection="column" height={rows} paddingX={3} paddingY={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <MiniMascot mood="asking" />
        <Text color="cyan" bold>
          {"  "}How much do you care about each area?
        </Text>
      </Box>
      <Box marginBottom={1}>
        <Text color="gray" dimColor>
          This controls how many questions you get per domain.
          Use ←→ to adjust, ↑↓ to navigate, enter to confirm.
        </Text>
      </Box>

      {/* Category list */}
      <Box flexDirection="column" marginBottom={1}>
        {categories.map((cat, i) => {
          const isCursor = i === cursorIdx;
          const level = priorities[cat];
          const levelInfo = LEVELS.find((l) => l.key === level)!;
          const count = decisions.filter((d) => d.category === cat).length;

          return (
            <Box key={cat} paddingLeft={1} marginBottom={0}>
              <Text color={isCursor ? "cyan" : "gray"}>
                {isCursor ? ">" : " "}{" "}
              </Text>
              <Text
                color={isCursor ? "white" : "gray"}
                bold={isCursor}
              >
                {cat.padEnd(18)}
              </Text>
              <Text color={levelInfo.color as any}>
                {LEVEL_BAR[level]}
              </Text>
              <Text color={levelInfo.color as any}>
                {" "}
                {levelInfo.label.padEnd(10)}
              </Text>
              <Text color="gray" dimColor>
                {count} decision{count !== 1 ? "s" : ""}
              </Text>
            </Box>
          );
        })}
      </Box>

      {/* Detail for selected category */}
      <Box
        flexDirection="column"
        marginTop={1}
        paddingX={1}
        borderStyle="single"
        borderColor="gray"
        borderTop
        borderBottom={false}
        borderLeft={false}
        borderRight={false}
      >
        <Box marginTop={1}>
          <Text color="cyan" bold>
            {currentCat}
          </Text>
          <Text color="gray">
            {"  "}
            {LEVELS.find((l) => l.key === currentLevel)?.description}
          </Text>
        </Box>
        <Box marginTop={1} flexDirection="column">
          <Text color="gray" dimColor>
            Questions in this domain:
          </Text>
          {decisionsInCat.slice(0, 5).map((d) => (
            <Box key={d.id} paddingLeft={1}>
              <Text color="gray" dimColor>
                {currentLevel === "skip" ? "◆" : "○"} {d.question}
              </Text>
            </Box>
          ))}
          {decisionsInCat.length > 5 ? (
            <Box paddingLeft={1}>
              <Text color="gray" dimColor>
                ...and {decisionsInCat.length - 5} more
              </Text>
            </Box>
          ) : null}
        </Box>
      </Box>

      {/* Add category input */}
      {addingCategory ? (
        <Box marginTop={1} paddingLeft={2}>
          <Text color="yellow">new domain: </Text>
          <Text>{newCatValue}</Text>
          <Text color="gray">|</Text>
          <Text color="gray" dimColor>
            {"  "}enter:add  esc:cancel
          </Text>
        </Box>
      ) : null}

      {/* Scale legend */}
      <Box flexGrow={1} />
      <Box>
        {LEVELS.map((l, i) => (
          <React.Fragment key={l.key}>
            <Text color={l.color as any} dimColor={l.key !== currentLevel}>
              {l.label}
            </Text>
            {i < LEVELS.length - 1 ? (
              <Text color="gray" dimColor>
                {"  "}
              </Text>
            ) : null}
          </React.Fragment>
        ))}
      </Box>
      <Box>
        <Text color="gray" dimColor>
          ←→:adjust  ↑↓:navigate  n:add domain  enter:confirm
        </Text>
      </Box>
    </Box>
  );
}
