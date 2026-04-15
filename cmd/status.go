package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewStatusCmd returns the status subcommand.
func NewStatusCmd() *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "status <project-uuid>",
		Short: "Show pending generation tasks for a project",
		Long:  "Display all pending/processing tasks (character images, voices, keyframes, videos, etc.) for a project. Use --watch to auto-refresh.",
		Example: `  latentcut status proj-xxx
  latentcut status proj-xxx --watch
  latentcut status proj-xxx --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0], watch)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Auto-refresh every 5 seconds until all tasks complete")

	return cmd
}

func runStatus(projectUUID string, watch bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	for {
		tasks, err := client.GetPendingTasks(context.Background(), projectUUID)
		if err != nil {
			return fmt.Errorf("get pending tasks: %w", err)
		}

		if jsonOut {
			data, _ := json.Marshal(tasks)
			fmt.Println(string(data))
			if !watch {
				return nil
			}
			if len(tasks) == 0 {
				return nil
			}
			time.Sleep(5 * time.Second)
			continue
		}

		if watch {
			// Clear screen for watch mode
			fmt.Print("\033[H\033[2J")
			fmt.Fprintf(os.Stdout, "Project: %s  (refreshing every 5s, Ctrl+C to stop)\n\n", projectUUID)
		}

		if len(tasks) == 0 {
			fmt.Fprintln(os.Stdout, "No pending tasks. All generation complete! ✅")
			return nil
		}

		// Count by type
		typeCounts := make(map[string]int)
		statusCounts := make(map[string]int)
		for _, t := range tasks {
			typeCounts[t.TaskType]++
			statusCounts[t.TaskStatus]++
		}

		fmt.Fprintf(os.Stdout, "Pending tasks: %d\n\n", len(tasks))

		// Summary by type
		fmt.Fprintln(os.Stdout, "By type:")
		for typ, count := range typeCounts {
			fmt.Fprintf(os.Stdout, "  %-20s %d\n", typ, count)
		}

		// Summary by status
		fmt.Fprintln(os.Stdout, "\nBy status:")
		for status, count := range statusCounts {
			fmt.Fprintf(os.Stdout, "  %-20s %d\n", status, count)
		}

		// Detail table
		fmt.Fprintln(os.Stdout, "\nDetails:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  TASK_UUID\tTYPE\tSTATUS\tERROR")
		for _, t := range tasks {
			errMsg := t.Error
			if len(errMsg) > 50 {
				errMsg = errMsg[:50] + "..."
			}
			uuid := t.TaskUUID
			if len(uuid) > 20 {
				uuid = uuid[:20] + "..."
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", uuid, t.TaskType, t.TaskStatus, errMsg)
		}
		w.Flush()

		if !watch {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}
