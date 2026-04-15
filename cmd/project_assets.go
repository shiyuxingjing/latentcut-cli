package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

func newProjectAssetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "assets <project-uuid>",
		Short: "List all characters, locations, and their generated resource URLs",
		Long:  "Show all characters and locations with their image/voice generation status and URLs. Use this to check if asset generation is done and preview the results.",
		Example: `  latentcut project assets proj-xxx
  latentcut project assets proj-xxx --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectAssets(args[0])
		},
	}
}

func runProjectAssets(projectUUID string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	canvas, err := client.GetCanvasData(context.Background(), projectUUID)
	if err != nil {
		return fmt.Errorf("get canvas data: %w", err)
	}

	type assetInfo struct {
		Type        string `json:"type"`
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		ImageStatus string `json:"image_status"`
		ImageURL    string `json:"image_url,omitempty"`
		TTSStatus   string `json:"tts_status,omitempty"`
	}

	// Collect character and location data, merging across multiple cards per UUID.
	// Canvas has separate cards like char-img-xxx (with image) and char-tts-xxx (with TTS).
	charMap := make(map[string]*assetInfo)
	locMap := make(map[string]*assetInfo)

	for key, card := range canvas.Cards {
		if strings.HasPrefix(key, "char-") {
			uuid := card.Raw.CharacterUUID
			if uuid == "" {
				continue
			}

			a, exists := charMap[uuid]
			if !exists {
				a = &assetInfo{Type: "character", UUID: uuid}
				charMap[uuid] = a
			}

			// Merge: take non-empty values
			if a.Name == "" {
				name := card.CharacterName
				if name == "" {
					name = card.Raw.CharacterName
				}
				if name == "" {
					name = card.Name
				}
				a.Name = name
			}
			if a.ImageStatus == "" || a.ImageStatus == "idle" {
				if s := card.ImageStatus; s != "" {
					a.ImageStatus = s
				} else if s := card.Raw.ImageStatus; s != "" {
					a.ImageStatus = s
				}
			}
			if a.ImageURL == "" {
				if u := card.Image; u != "" {
					a.ImageURL = u
				} else if u := card.Raw.ImageURL; u != "" {
					a.ImageURL = u
				}
			}
			if a.TTSStatus == "" || a.TTSStatus == "idle" {
				if s := card.Raw.TTSStatus; s != "" {
					a.TTSStatus = s
				}
			}
		}

		if strings.HasPrefix(key, "loc-") {
			uuid := card.Raw.LocationUUID
			if uuid == "" {
				continue
			}

			a, exists := locMap[uuid]
			if !exists {
				a = &assetInfo{Type: "location", UUID: uuid}
				locMap[uuid] = a
			}

			if a.Name == "" {
				name := card.Location
				if name == "" {
					name = card.Raw.Location
				}
				if name == "" {
					name = card.Description
				}
				a.Name = name
			}
			if a.ImageStatus == "" || a.ImageStatus == "idle" {
				if s := card.ImageStatus; s != "" {
					a.ImageStatus = s
				} else if s := card.Raw.ImageStatus; s != "" {
					a.ImageStatus = s
				}
			}
			if a.ImageURL == "" {
				if u := card.Image; u != "" {
					a.ImageURL = u
				} else if u := card.Raw.ImageURL; u != "" {
					a.ImageURL = u
				}
			}
		}
	}

	// Build final list
	var assets []assetInfo
	for _, a := range charMap {
		assets = append(assets, *a)
	}
	for _, a := range locMap {
		assets = append(assets, *a)
	}

	if jsonOut {
		data, _ := json.Marshal(assets)
		fmt.Println(string(data))
		return nil
	}

	if len(assets) == 0 {
		fmt.Fprintln(os.Stderr, "No characters or locations found. Has AI parsing completed?")
		return nil
	}

	// Print characters
	fmt.Fprintln(os.Stdout, "Characters:")
	for _, a := range assets {
		if a.Type != "character" {
			continue
		}
		imgIcon := statusIcon(a.ImageStatus)
		ttsIcon := statusIcon(a.TTSStatus)
		fmt.Fprintf(os.Stdout, "  %s %s  (uuid: %s)\n", imgIcon, a.Name, a.UUID)
		fmt.Fprintf(os.Stdout, "     image: %s", a.ImageStatus)
		if a.TTSStatus != "" {
			fmt.Fprintf(os.Stdout, "  |  voice: %s %s", ttsIcon, a.TTSStatus)
		}
		fmt.Fprintln(os.Stdout)
		if a.ImageURL != "" {
			fmt.Fprintf(os.Stdout, "     🖼  %s\n", a.ImageURL)
		}
	}

	// Print locations
	fmt.Fprintln(os.Stdout, "\nLocations:")
	for _, a := range assets {
		if a.Type != "location" {
			continue
		}
		imgIcon := statusIcon(a.ImageStatus)
		fmt.Fprintf(os.Stdout, "  %s %s  (uuid: %s)\n", imgIcon, a.Name, a.UUID)
		fmt.Fprintf(os.Stdout, "     image: %s\n", a.ImageStatus)
		if a.ImageURL != "" {
			fmt.Fprintf(os.Stdout, "     🖼  %s\n", a.ImageURL)
		}
	}

	// Hint for regeneration
	hasIdle := false
	for _, a := range assets {
		if a.ImageStatus == "idle" || a.ImageStatus == "" || a.ImageStatus == "failed" {
			hasIdle = true
			break
		}
	}
	if hasIdle {
		fmt.Fprintln(os.Stdout, "\nTo generate missing assets:")
		fmt.Fprintf(os.Stdout, "  latentcut generate character-image %s <character-uuid>\n", projectUUID)
		fmt.Fprintf(os.Stdout, "  latentcut generate location-image %s <location-uuid>\n", projectUUID)
	}

	return nil
}

func statusIcon(status string) string {
	switch status {
	case "done":
		return "✅"
	case "processing":
		return "🔄"
	case "pending":
		return "⏳"
	case "failed":
		return "❌"
	default:
		return "⬜"
	}
}
