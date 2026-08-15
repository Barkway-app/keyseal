package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrpbuilds/keyseal/internal/config"
)

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keyseal.yaml")
	body := []byte("version: 1\nrepository:\n  root: .\n")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Repository.EncryptedExtension != ".enc.yaml" {
		t.Fatalf("expected default encrypted extension, got %q", cfg.Repository.EncryptedExtension)
	}
	if cfg.Defaults.OutputFormat != "dotenv" {
		t.Fatalf("expected default output format, got %q", cfg.Defaults.OutputFormat)
	}
	if cfg.Git.AutoCommit {
		t.Fatal("expected git.auto_commit to default to false")
	}
	if cfg.SOPS.AgeBinary != "age" {
		t.Fatalf("expected default age binary, got %q", cfg.SOPS.AgeBinary)
	}
	if cfg.Validation.KeyPattern != "^[A-Z0-9_]+$" {
		t.Fatalf("expected default key pattern, got %q", cfg.Validation.KeyPattern)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keyseal.yaml")
	body := []byte("version: 1\ndefaults:\n  file_mode: not-a-mode\nvalidation:\n  key_pattern: \"[\"\n")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := config.Load(path); err == nil {
		t.Fatal("expected invalid config to fail")
	}
}
