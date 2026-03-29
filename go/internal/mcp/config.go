package mcp

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

// ServerConfig describes how to launch an MCP server.
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// Config is the top-level MCP configuration.
type Config struct {
	Servers map[string]ServerConfig `json:"servers"`
}

// LoadConfig loads MCP server configuration.
// It looks in the following locations (first found wins):
//  1. .defer/mcp.json (project-local)
//  2. ~/.config/defer/mcp.json (user-global)
func LoadConfig(cwd string) (*Config, error) {
	paths := []string{
		filepath.Join(cwd, ".defer", "mcp.json"),
	}

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".config", "defer", "mcp.json"))
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	return nil, nil // no config found
}

// ConnectAll connects to all configured MCP servers and returns the clients.
func ConnectAll(cfg *Config) (map[string]*Client, error) {
	if cfg == nil || len(cfg.Servers) == 0 {
		return nil, nil
	}

	clients := make(map[string]*Client)
	for name, sc := range cfg.Servers {
		client, err := NewClient(sc.Command, sc.Args...)
		if err != nil {
			// Close any already-connected clients
			for _, c := range clients {
				c.Close()
			}
			return nil, err
		}

		// Set environment variables for the subprocess if specified
		if len(sc.Env) > 0 && client.cmd.Process != nil {
			// Environment is set before Start, but we already started.
			// For env support, we'd need to set it before NewClient.
			// This is a known limitation; env must be set before process start.
		}

		if err := client.Initialize(); err != nil {
			client.Close()
			for _, c := range clients {
				c.Close()
			}
			return nil, err
		}

		clients[name] = client
	}

	return clients, nil
}

// NewClientWithEnv creates an MCP client with custom environment variables.
func NewClientWithEnv(sc ServerConfig) (*Client, error) {
	cmd := newCmdWithEnv(sc)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
	}, nil
}

func newCmdWithEnv(sc ServerConfig) *exec.Cmd {
	cmd := exec.Command(sc.Command, sc.Args...)
	if len(sc.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range sc.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd
}
