package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version variables set by goreleaser ldflags (-X main.version=...).
// Populated via SetVersionInfo() called from main.go.
var (
	cliVersion = "dev"
	cliCommit  = "none"
	cliDate    = "unknown"
)

// SetVersionInfo is called from main.go to inject goreleaser ldflags values.
func SetVersionInfo(v, c, d string) {
	cliVersion = v
	cliCommit = c
	cliDate = d
}

// NewVersionCmd returns the version subcommand.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("novelo-cli %s (commit: %s, built: %s)\n", cliVersion, cliCommit, cliDate)
		},
	}
}
