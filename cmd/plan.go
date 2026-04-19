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

// NewPlanCmd returns the plan subcommand with mode-based subcommands + execute.
func NewPlanCmd() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview and execute workflow plans",
		Long:  "Preview what will happen for a given mode, then execute the plan.",
	}

	planCmd.AddCommand(newPlanAssetsCmd())
	planCmd.AddCommand(newPlanShotVideoCmd())
	planCmd.AddCommand(newPlanEpisodeVideoCmd())
	planCmd.AddCommand(newPlanFullDramaCmd())
	planCmd.AddCommand(newPlanExecuteCmd())

	return planCmd
}

func newPlanAssetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "assets <project-uuid>",
		Short:   "Preview assets_only plan",
		Example: "  latentcut plan assets project-xxx --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanPreview(args[0], "assets_only", nil)
		},
	}
}

func newPlanShotVideoCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "shot-video <project-uuid> <shot-uuid>",
		Short:   "Preview shot_video plan for a specific shot",
		Example: "  latentcut plan shot-video project-xxx shot-xxx --json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanPreview(args[0], "shot_video", &latentcut.WorkflowPreviewTarget{
				ShotUUID: args[1],
			})
		},
	}
}

func newPlanEpisodeVideoCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "episode-video <project-uuid> <episode-uuid>",
		Short:   "Preview episode_video plan for a specific episode",
		Example: "  latentcut plan episode-video project-xxx episode-xxx --json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanPreview(args[0], "episode_video", &latentcut.WorkflowPreviewTarget{
				EpisodeUUID: args[1],
			})
		},
	}
}

func newPlanFullDramaCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "full-drama <project-uuid>",
		Short:   "Preview full_drama plan for the entire project",
		Example: "  latentcut plan full-drama project-xxx --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanPreview(args[0], "full_drama", nil)
		},
	}
}

func newPlanExecuteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "execute <project-uuid> <preview-id>",
		Short:   "Execute a previously previewed plan",
		Example: "  latentcut plan execute project-xxx preview-xxx --json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanExecute(args[0], args[1])
		},
	}
}

func runPlanPreview(projectUUID, mode string, target *latentcut.WorkflowPreviewTarget) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	preview, err := client.PreviewWorkflow(context.Background(), projectUUID, latentcut.WorkflowPreviewRequest{
		Mode:   mode,
		Target: target,
	})
	if err != nil {
		return fmt.Errorf("preview workflow: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(preview)
		fmt.Println(string(data))
		return nil
	}

	// Pretty print
	fmt.Fprintf(os.Stdout, "Plan: %s (%s)\n", preview.Mode, preview.PreviewID)

	if len(preview.Scope.ShotUUIDs) > 0 {
		fmt.Fprintf(os.Stdout, "Scope: %d shot(s), %d episode(s)\n",
			len(preview.Scope.ShotUUIDs), len(preview.Scope.EpisodeUUIDs))
	}

	fmt.Fprintf(os.Stdout, "Estimated: %d credits · ~%ds\n\n",
		preview.Cost.EstimatedCredits, preview.Timing.EstimatedSeconds)

	if len(preview.Outputs) > 0 {
		fmt.Fprintln(os.Stdout, "Will generate:")
		for _, o := range preview.Outputs {
			fmt.Fprintf(os.Stdout, "  • %d × %s\n", o.Count, o.Type)
		}
		fmt.Fprintln(os.Stdout)
	}

	if len(preview.MissingDependencies) > 0 {
		fmt.Fprintln(os.Stdout, "Missing dependencies:")
		for _, d := range preview.MissingDependencies {
			name := d.Name
			if name == "" {
				name = d.UUID
			}
			fmt.Fprintf(os.Stdout, "  • %s: %s\n", d.Type, name)
		}
		fmt.Fprintln(os.Stdout)
	}

	if preview.Explain.Summary != "" {
		fmt.Fprintf(os.Stdout, "Summary: %s\n", preview.Explain.Summary)
	}
	if preview.Explain.UserSafeSummary != "" {
		fmt.Fprintf(os.Stdout, "Note: %s\n", preview.Explain.UserSafeSummary)
	}

	if preview.CanExecute {
		fmt.Fprintf(os.Stdout, "\n✅ Ready to execute.\n")
		fmt.Fprintf(os.Stdout, "Run with: latentcut plan execute %s %s\n", projectUUID, preview.PreviewID)
	} else {
		reason := "unknown"
		if preview.BlockReason != nil {
			reason = *preview.BlockReason
		}
		fmt.Fprintf(os.Stdout, "\n❌ Cannot execute: %s\n", reason)
	}

	return nil
}

func runPlanExecute(projectUUID, previewID string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	result, err := client.ExecuteWorkflow(context.Background(), projectUUID, previewID)
	if err != nil {
		return fmt.Errorf("execute workflow: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}

	if result.Accepted {
		fmt.Fprintf(os.Stdout, "✅ Workflow accepted!\n")
		fmt.Fprintf(os.Stdout, "  Run ID:   %s\n", result.WorkflowRunID)
		fmt.Fprintf(os.Stdout, "  Mode:     %s\n", result.Mode)
		fmt.Fprintf(os.Stdout, "  Tasks:    %d created\n", result.CreatedTasks)
		fmt.Fprintf(os.Stdout, "  Credits:  %d estimated\n", result.EstimatedCredits)
		fmt.Fprintf(os.Stdout, "\nTrack progress with: latentcut progress %s\n", projectUUID)
	} else {
		fmt.Fprintln(os.Stderr, "❌ Workflow rejected.")
	}

	return nil
}
