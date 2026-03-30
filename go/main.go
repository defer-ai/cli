package main

import "github.com/defer-ai/cli/cmd"

// version is set at build time: go build -ldflags "-X main.version=0.1.0"
var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}
