package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all persistent CLI configuration.
type Config struct {
	APIKey    string `yaml:"api_key"`
	ServerURL string `yaml:"server_url"`
	OutputDir string `yaml:"output_dir"`
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
		ServerURL: "http://localhost:4111",
		OutputDir: "novelo-output",
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
