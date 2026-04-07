package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	jsonOut bool
)

// rootCmd is the base command for novelo-cli.
var rootCmd = &cobra.Command{
	Use:   "novelo-cli",
	Short: "Novelo AI pipeline CLI",
	Long:  "novelo-cli triggers the Novelo AI novel-to-drama pipeline and streams real-time progress.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output JSONL to stdout instead of progress display")

	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewVersionCmd())
}
