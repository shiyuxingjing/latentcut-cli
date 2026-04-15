package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
)

// NewLoginCmd returns the login subcommand.
func NewLoginCmd() *cobra.Command {
	var apiKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Configure API key for latentCut-server",
		Long:  "Set your API key for authenticating with latentCut-server. Get your API key from https://shiyuxingjing.com",
		Example: `  latentcut login --api-key nv-abc123...
  latentcut login  (interactive prompt)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(apiKey)
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (starts with nv-)")

	return cmd
}

func runLogin(apiKey string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Interactive prompt if no API key provided
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Enter your API key (get one from https://shiyuxingjing.com):")
		if cfg.APIKeyLatentCut != "" {
			masked := maskAPIKey(cfg.APIKeyLatentCut)
			fmt.Fprintf(os.Stderr, "API Key [%s]: ", masked)
		} else {
			fmt.Fprint(os.Stderr, "API Key: ")
		}
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(line)
		if apiKey == "" {
			if cfg.APIKeyLatentCut != "" {
				fmt.Fprintln(os.Stderr, "Keeping existing API key.")
				return nil
			}
			return fmt.Errorf("API key is required")
		}
	}

	// Validate prefix
	if !strings.HasPrefix(apiKey, "nv-") {
		fmt.Fprintln(os.Stderr, "Warning: API key does not start with 'nv-' prefix. Proceeding anyway.")
	}

	// Save API key
	cfg.APIKeyLatentCut = apiKey
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Verify the API key by fetching credit balance
	fmt.Fprintf(os.Stderr, "Verifying API key with %s...\n", cfg.LatentCutURL)
	client := latentcut.NewClient(cfg.LatentCutURL, apiKey)
	_, err = client.GetCreditsBalance(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: API key saved but verification failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "The key has been saved. If the server is not reachable, it may still work later.")
	} else {
		fmt.Fprintln(os.Stderr, "API key verified and saved successfully!")
	}

	fmt.Fprintf(os.Stderr, "Config: %s\n", config.DefaultConfigPath())
	return nil
}

// maskAPIKey masks an API key for display (show first 6 + last 4 chars).
func maskAPIKey(key string) string {
	if len(key) > 10 {
		return key[:6] + "..." + key[len(key)-4:]
	}
	return "****"
}
