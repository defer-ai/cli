package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/defer-ai/cli/internal/mcp"
	"github.com/spf13/cobra"
)

var serveMCP bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run defer as a server",
	Long: `Run defer as a headless server. Currently supports MCP (Model Context Protocol)
over stdio for integration with AI coding tools.

Example MCP configuration for Claude Code (~/.claude/mcp.json):
  {
    "servers": {
      "defer": {
        "command": "defer",
        "args": ["serve", "--mcp"]
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !serveMCP {
			return cmd.Help()
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		server := mcp.NewServer(cwd, Version)
		return server.Run(ctx)
	},
}

func init() {
	serveCmd.Flags().BoolVar(&serveMCP, "mcp", false, "Run as MCP server (stdio transport)")
	rootCmd.AddCommand(serveCmd)
}
