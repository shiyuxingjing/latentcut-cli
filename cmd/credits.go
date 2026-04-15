package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewCreditsCmd returns the credits subcommand.
func NewCreditsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credits",
		Short: "Show current credit balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredits()
		},
	}
}

func runCredits() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())
	balance, err := client.GetCreditsBalance(context.Background())
	if err != nil {
		return fmt.Errorf("get credits: %w", err)
	}

	if jsonOut {
		data, _ := json.Marshal(balance)
		fmt.Println(string(data))
	} else {
		total := toInt(balance["credits"])
		daily := toInt(balance["credits_daily"])
		purchased := toInt(balance["credits_purchased"])
		fmt.Fprintf(os.Stderr, "Credits: %d (daily: %d, purchased: %d)\n", total, daily, purchased)
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(math.Round(n))
	case int:
		return n
	default:
		return 0
	}
}
