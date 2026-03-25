import React, { useState, useEffect } from "react";
import { Box, Text } from "ink";
import { execSync } from "node:child_process";

export function GitTab() {
  const [gitInfo, setGitInfo] = useState<{
    isRepo: boolean;
    branch: string;
    recentCommits: string[];
    uncommittedFiles: string[];
  }>({
    isRepo: false,
    branch: "",
    recentCommits: [],
    uncommittedFiles: [],
  });

  useEffect(() => {
    try {
      execSync("git rev-parse --is-inside-work-tree", { stdio: "pipe" });

      const branch = execSync("git branch --show-current", {
        encoding: "utf-8",
      }).trim();

      let recentCommits: string[] = [];
      try {
        recentCommits = execSync("git log --oneline -10", {
          encoding: "utf-8",
        })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {
        // no commits yet
      }

      let uncommittedFiles: string[] = [];
      try {
        uncommittedFiles = execSync("git status --short", {
          encoding: "utf-8",
        })
          .trim()
          .split("\n")
          .filter(Boolean);
      } catch {
        // ignore
      }

      setGitInfo({ isRepo: true, branch, recentCommits, uncommittedFiles });
    } catch {
      setGitInfo({
        isRepo: false,
        branch: "",
        recentCommits: [],
        uncommittedFiles: [],
      });
    }
  }, []);

  if (!gitInfo.isRepo) {
    return (
      <Box padding={1}>
        <Text color="gray">Not a git repository.</Text>
      </Box>
    );
  }

  return (
    <Box padding={1} flexDirection="column">
      <Box marginBottom={1}>
        <Text color="cyan" bold>
          Branch:{" "}
        </Text>
        <Text>{gitInfo.branch}</Text>
      </Box>

      {gitInfo.uncommittedFiles.length > 0 && (
        <Box flexDirection="column" marginBottom={1}>
          <Text color="yellow" bold>
            Uncommitted ({gitInfo.uncommittedFiles.length})
          </Text>
          {gitInfo.uncommittedFiles.slice(0, 10).map((f, i) => (
            <Box key={i} paddingLeft={1}>
              <Text color="gray">{f}</Text>
            </Box>
          ))}
          {gitInfo.uncommittedFiles.length > 10 && (
            <Box paddingLeft={1}>
              <Text color="gray" dimColor>
                ...and {gitInfo.uncommittedFiles.length - 10} more
              </Text>
            </Box>
          )}
        </Box>
      )}

      <Box flexDirection="column">
        <Text color="cyan" bold>
          Recent commits
        </Text>
        {gitInfo.recentCommits.length === 0 ? (
          <Box paddingLeft={1}>
            <Text color="gray">No commits yet.</Text>
          </Box>
        ) : (
          gitInfo.recentCommits.map((c, i) => (
            <Box key={i} paddingLeft={1}>
              <Text color="gray">{c}</Text>
            </Box>
          ))
        )}
      </Box>
    </Box>
  );
}
