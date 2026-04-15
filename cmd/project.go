package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewProjectCmd returns the project subcommand with list/show/delete/create/assets subcommands.
func NewProjectCmd() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	projectCmd.AddCommand(newProjectListCmd())
	projectCmd.AddCommand(newProjectShowCmd())
	projectCmd.AddCommand(newProjectDeleteCmd())
	projectCmd.AddCommand(newProjectCreateCmd())
	projectCmd.AddCommand(newProjectAssetsCmd())

	return projectCmd
}

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Example: `  latentcut project list
  latentcut project list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectList()
		},
	}
}

func runProjectList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(projects)
		fmt.Println(string(data))
		return nil
	}

	if len(projects) == 0 {
		fmt.Fprintln(os.Stderr, "No projects found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UUID\tTITLE\tSTATUS\tCREATED")
	for _, p := range projects {
		uuid := getString(p, "project_uuid")
		title := getString(p, "title")
		status := getString(p, "status")
		created := getString(p, "created_at")
		if len(created) > 10 {
			created = created[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", uuid, truncate(title, 30), status, created)
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "\nTotal: %d projects\n", len(projects))
	return nil
}

func newProjectShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <project-uuid>",
		Short: "Show project details and structure summary",
		Example: `  latentcut project show proj-xxx
  latentcut project show proj-xxx --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectShow(args[0])
		},
	}
}

func runProjectShow(projectUUID string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	// Get canvas data for structure summary
	canvas, err := client.GetCanvasData(context.Background(), projectUUID)
	if err != nil {
		return fmt.Errorf("get canvas data: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(canvas)
		fmt.Println(string(data))
		return nil
	}

	parsed := canvas.ParseCanvas()

	fmt.Fprintf(os.Stdout, "Project: %s\n\n", projectUUID)
	fmt.Fprintf(os.Stdout, "Structure:\n")
	fmt.Fprintf(os.Stdout, "  Episodes:    %d\n", len(parsed.Episodes))
	fmt.Fprintf(os.Stdout, "  Shots:       %d\n", len(parsed.Shots))
	fmt.Fprintf(os.Stdout, "  Characters:  %d\n", parsed.Characters)
	fmt.Fprintf(os.Stdout, "  Locations:   %d\n", parsed.Locations)

	if len(parsed.Episodes) > 0 {
		fmt.Fprintln(os.Stdout, "\nEpisodes:")
		for _, ep := range parsed.Episodes {
			videoIcon := "⬜"
			switch ep.VideoStatus {
			case "done":
				videoIcon = "✅"
			case "processing":
				videoIcon = "🔄"
			case "pending":
				videoIcon = "⏳"
			case "failed":
				videoIcon = "❌"
			}
			fmt.Fprintf(os.Stdout, "  %s %s (shots: %d, video: %s)\n",
				videoIcon, ep.Title, len(ep.Shots), ep.VideoStatus)
			if ep.VideoURL != "" {
				fmt.Fprintf(os.Stdout, "     URL: %s\n", ep.VideoURL)
			}
		}
	}

	// Shot status summary
	var videoNone, videoPending, videoProcessing, videoDone, videoFailed int
	for _, shot := range parsed.Shots {
		switch shot.VideoStatus {
		case "done":
			videoDone++
		case "processing":
			videoProcessing++
		case "pending":
			videoPending++
		case "failed":
			videoFailed++
		default:
			videoNone++
		}
	}
	if len(parsed.Shots) > 0 {
		fmt.Fprintf(os.Stdout, "\nShot Video Status: done=%d, processing=%d, pending=%d, none=%d, failed=%d\n",
			videoDone, videoProcessing, videoPending, videoNone, videoFailed)
	}

	return nil
}

func newProjectDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <project-uuid>",
		Short: "Delete a project",
		Example: `  latentcut project delete proj-xxx
  latentcut project delete proj-xxx --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectDelete(args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	return cmd
}

func runProjectDelete(projectUUID string, force bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	if !force {
		fmt.Fprintf(os.Stderr, "Delete project %s? [y/N]: ", projectUUID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	if err := client.DeleteProject(context.Background(), projectUUID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Project deleted.")
	return nil
}

// Helper functions

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
