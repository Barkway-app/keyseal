package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jrpbuilds/keyseal/internal/config"
)

// TestLoadYAMLPinSemantics pins the lax yaml.Unmarshal contract of the config
// loader after the Go 1.27 / yaml.v3 dependency refresh.
//
// Strictness is deliberately frozen, not changed: the loader must keep ignoring
// unknown keys (yaml.Unmarshal behaviour) and must not switch to rejecting them
// (yaml.UnmarshalStrict behaviour). Any future yaml.v3 bump that alters either
// this unknown-key laxness or the duplicate-key semantics fails here.
func TestLoadYAMLPinSemantics(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantErr         bool
		wantErrContains string
		wantVersion     int
		wantRepoRoot    string
		wantSOPSBinary  string
	}{
		{
			name: "unknown keys are ignored",
			body: "version: 1\n" +
				"repository:\n" +
				"  root: .\n" +
				"unknown_top_level_key: ignored\n" +
				"sops:\n" +
				"  binary: sops\n" +
				"  not_a_real_option: ignored\n",
			wantErr:        false,
			wantVersion:    1,
			wantRepoRoot:   ".",
			wantSOPSBinary: "sops",
		},
		{
			name: "duplicate mapping key is an error",
			body: "version: 1\n" +
				"repository:\n" +
				"  root: .\n" +
				"repository:\n" +
				"  root: other\n",
			wantErr:         true,
			wantErrContains: "already defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "keyseal.yaml")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			cfg, err := config.Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected Load to fail, got cfg %+v", cfg)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg.Version != tt.wantVersion {
				t.Fatalf("expected version %d, got %d", tt.wantVersion, cfg.Version)
			}
			if cfg.Repository.Root != tt.wantRepoRoot {
				t.Fatalf("expected repository.root %q, got %q", tt.wantRepoRoot, cfg.Repository.Root)
			}
			if cfg.SOPS.Binary != tt.wantSOPSBinary {
				t.Fatalf("expected sops.binary %q, got %q", tt.wantSOPSBinary, cfg.SOPS.Binary)
			}
		})
	}
}
