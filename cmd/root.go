package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	jsonOut bool
)

// rootCmd is the base command for latentcut.
var rootCmd = &cobra.Command{
	Use:   "latentcut",
	Short: "Novelo AI pipeline CLI",
	Long:  "latentcut triggers the Novelo AI novel-to-drama pipeline and streams real-time progress.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output JSONL to stdout instead of progress display")

	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewLoginCmd())
	rootCmd.AddCommand(NewProduceCmd())
	rootCmd.AddCommand(NewChatCmd())
	rootCmd.AddCommand(NewCreditsCmd())
	rootCmd.AddCommand(NewRechargeCmd())
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewProjectCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(NewProgressCmd())
	rootCmd.AddCommand(NewGenerateCmd())
	rootCmd.AddCommand(NewInstallCmd())
	rootCmd.AddCommand(NewOpenCmd())
	rootCmd.AddCommand(NewNextCmd())
	rootCmd.AddCommand(NewGalleryCmd())
	rootCmd.AddCommand(NewPlanCmd())
	rootCmd.AddCommand(NewShotVideoCmd())
}
