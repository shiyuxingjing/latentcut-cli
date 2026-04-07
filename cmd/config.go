package cmd

import (
	"fmt"
	"strings"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns the config subcommand with set/get/list subcommands.
func NewConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage novelo-cli configuration",
	}

	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigListCmd())

	return configCmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Example: `  novelo-cli config set api-key mykey123
  novelo-cli config set server-url http://localhost:4111
  novelo-cli config set output-dir ./output`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			switch strings.ToLower(key) {
			case "api-key", "api_key":
				cfg.APIKey = value
			case "server-url", "server_url":
				cfg.ServerURL = value
			case "output-dir", "output_dir":
				cfg.OutputDir = value
			default:
				return fmt.Errorf("unknown config key: %s (valid: api-key, server-url, output-dir)", key)
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Set %s\n", key)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			switch strings.ToLower(key) {
			case "api-key", "api_key":
				if cfg.APIKey == "" {
					fmt.Println("(not set)")
				} else {
					fmt.Println(maskKey(cfg.APIKey))
				}
			case "server-url", "server_url":
				fmt.Println(cfg.ServerURL)
			case "output-dir", "output_dir":
				fmt.Println(cfg.OutputDir)
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			fmt.Printf("api-key:    %s\n", maskKey(cfg.APIKey))
			fmt.Printf("server-url: %s\n", cfg.ServerURL)
			fmt.Printf("output-dir: %s\n", cfg.OutputDir)
			fmt.Printf("\nConfig path: %s\n", config.DefaultConfigPath())
			return nil
		},
	}
}

// maskKey masks all but the last 4 characters of an API key.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}
