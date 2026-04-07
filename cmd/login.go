package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/novelo-ai/novelo-cli/internal/config"
	"github.com/novelo-ai/novelo-cli/internal/latentcut"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewLoginCmd returns the login subcommand.
func NewLoginCmd() *cobra.Command {
	var (
		account  string
		password string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to latentCut-server and save JWT token",
		Example: `  novelo-cli login --account user@example.com --password mypass
  novelo-cli login  (interactive prompt)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(account, password)
		},
	}

	cmd.Flags().StringVar(&account, "account", "", "Account (email or username)")
	cmd.Flags().StringVar(&password, "password", "", "Password")

	return cmd
}

func runLogin(account, password string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Interactive prompts for missing fields
	reader := bufio.NewReader(os.Stdin)
	if account == "" {
		if cfg.Account != "" {
			fmt.Fprintf(os.Stderr, "Account [%s]: ", cfg.Account)
		} else {
			fmt.Fprint(os.Stderr, "Account: ")
		}
		line, _ := reader.ReadString('\n')
		account = strings.TrimSpace(line)
		if account == "" {
			account = cfg.Account
		}
	}
	if account == "" {
		return fmt.Errorf("account is required")
	}

	if password == "" {
		fmt.Fprint(os.Stderr, "Password: ")
		pwBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		password = string(pwBytes)
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	client := latentcut.NewClient(cfg.LatentCutURL, "")
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Logging in to %s...\n", cfg.LatentCutURL)

	loginData, err := client.Login(ctx, account, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Use the JWT to create a persistent API key
	fmt.Fprintf(os.Stderr, "Creating API key...\n")
	apiKeyData, err := client.CreateAPIKey(ctx, loginData.Token, "novelo-cli")
	if err != nil {
		return fmt.Errorf("create API key failed: %w", err)
	}

	// Save API key as the primary auth credential; discard JWT
	cfg.APIKeyLatentCut = apiKeyData.APIKey
	cfg.Token = ""
	cfg.Account = account
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Show masked key: first 6 chars + **** + last 4 chars
	key := apiKeyData.APIKey
	masked := key
	if len(key) > 10 {
		masked = key[:6] + "..." + key[len(key)-4:]
	}
	fmt.Fprintf(os.Stderr, "Logged in and API key created: %s\n", masked)
	return nil
}
