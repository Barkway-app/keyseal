package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSOPSBackedCommandsFailEarlyWhenSOPSMissing verifies commands that need
// SOPS stop at a clear tool preflight instead of reaching workflow-specific
// decrypt or mutation errors.
func TestSOPSBackedCommandsFailEarlyWhenSOPSMissing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		args  []string
	}{
		{
			name: "add",
			args: []string{"add", "production/platform/app"},
		},
		{
			name: "edit",
			setup: func(t *testing.T, root string) {
				writeEncryptedFixture(t, root, "production/platform/app.enc.yaml")
			},
			args: []string{"edit", "production/platform/app"},
		},
		{
			name: "updatekeys",
			setup: func(t *testing.T, root string) {
				writeValidSOPSConfig(t, root)
				writeEncryptedFixture(t, root, "production/platform/app.enc.yaml")
			},
			args: []string{"updatekeys", "production/platform/app", "--yes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := seedKeysealRepo(t, false)
			if tt.setup != nil {
				tt.setup(t, root)
			}
			t.Setenv("PATH", gitOnlyPathDir(t))

			_, err := runRootCommand(t, root, tt.args...)
			if err == nil {
				t.Fatal("expected missing SOPS preflight to fail")
			}
			if !strings.Contains(err.Error(), "configured SOPS binary") || !strings.Contains(err.Error(), "binary not found") {
				t.Fatalf("expected clear SOPS preflight error, got %v", err)
			}
		})
	}
}

// TestReadOnlyCommandsDoNotRequireSOPSOrAgeBinaries verifies server/deploy
// hosts can decrypt with only Keyseal, encrypted files, and age key material.
func TestReadOnlyCommandsDoNotRequireSOPSOrAgeBinaries(t *testing.T) {
	t.Run("render", func(t *testing.T) {
		root := seedKeysealRepo(t, false)
		writeEncryptedFixture(t, root, "production/platform/app.enc.yaml")
		t.Setenv("PATH", gitOnlyPathDir(t))

		output, err := runRootCommand(t, root, "render", "production/platform/app", "--stdout")
		if err != nil {
			t.Fatalf("render should not require external sops or age binaries: %v", err)
		}
		if !strings.Contains(output, "APP_ENV=\"production\"") {
			t.Fatalf("expected decrypted dotenv output, got %q", output)
		}
	})

	t.Run("exec", func(t *testing.T) {
		root := seedKeysealRepo(t, false)
		writeEncryptedFixture(t, root, "production/platform/app.enc.yaml")
		t.Setenv("PATH", gitOnlyPathDir(t))
		outPath := filepath.Join(root, "child-output.txt")

		_, err := runRootCommand(t, root, "exec", "production/platform/app", "--", "/bin/sh", "-c", "printf %s \"$APP_ENV\" > \"$1\"", "sh", outPath)
		if err != nil {
			t.Fatalf("exec should not require external sops or age binaries: %v", err)
		}
		body, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		if string(body) != "production" {
			t.Fatalf("expected child process to receive APP_ENV, got %q", string(body))
		}
	})
}

// TestExplicitSOPSBinaryPathWorks verifies keyseal.yaml can point directly at
// a non-standard SOPS install location.
func TestExplicitSOPSBinaryPathWorks(t *testing.T) {
	root := seedKeysealRepo(t, false)
	binDir := filepath.Join(root, "custom-bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	sopsPath := filepath.Join(binDir, "custom-sops")
	writeFakeSOPS(t, sopsPath, "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nif [ \"$1\" = \"encrypt\" ] && [ \"$2\" = \"--filename-override\" ]; then\n  printf 'ENC[explicit-path]\\n'\n  exit 0\nfi\nexit 1\n")
	configPath := filepath.Join(root, "keyseal.yaml")
	body := readCLIFile(t, configPath)
	body = strings.Replace(body, "binary: sops", "binary: "+sopsPath, 1)
	writeCLIFile(t, configPath, body)
	t.Setenv("PATH", gitOnlyPathDir(t))

	output, err := runRootCommand(t, root, "add", "production/platform/app")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "created encrypted starter") {
		t.Fatalf("unexpected output: %q", output)
	}
	secret := readCLIFile(t, filepath.Join(root, "production/platform/app.enc.yaml"))
	if secret != "ENC[explicit-path]\n" {
		t.Fatalf("expected custom sops output, got %q", secret)
	}
}

// TestGitOnlyCommandDoesNotRequireSOPS verifies Git-only workflows stay usable
// even when the local SOPS binary is unavailable.
func TestGitOnlyCommandDoesNotRequireSOPS(t *testing.T) {
	root := seedKeysealRepo(t, false)
	writeEncryptedFixture(t, root, "production/platform/app.enc.yaml")
	t.Setenv("PATH", gitOnlyPathDir(t))

	output, err := runRootCommand(t, root, "status")
	if err != nil {
		t.Fatalf("status should not require SOPS: %v", err)
	}
	if !strings.Contains(output, "production/platform/app.enc.yaml") {
		t.Fatalf("expected status output for secret file, got %q", output)
	}
}
