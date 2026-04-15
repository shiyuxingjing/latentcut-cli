package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewRechargeCmd returns the recharge subcommand.
func NewRechargeCmd() *cobra.Command {
	var code string

	cmd := &cobra.Command{
		Use:   "recharge",
		Short: "Redeem a credit code to add credits",
		Example: `  latentcut recharge --code ABC123
  latentcut recharge -c ABC123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecharge(code)
		},
	}

	cmd.Flags().StringVarP(&code, "code", "c", "", "Redeem code (required)")
	_ = cmd.MarkFlagRequired("code")

	return cmd
}

func runRecharge(code string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.EffectiveToken() == "" {
		return fmt.Errorf("not logged in. Run: latentcut login")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, cfg.EffectiveToken())

	fmt.Fprintf(os.Stderr, "Redeeming code: %s...\n", code)
	err = client.RedeemCode(context.Background(), code)
	if err != nil {
		return fmt.Errorf("redeem failed: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Redeemed successfully!")

	// Show updated balance
	balance, err := client.GetCreditsBalance(context.Background())
	if err == nil {
		total := toInt(balance["credits"])
		fmt.Fprintf(os.Stderr, "Current credits: %d\n", total)
	}

	return nil
}
