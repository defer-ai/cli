import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import { DecisionSummary } from "./DecisionSummary.js";
import type { AgentState } from "../agents/agent.js";

interface Props {
  agents: AgentState[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onClose: () => void;
  rows: number;
}

export function DashboardOverlay({
  agents,
  selectedId,
  onSelect,
  onClose,
  rows,
}: Props) {
  const [tab, setTab] = useState<"decisions" | "agents" | "git">("decisions");

  const current = agents.find((a) => a.id === selectedId) || agents[0];

  useInput((input, key) => {
    if (key.escape || input === "d") {
      onClose();
      return;
    }
    if (key.tab) {
      setTab((prev) =>
        prev === "decisions"
          ? "agents"
          : prev === "agents"
            ? "git"
            : "decisions"
      );
    }
  });

  return (
    <Box flexDirection="column" height={rows} paddingX={1}>
      {/* Tab bar */}
      <Box marginBottom={1}>
        {(["decisions", "agents", "git"] as const).map((t) => (
          <React.Fragment key={t}>
            <Text color={tab === t ? "cyan" : "gray"} bold={tab === t}>
              [{t}]
            </Text>
            <Text> </Text>
          </React.Fragment>
        ))}
        <Box flexGrow={1} />
        <Text color="gray" dimColor>
          tab:switch  esc:close
        </Text>
      </Box>

      {/* Content */}
      <Box flexDirection="column" flexGrow={1}>
        {tab === "decisions" && current && (
          <DecisionSummary decisions={current.decisions} />
        )}

        {tab === "agents" && (
          <Box flexDirection="column" paddingX={1}>
            {agents.map((a) => {
              const answered = a.decisions.filter(
                (d) => d.answer !== null
              ).length;
              return (
                <Box key={a.id} marginBottom={1} flexDirection="column">
                  <Box>
                    <Text
                      color={a.id === selectedId ? "cyan" : "white"}
                      bold={a.id === selectedId}
                    >
                      {a.id}
                    </Text>
                    <Text color="gray"> | </Text>
                    <Text
                      color={
                        a.status === "thinking"
                          ? "cyan"
                          : a.status === "asking"
                            ? "yellow"
                            : a.status === "done"
                              ? "green"
                              : a.status === "error"
                                ? "red"
                                : "gray"
                      }
                    >
                      {a.status}
                    </Text>
                    <Text color="gray">
                      {" "}| {answered}/{a.decisions.length} decisions
                    </Text>
                  </Box>
                  <Box paddingLeft={2}>
                    <Text color="gray" wrap="wrap">
                      {a.task}
                    </Text>
                  </Box>
                </Box>
              );
            })}
          </Box>
        )}

        {tab === "git" && <GitInfo />}
      </Box>
    </Box>
  );
}

function GitInfo() {
  const [info, setInfo] = React.useState<{
    branch: string;
    commits: string[];
    dirty: string[];
  } | null>(null);

  React.useEffect(() => {
    try {
      const { execSync } = require("node:child_process");
      execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });

      const branch = execSync("git branch --show-current", {
        encoding: "utf-8",
      }).trim();

      let commits: string[] = [];
      try {
        commits = execSync("git log --oneline -8", { encoding: "utf-8" })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {}

      let dirty: string[] = [];
      try {
        dirty = execSync("git status --short", { encoding: "utf-8" })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {}

      setInfo({ branch, commits, dirty });
    } catch {
      setInfo(null);
    }
  }, []);

  if (!info) {
    return (
      <Box padding={1}>
        <Text color="gray">Not a git repository.</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={1}>
      <Box>
        <Text color="cyan" bold>
          {info.branch}
        </Text>
      </Box>

      {info.dirty.length > 0 && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="yellow">
            {info.dirty.length} uncommitted
          </Text>
          {info.dirty.slice(0, 5).map((f, i) => (
            <Text key={i} color="gray">
              {"  "}{f}
            </Text>
          ))}
        </Box>
      )}

      {info.commits.length > 0 && (
        <Box flexDirection="column" marginTop={1}>
          <Text color="gray" dimColor>
            Recent commits
          </Text>
          {info.commits.map((c, i) => (
            <Text key={i} color="gray">
              {"  "}{c}
            </Text>
          ))}
        </Box>
      )}
    </Box>
  );
}
