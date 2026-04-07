package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("expected non-empty config path")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(path))
	}
}

func TestLoadDefaults(t *testing.T) {
	// Point to a non-existent file to test defaults
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ServerURL != "http://localhost:4111" {
		t.Errorf("expected default server URL, got %s", cfg.ServerURL)
	}
	if cfg.OutputDir != "novelo-output" {
		t.Errorf("expected default output dir, got %s", cfg.OutputDir)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{
		APIKey:    "testkey123",
		ServerURL: "http://example.com:4111",
		OutputDir: "/tmp/output",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file was created
	path := DefaultConfigPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if loaded.APIKey != cfg.APIKey {
		t.Errorf("APIKey mismatch: got %s, want %s", loaded.APIKey, cfg.APIKey)
	}
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL mismatch: got %s, want %s", loaded.ServerURL, cfg.ServerURL)
	}
	if loaded.OutputDir != cfg.OutputDir {
		t.Errorf("OutputDir mismatch: got %s, want %s", loaded.OutputDir, cfg.OutputDir)
	}
}
