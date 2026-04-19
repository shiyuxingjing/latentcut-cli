package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

type resolvedShotTarget struct {
	ShotUUID      string
	EpisodeNumber int
	ShotNumber    int
	EpisodeTitle  string
}

type shotVideoJSONResult struct {
	ProjectUUID   string `json:"projectUuid"`
	ShotUUID      string `json:"shotUuid"`
	EpisodeNumber int    `json:"episodeNumber,omitempty"`
	ShotNumber    int    `json:"shotNumber,omitempty"`
	EpisodeTitle  string `json:"episodeTitle,omitempty"`
	PreviewID     string `json:"previewId,omitempty"`
	WorkflowRunID string `json:"workflowRunId,omitempty"`
	Status        string `json:"status"`
	VideoURL      string `json:"videoUrl,omitempty"`
}

// NewShotVideoCmd returns a high-level command for generating a single shot video.
func NewShotVideoCmd() *cobra.Command {
	var (
		wait        bool
		previewOnly bool
		episodeNum  int
		shotNum     int
		interval    time.Duration
		timeout     time.Duration
	)

	cmd := &cobra.Command{
		Use:   "shot-video <project-uuid> [shot-uuid]",
		Short: "Generate a single shot video with automatic dependency planning",
		Long: "Preview and execute the shot_video workflow for one shot. " +
			"You can target the shot by UUID or by episode/shot number.",
		Example: `  latentcut shot-video project-xxx shot-xxx
  latentcut shot-video project-xxx shot-xxx --wait
  latentcut shot-video project-xxx --episode 1 --shot 3
  latentcut shot-video project-xxx --episode 1 --shot 3 --preview-only --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShotVideo(args, wait, previewOnly, episodeNum, shotNum, interval, timeout)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until the target shot video is done or failed")
	cmd.Flags().BoolVar(&previewOnly, "preview-only", false, "Show the shot plan without executing it")
	cmd.Flags().IntVar(&episodeNum, "episode", 0, "Target episode number (requires --shot)")
	cmd.Flags().IntVar(&shotNum, "shot", 0, "Target shot number within the episode (requires --episode)")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "Polling interval when --wait is used")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "Maximum wait time when --wait is used")

	return cmd
}

func runShotVideo(
	args []string,
	wait bool,
	previewOnly bool,
	episodeNum int,
	shotNum int,
	interval time.Duration,
	timeout time.Duration,
) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	if interval <= 0 {
		return fmt.Errorf("--interval must be greater than 0")
	}
	if timeout <= 0 {
		return fmt.Errorf("--timeout must be greater than 0")
	}

	projectUUID := args[0]
	shotUUIDArg := ""
	if len(args) == 2 {
		shotUUIDArg = args[1]
	}

	if shotUUIDArg != "" && (episodeNum > 0 || shotNum > 0) {
		return fmt.Errorf("provide either a shot UUID or --episode/--shot, not both")
	}
	if (episodeNum > 0 && shotNum == 0) || (episodeNum == 0 && shotNum > 0) {
		return fmt.Errorf("--episode and --shot must be used together")
	}
	if shotUUIDArg == "" && (episodeNum == 0 || shotNum == 0) {
		return fmt.Errorf("provide a shot UUID or use --episode <n> --shot <n>")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	ctx := context.Background()

	target, err := resolveShotTarget(ctx, client, projectUUID, shotUUIDArg, episodeNum, shotNum)
	if err != nil {
		return err
	}

	preview, err := client.PreviewWorkflow(ctx, projectUUID, latentcut.WorkflowPreviewRequest{
		Mode: "shot_video",
		Target: &latentcut.WorkflowPreviewTarget{
			ShotUUID: target.ShotUUID,
		},
	})
	if err != nil {
		return fmt.Errorf("preview shot workflow: %w", err)
	}

	if previewOnly {
		if jsonOut {
			data, _ := json.Marshal(preview)
			fmt.Println(string(data))
			return nil
		}
		printShotVideoPreview(preview, target)
		return nil
	}

	if !preview.CanExecute {
		if !jsonOut {
			printShotVideoPreview(preview, target)
		}
		reason := "unknown"
		if preview.BlockReason != nil && *preview.BlockReason != "" {
			reason = *preview.BlockReason
		}
		return fmt.Errorf("shot workflow cannot execute: %s", reason)
	}

	if !jsonOut {
		printShotVideoPreview(preview, target)
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Executing shot workflow...")
	}

	execution, err := client.ExecuteWorkflow(ctx, projectUUID, preview.PreviewID)
	if err != nil {
		return fmt.Errorf("execute shot workflow: %w", err)
	}
	if !execution.Accepted {
		return fmt.Errorf("shot workflow was rejected by server")
	}

	if !wait {
		if jsonOut {
			data, _ := json.Marshal(shotVideoJSONResult{
				ProjectUUID:   projectUUID,
				ShotUUID:      target.ShotUUID,
				EpisodeNumber: target.EpisodeNumber,
				ShotNumber:    target.ShotNumber,
				EpisodeTitle:  target.EpisodeTitle,
				PreviewID:     preview.PreviewID,
				WorkflowRunID: execution.WorkflowRunID,
				Status:        "accepted",
			})
			fmt.Println(string(data))
			return nil
		}

		fmt.Fprintln(os.Stdout, "✅ Shot workflow accepted.")
		if target.EpisodeNumber > 0 && target.ShotNumber > 0 {
			fmt.Fprintf(os.Stdout, "Target: Episode %d, Shot %d (%s)\n",
				target.EpisodeNumber, target.ShotNumber, target.ShotUUID)
		} else {
			fmt.Fprintf(os.Stdout, "Target: %s\n", target.ShotUUID)
		}
		fmt.Fprintf(os.Stdout, "Run ID: %s\n", execution.WorkflowRunID)
		fmt.Fprintf(os.Stdout, "Track project progress: latentcut progress %s\n", projectUUID)
		fmt.Fprintf(os.Stdout, "Wait for this shot specifically: latentcut shot-video %s %s --wait\n", projectUUID, target.ShotUUID)
		return nil
	}

	if !jsonOut {
		fmt.Fprintf(os.Stdout, "Waiting for shot %s...\n", target.ShotUUID)
	}

	finalShot, err := waitForShotVideo(ctx, client, projectUUID, target.ShotUUID, interval, timeout, !jsonOut)
	if err != nil {
		return err
	}

	if jsonOut {
		data, _ := json.Marshal(shotVideoJSONResult{
			ProjectUUID:   projectUUID,
			ShotUUID:      target.ShotUUID,
			EpisodeNumber: firstNonZero(target.EpisodeNumber, finalShot.EpisodeNumber),
			ShotNumber:    firstNonZero(target.ShotNumber, finalShot.ShotNumber),
			EpisodeTitle:  pickEpisodeTitle(target.EpisodeTitle, finalShot.EpisodeTitle),
			PreviewID:     preview.PreviewID,
			WorkflowRunID: execution.WorkflowRunID,
			Status:        finalShot.VideoStatus,
			VideoURL:      finalShot.VideoURL,
		})
		fmt.Println(string(data))
		return nil
	}

	fmt.Fprintln(os.Stdout, "✅ Shot video is ready.")
	if finalShot.EpisodeNumber > 0 && finalShot.ShotNumber > 0 {
		fmt.Fprintf(os.Stdout, "Target: Episode %d, Shot %d\n", finalShot.EpisodeNumber, finalShot.ShotNumber)
	}
	if finalShot.VideoURL != "" {
		fmt.Fprintf(os.Stdout, "Video URL: %s\n", finalShot.VideoURL)
	}

	return nil
}

func resolveShotTarget(
	ctx context.Context,
	client *latentcut.Client,
	projectUUID string,
	shotUUID string,
	episodeNum int,
	shotNum int,
) (*resolvedShotTarget, error) {
	gallery, galleryErr := client.GetGallery(ctx, projectUUID)

	if shotUUID != "" {
		target := &resolvedShotTarget{ShotUUID: shotUUID}
		if galleryErr == nil {
			if shot := findGalleryShotByUUID(gallery, shotUUID); shot != nil {
				target.EpisodeNumber = shot.EpisodeNumber
				target.ShotNumber = shot.ShotNumber
				target.EpisodeTitle = shot.EpisodeTitle
			}
		}
		return target, nil
	}

	if galleryErr != nil {
		return nil, fmt.Errorf(
			"resolve shot by episode/shot requires gallery support: %w",
			galleryErr,
		)
	}

	shot := findGalleryShotByNumber(gallery, episodeNum, shotNum)
	if shot == nil {
		return nil, fmt.Errorf(
			"shot not found: episode %d shot %d in project %s. Run `latentcut gallery %s` to inspect available shots",
			episodeNum,
			shotNum,
			projectUUID,
			projectUUID,
		)
	}

	return &resolvedShotTarget{
		ShotUUID:      shot.UUID,
		EpisodeNumber: shot.EpisodeNumber,
		ShotNumber:    shot.ShotNumber,
		EpisodeTitle:  shot.EpisodeTitle,
	}, nil
}

func printShotVideoPreview(preview *latentcut.WorkflowPreview, target *resolvedShotTarget) {
	fmt.Fprintf(os.Stdout, "Plan: shot_video (%s)\n", preview.PreviewID)
	if target.EpisodeNumber > 0 && target.ShotNumber > 0 {
		if target.EpisodeTitle != "" {
			fmt.Fprintf(os.Stdout, "Target: Episode %d \"%s\", Shot %d (%s)\n",
				target.EpisodeNumber, target.EpisodeTitle, target.ShotNumber, target.ShotUUID)
		} else {
			fmt.Fprintf(os.Stdout, "Target: Episode %d, Shot %d (%s)\n",
				target.EpisodeNumber, target.ShotNumber, target.ShotUUID)
		}
	} else {
		fmt.Fprintf(os.Stdout, "Target: %s\n", target.ShotUUID)
	}

	fmt.Fprintf(os.Stdout, "Estimated: %d credits · ~%ds\n",
		preview.Cost.EstimatedCredits, preview.Timing.EstimatedSeconds)

	if len(preview.Outputs) > 0 {
		fmt.Fprintln(os.Stdout, "Will generate:")
		for _, output := range preview.Outputs {
			fmt.Fprintf(os.Stdout, "  • %d × %s\n", output.Count, output.Type)
		}
	}

	if len(preview.MissingDependencies) > 0 {
		fmt.Fprintln(os.Stdout, "Missing dependencies:")
		for _, dep := range preview.MissingDependencies {
			name := dep.Name
			if name == "" {
				name = dep.UUID
			}
			fmt.Fprintf(os.Stdout, "  • %s: %s\n", dep.Type, name)
		}
	}

	if preview.Explain.Summary != "" {
		fmt.Fprintf(os.Stdout, "Summary: %s\n", preview.Explain.Summary)
	}
	if preview.Explain.UserSafeSummary != "" {
		fmt.Fprintf(os.Stdout, "Note: %s\n", preview.Explain.UserSafeSummary)
	}
}

func waitForShotVideo(
	ctx context.Context,
	client *latentcut.Client,
	projectUUID string,
	shotUUID string,
	interval time.Duration,
	timeout time.Duration,
	verbose bool,
) (*latentcut.GalleryShot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastStatus := ""

	for {
		shot, err := fetchShotStatus(ctx, client, projectUUID, shotUUID)
		if err == nil && shot != nil {
			if verbose && shot.VideoStatus != "" && shot.VideoStatus != lastStatus {
				fmt.Fprintf(os.Stdout, "  status: %s\n", shot.VideoStatus)
				lastStatus = shot.VideoStatus
			}

			switch shot.VideoStatus {
			case "done":
				return shot, nil
			case "failed":
				return nil, fmt.Errorf("shot video generation failed for %s", shotUUID)
			}
		}

		select {
		case <-ctx.Done():
			if err != nil {
				return nil, fmt.Errorf("timeout waiting for shot video (last poll error: %w)", err)
			}
			return nil, fmt.Errorf("timeout waiting for shot video %s", shotUUID)
		case <-ticker.C:
		}
	}
}

func fetchShotStatus(
	ctx context.Context,
	client *latentcut.Client,
	projectUUID string,
	shotUUID string,
) (*latentcut.GalleryShot, error) {
	gallery, err := client.GetGallery(ctx, projectUUID)
	if err == nil {
		if shot := findGalleryShotByUUID(gallery, shotUUID); shot != nil {
			return shot, nil
		}
	}

	canvas, canvasErr := client.GetCanvasData(ctx, projectUUID)
	if canvasErr != nil {
		if err != nil {
			return nil, err
		}
		return nil, canvasErr
	}

	for key, card := range canvas.Cards {
		if key != shotUUID && card.Raw.ShotUUID != shotUUID {
			continue
		}
		status := card.VideoStatus
		if status == "" {
			status = card.Raw.VideoStatus
		}
		videoURL := card.VideoURL
		if videoURL == "" {
			videoURL = card.Raw.VideoURL
		}
		return &latentcut.GalleryShot{
			UUID:        shotUUID,
			VideoStatus: status,
			VideoURL:    videoURL,
		}, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("shot %s not found in project %s", shotUUID, projectUUID)
}

func findGalleryShotByUUID(gallery *latentcut.Gallery, shotUUID string) *latentcut.GalleryShot {
	if gallery == nil {
		return nil
	}
	for i := range gallery.Shots {
		if gallery.Shots[i].UUID == shotUUID {
			return &gallery.Shots[i]
		}
	}
	return nil
}

func findGalleryShotByNumber(gallery *latentcut.Gallery, episodeNum int, shotNum int) *latentcut.GalleryShot {
	if gallery == nil {
		return nil
	}
	for i := range gallery.Shots {
		if gallery.Shots[i].EpisodeNumber == episodeNum && gallery.Shots[i].ShotNumber == shotNum {
			return &gallery.Shots[i]
		}
	}
	return nil
}

func firstNonZero(primary int, fallback int) int {
	if primary > 0 {
		return primary
	}
	return fallback
}

func pickEpisodeTitle(primary string, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
