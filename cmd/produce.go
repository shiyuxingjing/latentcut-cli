package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/novelo-ai/novelo-cli/internal/merge"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// NewProduceCmd returns the produce subcommand.
func NewProduceCmd() *cobra.Command {
	var (
		style     string
		outputDir string
		noMerge   bool
	)

	cmd := &cobra.Command{
		Use:   "produce <input-file>",
		Short: "Produce short drama video from a novel text file via latentCut-server",
		Long:  "Reads a novel text file and produces short drama videos through latentCut-server: AI parsing, asset generation, video production, and episode assembly.",
		Example: `  latentcut produce novel.txt
  latentcut produce novel.txt --style "精致国漫/仙侠风"
  latentcut produce novel.txt --output-dir ./my-drama --no-merge`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProduce(args[0], style, outputDir, noMerge)
		},
	}

	cmd.Flags().StringVar(&style, "style", "", "Visual style (e.g. 精致国漫/仙侠风, 写实风格)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: novelo-output)")
	cmd.Flags().BoolVar(&noMerge, "no-merge", false, "Skip download, just print video URLs")

	return cmd
}

func runProduce(inputFile, style, outputDir string, noMerge bool) error {
	// 1. Load config and validate auth
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if outputDir != "" {
		cfg.OutputDir = outputDir
	}

	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	// 2. Read input file
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}
	novelContent := string(inputData)
	charCount := utf8.RuneCountInString(novelContent)
	if charCount < 100 {
		return fmt.Errorf("novel text too short (%d chars), minimum 100 characters", charCount)
	}

	title := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))

	// 3. Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted. Closing...")
		cancel()
	}()

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	// 4. Create project
	fmt.Fprintf(os.Stderr, "Creating project \"%s\" on %s...\n", title, cfg.LatentCutURL)
	projectData, err := client.CreateProject(ctx, title, novelContent, style)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	projectUUID := projectData.ProjectUUID
	taskUUID := projectData.TaskUUID
	fmt.Fprintf(os.Stderr, "Project: %s\nTask: %s\n", projectUUID, taskUUID)

	// 5. Subscribe to SSE for AI parsing progress
	fmt.Fprintln(os.Stderr, "\n--- Phase 1: AI Parsing ---")
	bar := newProgressBar("Parsing novel")

	dramaDone := false
	var dramaErr error

	err = client.SubscribeSSEWithRetry(ctx, projectUUID, taskUUID, func(event latentcut.SSEEvent) bool {
		switch event.Name {
		case latentcut.EventDramaProgress:
			var p latentcut.DramaProgressEvent
			if json.Unmarshal([]byte(event.Data), &p) == nil {
				updateBar(bar, int(p.Progress), p.CurrentStep)
			}
		case latentcut.EventDramaDone:
			_ = bar.Finish()
			fmt.Fprintln(os.Stderr, "\nAI parsing complete!")
			dramaDone = true
			return false
		case latentcut.EventDramaFailed:
			var f latentcut.DramaFailedEvent
			if json.Unmarshal([]byte(event.Data), &f) == nil {
				dramaErr = fmt.Errorf("AI parsing failed: %s (stage: %s)", f.Error, f.Stage)
			} else {
				dramaErr = fmt.Errorf("AI parsing failed: %s", event.Data)
			}
			return false
		}
		return true
	}, 3)

	if dramaErr != nil {
		return dramaErr
	}
	if err != nil && !dramaDone {
		return fmt.Errorf("SSE connection: %w", err)
	}

	// 6. Get canvas data
	fmt.Fprintln(os.Stderr, "\nFetching project structure...")
	canvas, err := client.GetCanvasData(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("get canvas data: %w", err)
	}

	parsed := canvas.ParseCanvas()
	fmt.Fprintf(os.Stderr, "  %d episodes, %d shots, %d characters, %d locations\n",
		len(parsed.Episodes), len(parsed.Shots), parsed.Characters, parsed.Locations)

	if len(parsed.Shots) == 0 {
		return fmt.Errorf("no shots found in project — AI parsing may have produced empty results")
	}

	// 7. Generate prerequisite assets (character images, voices, location images)
	fmt.Fprintln(os.Stderr, "\n--- Phase 2a: Character & Location Assets ---")

	if len(parsed.CharacterUUIDs) > 0 {
		fmt.Fprintf(os.Stderr, "  Generating %d character images...\n", len(parsed.CharacterUUIDs))
		for _, uuid := range parsed.CharacterUUIDs {
			if err := client.GenerateCharacterImage(ctx, projectUUID, uuid); err != nil {
				fmt.Fprintf(os.Stderr, "    character image %s: %v\n", uuid, err)
			}
		}
		fmt.Fprintf(os.Stderr, "  Generating %d character voices...\n", len(parsed.CharacterUUIDs))
		for _, uuid := range parsed.CharacterUUIDs {
			if err := client.GenerateCharacterVoice(ctx, projectUUID, uuid); err != nil {
				fmt.Fprintf(os.Stderr, "    character voice %s: %v\n", uuid, err)
			}
		}
	}
	if len(parsed.LocationUUIDs) > 0 {
		fmt.Fprintf(os.Stderr, "  Generating %d location images...\n", len(parsed.LocationUUIDs))
		for _, uuid := range parsed.LocationUUIDs {
			if err := client.GenerateLocationImage(ctx, projectUUID, uuid); err != nil {
				fmt.Fprintf(os.Stderr, "    location image %s: %v\n", uuid, err)
			}
		}
	}

	// Wait for character/location assets to complete
	if len(parsed.CharacterUUIDs) > 0 || len(parsed.LocationUUIDs) > 0 {
		fmt.Fprintln(os.Stderr, "  Waiting for character & location assets...")
		assetBar := newProgressBar("Assets")
		totalAssets := len(parsed.CharacterUUIDs)*2 + len(parsed.LocationUUIDs) // images + voices + locations
		for waited := 0; waited < 120; waited++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}

			tasks, err := client.GetPendingTasks(ctx, projectUUID)
			if err != nil {
				continue
			}

			// Count remaining character/location tasks
			remaining := 0
			for _, t := range tasks {
				switch t.TaskType {
				case "character_image", "character_voice", "locationtime_image":
					remaining++
				}
			}
			done := totalAssets - remaining
			if done < 0 {
				done = 0
			}
			if totalAssets > 0 {
				pct := (done * 100) / totalAssets
				updateBar(assetBar, pct, fmt.Sprintf("%d/%d assets", done, totalAssets))
			}
			if remaining == 0 {
				break
			}
		}
		_ = assetBar.Finish()
		fmt.Fprintln(os.Stderr, "\n  Character & location assets ready!")
	}

	// 8. Preview and trigger batch generation
	fmt.Fprintln(os.Stderr, "\n--- Phase 2b: Shot Generation (keyframes + audio + video) ---")

	preview, err := client.PreviewShotBatch(ctx, projectUUID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (preview unavailable: %v, proceeding anyway)\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "  %d shots to generate, estimated cost: %d credits\n",
			len(preview.Candidates), preview.TotalCredits)
	}

	if preview != nil && len(preview.Candidates) == 0 {
		fmt.Fprintln(os.Stderr, "  No shots eligible for batch generation yet.")
		fmt.Fprintln(os.Stderr, "  This may mean character/location assets are still processing.")
		fmt.Fprintln(os.Stderr, "  Check the project in the web UI for details.")
		return nil
	}

	batchData, err := client.CreateShotBatch(ctx, projectUUID, "all", 0)
	if err != nil {
		return fmt.Errorf("create shot batch: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  Batch started: %d shots accepted\n", batchData.AcceptedCount)

	// 9. Poll for batch generation completion
	genBar := newProgressBar("Generating assets")
	err = waitForGeneration(ctx, client, projectUUID, batchData.AcceptedCount, genBar)
	if err != nil {
		return fmt.Errorf("generation: %w", err)
	}
	_ = genBar.Finish()
	fmt.Fprintln(os.Stderr, "\nAsset generation complete!")

	// 9. Generate episode videos
	fmt.Fprintln(os.Stderr, "\n--- Phase 3: Episode Video Assembly ---")
	// Re-fetch canvas to get latest shot statuses
	canvas, err = client.GetCanvasData(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("refresh canvas: %w", err)
	}
	parsed = canvas.ParseCanvas()

	var episodeVideoURLs []string
	for i, ep := range parsed.Episodes {
		// Check if all shots have video
		allDone := true
		for _, shot := range ep.Shots {
			if shot.VideoStatus != "done" {
				allDone = false
				break
			}
		}
		if !allDone {
			fmt.Fprintf(os.Stderr, "  Episode %d (%s): not all shots have video, skipping\n", i+1, ep.Title)
			continue
		}
		if ep.VideoStatus == "done" && ep.VideoURL != "" {
			fmt.Fprintf(os.Stderr, "  Episode %d (%s): already assembled\n", i+1, ep.Title)
			episodeVideoURLs = append(episodeVideoURLs, ep.VideoURL)
			continue
		}

		fmt.Fprintf(os.Stderr, "  Episode %d (%s): assembling %d shots...\n", i+1, ep.Title, len(ep.Shots))
		if err := client.GenerateEpisodeVideo(ctx, projectUUID, ep.UUID); err != nil {
			fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
			continue
		}

		// Poll for episode video completion via SSE
		epDone := false
		_ = client.SubscribeSSEWithRetry(ctx, projectUUID, "", func(event latentcut.SSEEvent) bool {
			if event.Name == latentcut.EventEpisodeVideoDone {
				var r latentcut.ResourceDoneEvent
				if json.Unmarshal([]byte(event.Data), &r) == nil && r.EntityUUID == ep.UUID {
					episodeVideoURLs = append(episodeVideoURLs, r.FileURL)
					epDone = true
					return false
				}
			}
			return true
		}, 2)

		if !epDone {
			// Fallback: poll canvas data
			for attempts := 0; attempts < 60; attempts++ {
				time.Sleep(5 * time.Second)
				refreshed, err := client.GetCanvasData(ctx, projectUUID)
				if err != nil {
					continue
				}
				reParsed := refreshed.ParseCanvas()
				for _, re := range reParsed.Episodes {
					if re.UUID == ep.UUID && re.VideoStatus == "done" && re.VideoURL != "" {
						episodeVideoURLs = append(episodeVideoURLs, re.VideoURL)
						epDone = true
						break
					}
				}
				if epDone {
					break
				}
			}
		}

		if epDone {
			fmt.Fprintf(os.Stderr, "    Done!\n")
		} else {
			fmt.Fprintf(os.Stderr, "    Timeout waiting for episode video\n")
		}
	}

	if len(episodeVideoURLs) == 0 {
		fmt.Fprintln(os.Stderr, "\nNo episode videos were produced.")
		fmt.Fprintln(os.Stderr, "Check the project in the web UI for details.")
		return nil
	}

	// 10. Output results
	fmt.Fprintf(os.Stderr, "\n--- Results: %d episode videos ---\n", len(episodeVideoURLs))
	for i, url := range episodeVideoURLs {
		fmt.Fprintf(os.Stderr, "  Episode %d: %s\n", i+1, url)
	}

	if noMerge {
		for _, url := range episodeVideoURLs {
			fmt.Println(url)
		}
		return nil
	}

	// 11. Download and merge
	ffmpegPath, err := merge.DetectFFmpeg()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n\nVideo URLs printed above.\n", err)
		return nil
	}

	episodesDir := filepath.Join(cfg.OutputDir, "episodes")
	fmt.Fprintf(os.Stderr, "\nDownloading %d episode videos...\n", len(episodeVideoURLs))

	// Convert URLs to ShotResult for reuse of existing download logic
	var shots []struct {
		URL string
		Num int
	}
	for i, url := range episodeVideoURLs {
		shots = append(shots, struct {
			URL string
			Num int
		}{url, i + 1})
	}

	if err := os.MkdirAll(episodesDir, 0755); err != nil {
		return fmt.Errorf("create episodes dir: %w", err)
	}

	var downloadedFiles []string
	for _, s := range shots {
		dest := filepath.Join(episodesDir, fmt.Sprintf("episode_%02d.mp4", s.Num))
		fmt.Fprintf(os.Stderr, "  Downloading episode %d...\n", s.Num)
		if err := downloadURL(ctx, s.URL, dest); err != nil {
			fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
			continue
		}
		downloadedFiles = append(downloadedFiles, dest)
	}

	if len(downloadedFiles) == 0 {
		return fmt.Errorf("all downloads failed")
	}

	if len(downloadedFiles) == 1 {
		fmt.Fprintf(os.Stderr, "\nOutput: %s\n", downloadedFiles[0])
		return nil
	}

	// Merge multiple episodes
	outputPath := filepath.Join(cfg.OutputDir, "drama.mp4")
	fmt.Fprintf(os.Stderr, "Merging %d episodes into %s...\n", len(downloadedFiles), outputPath)
	if err := merge.MergeVideos(ctx, ffmpegPath, downloadedFiles, outputPath); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nOutput: %s\n", outputPath)
	return nil
}

