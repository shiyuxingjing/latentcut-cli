package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

func newProjectCreateCmd() *cobra.Command {
	var style string
	var noWait bool

	cmd := &cobra.Command{
		Use:   "create <input-file>",
		Short: "Create a project and run AI parsing only (no asset generation)",
		Long:  "Creates a project from novel text, runs AI parsing (episode split, character/location extraction, shot generation), then stops. Use 'project assets' to inspect results, 'generate' to create resources.",
		Example: `  latentcut project create novel.txt
  latentcut project create novel.txt --style "精致国漫/仙侠风"
  latentcut project create novel.txt --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectCreate(args[0], style, noWait)
		},
	}

	cmd.Flags().StringVar(&style, "style", "", "Visual style (e.g. 精致国漫/仙侠风, 写实风格)")
	cmd.Flags().BoolVar(&noWait, "wait", false, "Wait for AI parsing to complete (synchronous)")

	return cmd
}

func runProjectCreate(inputFile, style string, wait bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	// Read input file
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

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted.")
		cancel()
	}()

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	// Create project
	fmt.Fprintf(os.Stderr, "Creating project \"%s\" on %s...\n", title, cfg.LatentCutURL)
	projectData, err := client.CreateProject(ctx, title, novelContent, style)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	projectUUID := projectData.ProjectUUID
	taskUUID := projectData.TaskUUID
	fmt.Fprintf(os.Stderr, "Project: %s\nTask: %s\n", projectUUID, taskUUID)

	if !wait {
		if jsonOut {
			out := map[string]any{
				"project_uuid": projectUUID,
				"task_uuid":    taskUUID,
			}
			data, _ := json.Marshal(out)
			fmt.Println(string(data))
		} else {
			fmt.Fprintf(os.Stdout, "\nProject created: %s\n", projectUUID)
			fmt.Fprintln(os.Stdout, "Wait skipped. AI parsing running in background.")
		}
		return nil
	}

	// Wait for AI parsing via SSE
	fmt.Fprintln(os.Stderr, "\nWaiting for AI parsing...")
	bar := newProgressBar("AI Parsing")

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
			finishBar(bar)
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

	// Show structure summary
	canvas, err := client.GetCanvasData(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("get canvas: %w", err)
	}
	parsed := canvas.ParseCanvas()

	if jsonOut {
		out := map[string]any{
			"project_uuid": projectUUID,
			"task_uuid":    taskUUID,
			"episodes":     len(parsed.Episodes),
			"shots":        len(parsed.Shots),
			"characters":   parsed.Characters,
			"locations":    parsed.Locations,
		}
		data, _ := json.Marshal(out)
		fmt.Println(string(data))
	} else {
		fmt.Fprintf(os.Stdout, "\nProject created: %s\n", projectUUID)
		fmt.Fprintf(os.Stdout, "  Episodes:   %d\n", len(parsed.Episodes))
		fmt.Fprintf(os.Stdout, "  Shots:      %d\n", len(parsed.Shots))
		fmt.Fprintf(os.Stdout, "  Characters: %d\n", parsed.Characters)
		fmt.Fprintf(os.Stdout, "  Locations:  %d\n", parsed.Locations)
		fmt.Fprintln(os.Stdout, "\nNext steps:")
		fmt.Fprintf(os.Stdout, "  latentcut project assets %s    # View characters/locations\n", projectUUID)
		fmt.Fprintf(os.Stdout, "  latentcut produce %s            # Continue full pipeline\n", inputFile)
	}

	return nil
}
