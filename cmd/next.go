package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewNextCmd returns the next subcommand.
func NewNextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next <project-uuid>",
		Short: "Show recommended next steps for a project",
		Long:  "Calls assistant-summary to display project status and recommended next actions.",
		Example: `  latentcut next project-xxx
  latentcut next project-xxx --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNext(args[0])
		},
	}
}

func runNext(projectUUID string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	summary, err := client.GetAssistantSummary(context.Background(), projectUUID)
	if err != nil {
		return fmt.Errorf("get assistant summary: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(summary)
		fmt.Println(string(data))
		return nil
	}

	// Pretty print
	fmt.Fprintf(os.Stdout, "Project: %s (%s)\n\n", summary.Title, summary.Status)

	fmt.Fprintf(os.Stdout, "Structure: %d episode(s) · %d shots · %d characters · %d locations\n",
		summary.Structure.Episodes, summary.Structure.Shots,
		summary.Structure.Characters, summary.Structure.Locations)

	fmt.Fprintf(os.Stdout, "Assets:    %d/%d character images · %d/%d voices · %d/%d location images\n",
		summary.AssetsReady.CharacterImages, summary.Structure.Characters,
		summary.AssetsReady.CharacterVoices, summary.Structure.Characters,
		summary.AssetsReady.LocationImages, summary.Structure.Locations)

	fmt.Fprintf(os.Stdout, "Progress:  %d%% · Phase: %s · %d pending tasks\n\n",
		summary.Progress.Overall, summary.Progress.Phase, summary.Progress.PendingTasks)

	if len(summary.ModeRecommendations) > 0 {
		fmt.Fprintln(os.Stdout, "Recommended next steps:")
		for i, rec := range summary.ModeRecommendations {
			fmt.Fprintf(os.Stdout, "  %d. [%s] %s — %s\n", i+1, rec.Mode, rec.Label, rec.Reason)
		}
	}

	if summary.NextSuggestedTarget != nil {
		fmt.Fprintf(os.Stdout, "\nNext target: Episode %d, Shot %d (%s)\n",
			summary.NextSuggestedTarget.EpisodeNumber,
			summary.NextSuggestedTarget.ShotNumber,
			summary.NextSuggestedTarget.ShotUUID)
	}

	return nil
}
