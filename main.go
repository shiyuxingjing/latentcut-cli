package main

import (
	"fmt"
	"os"

	"github.com/novelo-ai/novelo-cli/cmd"
)

// These variables are populated by goreleaser ldflags at build time:
//   -X main.version={{.Version}}
//   -X main.commit={{.Commit}}
//   -X main.date={{.Date}}
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
