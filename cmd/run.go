package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/novelo-ai/novelo-cli/internal/client"
	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/merge"
	"github.com/novelo-ai/novelo-cli/internal/progress"
	"github.com/novelo-ai/novelo-cli/internal/types"
	"github.com/spf13/cobra"
)

// NewRunCmd returns the run subcommand.
func NewRunCmd() *cobra.Command {
	var (
		style     string
		outputDir string
		noMerge   bool
		serverURL string
	)

	cmd := &cobra.Command{
		Use:   "run <input-file>",
		Short: "Run the Novelo AI pipeline on an input file",
		Long:  "Reads a novel text file and runs the full Novelo AI pipeline: narration rewrite, episode split, asset extraction, and shot/video generation.",
		Example: `  latentcut run input.txt
  latentcut run input.txt --style cinematic --output-dir ./output
  latentcut run input.txt --json
  latentcut run input.txt --no-merge`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(args[0], style, outputDir, noMerge, serverURL)
		},
	}

	cmd.Flags().StringVar(&style, "style", "", "Visual style override (default from config)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: novelo-output)")
	cmd.Flags().BoolVar(&noMerge, "no-merge", false, "Skip ffmpeg merge step, just print video URLs")
	cmd.Flags().StringVar(&serverURL, "server-url", "", "Override server URL")

	return cmd
}

func runPipeline(inputFile, style, outputDir string, noMerge bool, serverURLFlag string) error {
	// 1. Load config, apply flag overrides
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if serverURLFlag != "" {
		cfg.ServerURL = serverURLFlag
	}
	if outputDir != "" {
		cfg.OutputDir = outputDir
	}
	if style == "" {
		style = ""
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key not set. Run: latentcut config set api-key <your-key>")
	}

	// 2. Read input file
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read input file %s: %w", inputFile, err)
	}

	// 3. Set up context with SIGINT/SIGTERM handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted. Closing connection...")
		cancel()
	}()

	// 4. Trigger pipeline via HTTP
	httpClient := client.NewHTTPClient(cfg.ServerURL, cfg.APIKey)
	req := types.PipelineRunRequest{
		InputText: string(inputData),
		Style:     style,
	}

	if !jsonOut {
		fmt.Fprintf(os.Stderr, "Starting pipeline on %s...\n", cfg.ServerURL)
	}

	pipelineResp, err := httpClient.TriggerPipeline(ctx, req)
	if err != nil {
		return fmt.Errorf("trigger pipeline: %w", err)
	}

	runID := pipelineResp.RunID
	if !jsonOut {
		fmt.Fprintf(os.Stderr, "Run ID: %s\n", runID)
	}

	// 5. Connect WebSocket and stream progress
	var display *progress.ProgressDisplay
	var jsonWriter *progress.JSONLWriter

	if jsonOut {
		jsonWriter = progress.NewJSONLWriter()
	} else {
		display = progress.NewProgressDisplay(verbose)
	}

	// Collect shot results from complete event
	var shotResults []types.ShotResult

	wsClient := client.NewWSClient(cfg.ServerURL, runID, cfg.APIKey)
	wsClient.OnMessage = func(event progress.WsProgressEvent) {
		if jsonOut {
			jsonWriter.HandleEvent(event)
		} else {
			display.HandleEvent(event)
		}

		// Extract shot results from complete event data
		if event.Type == "complete" && event.Data != nil {
			extractShots(event.Data, &shotResults)
		}
	}
	wsClient.OnError = func(err error) {
		if verbose {
			fmt.Fprintf(os.Stderr, "ws error: %v\n", err)
		}
	}

	if err := wsClient.Connect(ctx); err != nil {
		if ctx.Err() != nil {
			return nil // user cancelled
		}
		return fmt.Errorf("websocket: %w", err)
	}

	// 6. Handle --no-merge: just print URLs
	if noMerge || len(shotResults) == 0 {
		if len(shotResults) > 0 {
			fmt.Fprintln(os.Stderr, "\nVideo URLs:")
			for _, s := range shotResults {
				fmt.Printf("%s\n", s.VideoURL)
			}
		}
		return nil
	}

	// 7. Download videos and merge with ffmpeg
	ffmpegPath, err := merge.DetectFFmpeg()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n\nVideo URLs:\n", err)
		for _, s := range shotResults {
			fmt.Printf("%s\n", s.VideoURL)
		}
		return nil
	}

	shotsDir := filepath.Join(cfg.OutputDir, "shots")
	if !jsonOut {
		fmt.Fprintf(os.Stderr, "\nDownloading %d videos...\n", len(shotResults))
	}

	files, err := merge.DownloadVideos(ctx, shotResults, shotsDir)
	if err != nil {
		return fmt.Errorf("download videos: %w", err)
	}

	outputPath := filepath.Join(cfg.OutputDir, "drama.mp4")
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if !jsonOut {
		fmt.Fprintf(os.Stderr, "Merging %d clips into %s...\n", len(files), outputPath)
	}

	if err := merge.MergeVideos(ctx, ffmpegPath, files, outputPath); err != nil {
		return fmt.Errorf("merge videos: %w", err)
	}

	if jsonOut {
		out := map[string]string{"type": "output", "path": outputPath}
		data, _ := json.Marshal(out)
		fmt.Println(string(data))
	} else {
		fmt.Printf("\nOutput: %s\n", outputPath)
	}

	return nil
}

// extractShots attempts to parse shot results from the complete event's data field.
func extractShots(data any, shots *[]types.ShotResult) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}

	// Try RunFullData format (from /pipeline/run-full)
	var fullData types.RunFullData
	if err := json.Unmarshal(raw, &fullData); err == nil && len(fullData.Shots) > 0 {
		*shots = fullData.Shots
		return
	}

	// Try direct array of ShotResult
	var direct []types.ShotResult
	if err := json.Unmarshal(raw, &direct); err == nil && len(direct) > 0 {
		*shots = direct
		return
	}

	// Try wrapped in {"shots": [...]}
	var wrapped struct {
		Shots []types.ShotResult `json:"shots"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Shots) > 0 {
		*shots = wrapped.Shots
	}
}
