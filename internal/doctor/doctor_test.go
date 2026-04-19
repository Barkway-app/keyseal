package doctor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/doctor"
)

func TestDoctorRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	fakeSOPS := filepath.Join(dir, "bin", "fake-sops")
	script := "#!/bin/sh\nif [ \"$1\" = \"--decrypt\" ]; then\n  printf 'version: 1\\nkind: env\\nname: production/platform/app\\nvalues:\\n  APP_ENV: production\\n'\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeSOPS, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := `version: 1
repository:
  root: .
  encrypted_extension: .enc.yaml
sops:
  binary: fake-sops
defaults:
  output_format: dotenv
  output_dir: /run/secrets
  file_mode: "0600"
validation:
  require_values: true
  key_pattern: "^[A-Z0-9_]+$"
profiles:
  default:
    renders: []
`
	if err := os.WriteFile(filepath.Join(dir, "keyseal.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte("creation_rules: []\n"), 0o600); err != nil {
		t.Fatalf("write sops config: %v", err)
	}
	secretPath := filepath.Join(dir, "production/platform/app.enc.yaml")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	secret := "version: 1\nkind: env\nname: ENC[AES256_GCM,data:abc,type:str]\nvalues:\n  APP_ENV: ENC[AES256_GCM,data:def,type:str]\nsops:\n  version: 3.8.0\n"
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("expected doctor to pass, got errors: %#v", result.Errors)
	}
}

func TestDoctorFlagsPlaintextStarterFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := `version: 1
repository:
  root: .
  encrypted_extension: .enc.yaml
sops:
  binary: missing-sops
defaults:
  output_format: dotenv
  output_dir: /run/secrets
  file_mode: "0600"
validation:
  require_values: true
  key_pattern: "^[A-Z0-9_]+$"
profiles:
  default:
    renders: []
`
	if err := os.WriteFile(filepath.Join(dir, "keyseal.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte("creation_rules: []\n"), 0o600); err != nil {
		t.Fatalf("write sops config: %v", err)
	}
	secretPath := filepath.Join(dir, "production/platform/app.enc.yaml")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	secret := "version: 1\nkind: env\nname: production/platform/app\nvalues:\n  APP_ENV: production\n"
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.HasErrors() {
		t.Fatal("expected plaintext starter document to be flagged")
	}
	found := false
	for _, msg := range result.Errors {
		if strings.Contains(msg, "plaintext starter document") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected plaintext starter error, got %#v", result.Errors)
	}
}