// waitForGeneration polls pending tasks until all are complete or timeout.
func waitForGeneration(ctx context.Context, client *latentcut.Client, projectUUID string, totalShots int, bar *progressbar.ProgressBar) error {
	maxWait := 30 * time.Minute
	deadline := time.Now().Add(maxWait)
	pollInterval := 10 * time.Second
	initialTotal := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %v waiting for generation", maxWait)
		}

		tasks, err := client.GetPendingTasks(ctx, projectUUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n  (poll error: %v, retrying...)\n", err)
			time.Sleep(pollInterval)
			continue
		}

		if len(tasks) == 0 {
			updateBar(bar, 100, "Complete")
			return nil
		}

		// Track initial total on first poll
		if initialTotal == 0 {
			initialTotal = len(tasks)
		}

		remaining := len(tasks)
		done := initialTotal - remaining
		if done < 0 {
			done = 0
		}
		pct := 0
		if initialTotal > 0 {
			pct = (done * 100) / initialTotal
		}
		updateBar(bar, pct, fmt.Sprintf("%d/%d tasks done", done, initialTotal))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func newProgressBar(desc string) *progressbar.ProgressBar {
	return progressbar.NewOptions(100,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("  "+desc),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
	)
}

func updateBar(bar *progressbar.ProgressBar, pct int, desc string) {
	if desc != "" {
		bar.Describe("  " + desc)
	}
	current := int(bar.State().CurrentPercent * 100)
	delta := pct - current
	if delta > 0 {
		_ = bar.Add(delta)
	}
}

// downloadURL downloads a URL to a local file.
func downloadURL(ctx context.Context, rawURL, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: status %d", rawURL, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
