package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all persistent CLI configuration.
type Config struct {
	APIKey           string `yaml:"api_key"`
	ServerURL        string `yaml:"server_url"`
	OutputDir        string `yaml:"output_dir"`
	LatentCutURL     string `yaml:"latentcut_url"`
	Token            string `yaml:"token"`
	APIKeyLatentCut  string `yaml:"api_key_latentcut"`
	Account          string `yaml:"account"`
	LastThreadID     string `yaml:"last_thread_id,omitempty"`
}

// EffectiveToken returns the API key to use for latentCut-server requests.
// Prefers APIKeyLatentCut when set, falls back to Token for backward compatibility.
func (c *Config) EffectiveToken() string {
	if c.APIKeyLatentCut != "" {
		return c.APIKeyLatentCut
	}
	return c.Token
}

// DefaultConfigPath returns the path to the config file (~/.novelo/config.yaml).
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".novelo/config.yaml"
	}
	return filepath.Join(home, ".novelo", "config.yaml")
}

// Load reads config from ~/.novelo/config.yaml and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		ServerURL:    "http://localhost:4111",
		OutputDir:    "novelo-output",
		LatentCutURL: "http://shiyuxingjing.com",
	}

	path := DefaultConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for unset fields
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:4111"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "novelo-output"
	}
	if cfg.LatentCutURL == "" {
		cfg.LatentCutURL = "http://localhost:7001"
	}

	return cfg, nil
}

// Save writes the config to ~/.novelo/config.yaml.
func (c *Config) Save() error {
	path := DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
