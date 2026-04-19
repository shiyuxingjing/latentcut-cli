package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewProgressCmd returns the `latentcut progress` subcommand.
//
// Unlike `status`, which returns only the list of pending generation tasks,
// `progress` returns a single flat snapshot of the entire pipeline: overall %,
// current phase, step text, episode / shot counts, and a pending-task summary.
// It is the command agents should call when they need to describe progress
// to a human in one line, and it is safe to poll from any LLM framework
// (single fast round-trip, no streaming, no blocking).
func NewProgressCmd() *cobra.Command {
	var watch bool
	var interval int

	cmd := &cobra.Command{
		Use:   "progress <project-uuid>",
		Short: "Show a flat progress snapshot for a project (agent-friendly)",
		Long: `Return a single-shot structured snapshot of project progress,
including overall percent, current phase, phase step, episode/shot counts,
and a pending-task summary aggregated by type.

Prefer this over 'status' when an AI agent or automation needs to report
pipeline progress to a human without parsing the TUI output of 'produce'.`,
		Example: `  latentcut progress project-abc
  latentcut progress project-abc --json
  latentcut progress project-abc --watch
  latentcut progress project-abc --watch --interval 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProgress(args[0], watch, interval)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Keep polling until the project is completed or failed")
	cmd.Flags().IntVar(&interval, "interval", 15, "Polling interval in seconds when --watch is set")

	return cmd
}

func runProgress(projectUUID string, watch bool, intervalSec int) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}
	if intervalSec < 5 {
		intervalSec = 5
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	for {
		snap, err := client.GetProjectProgress(context.Background(), projectUUID)
		if err != nil {
			return fmt.Errorf("get progress: %w", err)
		}

		if jsonOut {
			data, mErr := json.Marshal(snap)
			if mErr != nil {
				return fmt.Errorf("marshal progress: %w", mErr)
			}
			fmt.Println(string(data))
		} else {
			renderProgressHuman(snap)
		}

		if !watch {
			return nil
		}
		// Stop watching when terminal state reached.
		if snap.Status == "completed" || snap.Status == "failed" {
			return nil
		}
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
}

// renderProgressHuman prints a compact one-screen summary of the snapshot
// to stdout. The format is stable and easy to eyeball; the --json flag is
// the right choice for programmatic consumers.
func renderProgressHuman(s *latentcut.ProjectProgress) {
	fmt.Fprintf(os.Stdout, "Project: %s  (%s)\n", s.ProjectUUID, s.Title)
	fmt.Fprintf(os.Stdout, "Status:  %s  overall=%.0f%%\n", s.Status, s.OverallProgress)
	fmt.Fprintf(os.Stdout, "Phase:   %s", s.CurrentPhase)
	if s.PhaseProgress != nil {
		fmt.Fprintf(os.Stdout, "  (%.0f%%)", *s.PhaseProgress)
	}
	if s.PhaseStep != "" {
		fmt.Fprintf(os.Stdout, "  %s", s.PhaseStep)
	}
	fmt.Fprintln(os.Stdout)

	episodesLine := fmt.Sprintf("Episodes: %d total", s.Episodes.Total)
	if s.Episodes.Parsed != nil {
		episodesLine += fmt.Sprintf(", %d parsed", *s.Episodes.Parsed)
	}
	fmt.Fprintln(os.Stdout, episodesLine)
	fmt.Fprintf(os.Stdout, "Shots:    %d total\n", s.Shots.Total)

	if s.PendingTasks.Total > 0 {
		fmt.Fprintf(os.Stdout, "\nPending tasks: %d\n", s.PendingTasks.Total)
		keys := make([]string, 0, len(s.PendingTasks.ByType))
		for k := range s.PendingTasks.ByType {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stdout, "  %-22s %d\n", k, s.PendingTasks.ByType[k])
		}
	} else {
		fmt.Fprintln(os.Stdout, "\nPending tasks: none")
	}

	if s.ShotParse != nil && s.ShotParse.ErrorMessage != "" {
		fmt.Fprintf(os.Stdout, "\nShot-parse error: %s\n", strings.TrimSpace(s.ShotParse.ErrorMessage))
		if s.ShotParse.RetryEligible != nil && *s.ShotParse.RetryEligible {
			fmt.Fprintln(os.Stdout, "  (retry eligible — use: latentcut generate ... --wait)")
		}
	}
}
