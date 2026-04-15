package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewGenerateCmd returns the generate subcommand.
func NewGenerateCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:   "generate <type> <project-uuid> <entity-uuid>",
		Short: "Regenerate a single resource (image, voice, audio, video)",
		Long: `Trigger generation for a specific resource within a project.

Types:
  character-image     Generate character reference image
  character-voice     Generate character TTS voice
  location-image      Generate location/scene image
  keyframe-image      Generate keyframe image for a shot
  dialogue-audio      Generate dialogue TTS audio
  shot-video          Generate video for a shot
  shot-storyboard     Generate storyboard for a shot
  episode-video       Assemble episode video from shots`,
		Example: `  latentcut generate character-image proj-xxx char-yyy
  latentcut generate shot-video proj-xxx shot-zzz --wait
  latentcut generate episode-video proj-xxx episode-aaa`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(args[0], args[1], args[2], wait)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for generation to complete (polls every 5s)")

	return cmd
}

func runGenerateCmd(genType, projectUUID, entityUUID string, wait bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Generating %s for %s in project %s...\n", genType, entityUUID, projectUUID)

	switch genType {
	case "character-image":
		err = client.GenerateCharacterImage(ctx, projectUUID, entityUUID)
	case "character-voice":
		err = client.GenerateCharacterVoice(ctx, projectUUID, entityUUID)
	case "location-image":
		err = client.GenerateLocationImage(ctx, projectUUID, entityUUID)
	case "keyframe-image":
		err = client.GenerateKeyframeImage(ctx, projectUUID, entityUUID)
	case "dialogue-audio":
		err = client.GenerateDialogueAudio(ctx, projectUUID, entityUUID)
	case "shot-video":
		err = client.GenerateShotVideo(ctx, projectUUID, entityUUID)
	case "shot-storyboard":
		err = client.GenerateShotStoryboard(ctx, projectUUID, entityUUID)
	case "episode-video":
		err = client.GenerateEpisodeVideo(ctx, projectUUID, entityUUID)
	default:
		return fmt.Errorf("unknown type: %s\nValid types: character-image, character-voice, location-image, keyframe-image, dialogue-audio, shot-video, shot-storyboard, episode-video", genType)
	}

	if err != nil {
		return fmt.Errorf("trigger generation: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Generation triggered.")

	if wait {
		fmt.Fprintln(os.Stderr, "Waiting for completion...")
		for i := 0; i < 360; i++ { // max 30 minutes
			time.Sleep(5 * time.Second)
			tasks, err := client.GetPendingTasks(ctx, projectUUID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  (poll error: %v, retrying...)\n", err)
				continue
			}
			remaining := 0
			for _, t := range tasks {
				if matchTaskType(genType, t.TaskType) {
					remaining++
				}
			}
			if remaining == 0 {
				fmt.Fprintln(os.Stderr, "Done! ✅")
				return nil
			}
			fmt.Fprintf(os.Stderr, "  Still %d task(s) pending...\n", remaining)
		}
		return fmt.Errorf("timeout waiting for generation (30 min)")
	}

	return nil
}

// matchTaskType maps CLI type names to server task_type values.
func matchTaskType(cliType, serverType string) bool {
	mapping := map[string]string{
		"character-image": "character_image",
		"character-voice": "character_voice",
		"location-image":  "locationtime_image",
		"keyframe-image":  "keyframe_image",
		"dialogue-audio":  "dialogue_audio",
		"shot-video":      "shot_video",
		"shot-storyboard": "shot_storyboard",
		"episode-video":   "episode_video",
	}
	if mapped, ok := mapping[cliType]; ok {
		return mapped == serverType
	}
	return false
}
