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

type galleryQuery struct {
	characterUUID string
	locationUUID  string
	shotUUID      string
	resourceUUID  string
	openResult    bool
}

// NewGalleryCmd returns the gallery subcommand.
func NewGalleryCmd() *cobra.Command {
	query := &galleryQuery{}
	cmd := &cobra.Command{
		Use:   "gallery <project-uuid>",
		Short: "Browse project assets (characters, locations, shots)",
		Long:  "Displays a structured view of all characters, locations, and shot videos with their status, UUIDs, resource UUIDs, and URLs.",
		Example: `  latentcut gallery project-xxx
  latentcut gallery project-xxx --character char-xxx
  latentcut gallery project-xxx --resource res-xxx --open
  latentcut gallery project-xxx --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGallery(args[0], *query)
		},
	}
	cmd.Flags().StringVar(&query.characterUUID, "character", "", "Show a single character asset by character UUID")
	cmd.Flags().StringVar(&query.locationUUID, "location", "", "Show a single location asset by location UUID")
	cmd.Flags().StringVar(&query.shotUUID, "shot", "", "Show a single shot asset by shot UUID")
	cmd.Flags().StringVar(&query.resourceUUID, "resource", "", "Show a single latest resource by resource UUID")
	cmd.Flags().BoolVar(&query.openResult, "open", false, "Open the selected resource URL in the default browser")
	return cmd
}

func runGallery(projectUUID string, query galleryQuery) error {
	if err := query.validate(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	gallery, err := client.GetGallery(context.Background(), projectUUID)
	if err != nil {
		return fmt.Errorf("get gallery: %w", err)
	}

	selected, err := query.selectItem(gallery)
	if err != nil {
		if query.resourceUUID != "" {
			resource, detailErr := client.GetResourceDetail(context.Background(), projectUUID, query.resourceUUID)
			if detailErr != nil {
				return err
			}
			selected = &gallerySelection{
				Kind:         "resource",
				Label:        resource.Label,
				TargetUUID:   resource.TargetUUID,
				ResourceUUID: resource.ResourceUUID,
				Status:       resource.Status,
				URL:          resource.FileURL,
				Detail:       resource,
			}
		} else {
			return err
		}
	}

	if jsonOut {
		payload := any(gallery)
		if selected != nil {
			payload = selected
		}
		data, _ := json.Marshal(payload)
		fmt.Println(string(data))
		return nil
	}

	if selected != nil {
		printGallerySelection(*selected)
		if query.openResult {
			if selected.URL == "" {
				return fmt.Errorf("selected item has no preview URL")
			}
			return runOpen(selected.URL)
		}
		return nil
	}

	// Characters
	if len(gallery.Characters) > 0 {
		fmt.Fprintln(os.Stdout, "Characters:")
		for _, c := range gallery.Characters {
			icon := galleryStatusIcon(c.ImageStatus)
			voiceIcon := galleryStatusIcon(c.VoiceStatus)
			fmt.Fprintf(os.Stdout, "  %s %s  (character: %s)\n", icon, c.Name, c.UUID)
			fmt.Fprintf(os.Stdout, "     image: %s", c.ImageStatus)
			if c.ImageResourceUUID != "" {
				fmt.Fprintf(os.Stdout, "  [res: %s]", c.ImageResourceUUID)
			}
			fmt.Fprintln(os.Stdout)
			if c.ImageURL != "" {
				fmt.Fprintf(os.Stdout, "     %s\n", c.ImageURL)
			}
			fmt.Fprintf(os.Stdout, "     voice: %s %s", voiceIcon, c.VoiceStatus)
			if c.VoiceResourceUUID != "" {
				fmt.Fprintf(os.Stdout, "  [res: %s]", c.VoiceResourceUUID)
			}
			fmt.Fprintln(os.Stdout)
			if c.VoiceURL != "" {
				fmt.Fprintf(os.Stdout, "     %s\n", c.VoiceURL)
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	// Locations
	if len(gallery.Locations) > 0 {
		fmt.Fprintln(os.Stdout, "Locations:")
		for _, l := range gallery.Locations {
			icon := galleryStatusIcon(l.ImageStatus)
			fmt.Fprintf(os.Stdout, "  %s %s  (location: %s)\n", icon, l.Name, l.UUID)
			fmt.Fprintf(os.Stdout, "     image: %s", l.ImageStatus)
			if l.ImageResourceUUID != "" {
				fmt.Fprintf(os.Stdout, "  [res: %s]", l.ImageResourceUUID)
			}
			fmt.Fprintln(os.Stdout)
			if l.ImageURL != "" {
				fmt.Fprintf(os.Stdout, "     %s\n", l.ImageURL)
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	// Shots grouped by episode
	if len(gallery.Shots) > 0 {
		currentEp := -1
		for _, s := range gallery.Shots {
			if s.EpisodeNumber != currentEp {
				currentEp = s.EpisodeNumber
				fmt.Fprintf(os.Stdout, "Shots (Episode %d: %s):\n", s.EpisodeNumber, s.EpisodeTitle)
			}
			icon := galleryStatusIcon(s.VideoStatus)
			fmt.Fprintf(os.Stdout, "  %s Shot %d  (shot: %s)\n", icon, s.ShotNumber, s.UUID)
			fmt.Fprintf(os.Stdout, "     video: %s", s.VideoStatus)
			if s.VideoResourceUUID != "" {
				fmt.Fprintf(os.Stdout, "  [res: %s]", s.VideoResourceUUID)
			}
			fmt.Fprintln(os.Stdout)
			if s.VideoURL != "" {
				fmt.Fprintf(os.Stdout, "     %s\n", s.VideoURL)
			}
		}
	}

	if query.openResult {
		return fmt.Errorf("--open requires one of --character / --location / --shot / --resource")
	}

	return nil
}

type gallerySelection struct {
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	TargetUUID   string `json:"targetUuid,omitempty"`
	ResourceUUID string `json:"resourceUuid,omitempty"`
	Status       string `json:"status,omitempty"`
	URL          string `json:"url,omitempty"`
	Detail       any    `json:"detail"`
}

func (q galleryQuery) validate() error {
	count := 0
	if q.characterUUID != "" {
		count++
	}
	if q.locationUUID != "" {
		count++
	}
	if q.shotUUID != "" {
		count++
	}
	if q.resourceUUID != "" {
		count++
	}
	if count > 1 {
		return fmt.Errorf("choose only one of --character / --location / --shot / --resource")
	}
	return nil
}

func (q galleryQuery) selectItem(gallery *latentcut.Gallery) (*gallerySelection, error) {
	switch {
	case q.characterUUID != "":
		for _, item := range gallery.Characters {
			if item.UUID == q.characterUUID {
				url := item.ImageURL
				if url == "" {
					url = item.VoiceURL
				}
				resourceUUID := item.ImageResourceUUID
				if resourceUUID == "" {
					resourceUUID = item.VoiceResourceUUID
				}
				return &gallerySelection{
					Kind:         "character",
					Label:        item.Name,
					TargetUUID:   item.UUID,
					ResourceUUID: resourceUUID,
					Status:       item.ImageStatus,
					URL:          url,
					Detail:       item,
				}, nil
			}
		}
		return nil, fmt.Errorf("character not found: %s", q.characterUUID)
	case q.locationUUID != "":
		for _, item := range gallery.Locations {
			if item.UUID == q.locationUUID {
				return &gallerySelection{
					Kind:         "location",
					Label:        item.Name,
					TargetUUID:   item.UUID,
					ResourceUUID: item.ImageResourceUUID,
					Status:       item.ImageStatus,
					URL:          item.ImageURL,
					Detail:       item,
				}, nil
			}
		}
		return nil, fmt.Errorf("location not found: %s", q.locationUUID)
	case q.shotUUID != "":
		for _, item := range gallery.Shots {
			if item.UUID == q.shotUUID {
				return &gallerySelection{
					Kind:         "shot",
					Label:        fmt.Sprintf("Episode %d Shot %d", item.EpisodeNumber, item.ShotNumber),
					TargetUUID:   item.UUID,
					ResourceUUID: item.VideoResourceUUID,
					Status:       item.VideoStatus,
					URL:          item.VideoURL,
					Detail:       item,
				}, nil
			}
		}
		return nil, fmt.Errorf("shot not found: %s", q.shotUUID)
	case q.resourceUUID != "":
		for _, item := range gallery.Resources {
			if item.ResourceUUID == q.resourceUUID {
				return &gallerySelection{
					Kind:         "resource",
					Label:        item.Label,
					TargetUUID:   item.TargetUUID,
					ResourceUUID: item.ResourceUUID,
					Status:       item.Status,
					URL:          item.FileURL,
					Detail:       item,
				}, nil
			}
		}
		return nil, fmt.Errorf("resource not found: %s", q.resourceUUID)
	default:
		return nil, nil
	}
}

func printGallerySelection(sel gallerySelection) {
	fmt.Fprintf(os.Stdout, "%s: %s\n", galleryKindLabel(sel.Kind), sel.Label)
	if sel.TargetUUID != "" {
		fmt.Fprintf(os.Stdout, "  target: %s\n", sel.TargetUUID)
	}
	if sel.ResourceUUID != "" {
		fmt.Fprintf(os.Stdout, "  resource: %s\n", sel.ResourceUUID)
	}
	if sel.Status != "" {
		fmt.Fprintf(os.Stdout, "  status: %s\n", sel.Status)
	}
	if sel.URL != "" {
		fmt.Fprintf(os.Stdout, "  url: %s\n", sel.URL)
	}
}

func galleryKindLabel(kind string) string {
	switch kind {
	case "character":
		return "Character"
	case "location":
		return "Location"
	case "shot":
		return "Shot"
	case "resource":
		return "Resource"
	default:
		return "Item"
	}
}

func galleryStatusIcon(status string) string {
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
