package doctor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/doctor"
)

func TestDoctorRunHappyPath(t *testing.T) {
	dir := t.TempDir()
	writeFakeSOPS(t, dir, "fake-sops", "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo 'sops 3.9.0'\n  exit 0\nfi\nif [ \"$1\" = \"--decrypt\" ]; then\n  printf 'version: 1\\nkind: env\\nname: production/platform/app\\nvalues:\\n  APP_ENV: production\\n'\n  exit 0\nfi\nexit 0\n")
	writeDoctorConfig(t, dir, "fake-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
	writeEncryptedSecret(t, dir, "production/platform/app.enc.yaml")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.HasFailures() {
		t.Fatalf("expected doctor to pass, got %#v", result.Checks)
	}

	check := findCheck(t, result, "sops binary")
	if check.Status != doctor.StatusOK {
		t.Fatalf("expected sops binary check to pass, got %#v", check)
	}
	if !containsSubstring(check.Details, "sops 3.9.0") {
		t.Fatalf("expected version detail, got %#v", check.Details)
	}
}

func TestDoctorFailsWhenKeysealConfigMissing(t *testing.T) {
	dir := t.TempDir()

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.HasFailures() {
		t.Fatal("expected missing config to fail")
	}

	check := findCheck(t, result, "keyseal.yaml")
	if check.Status != doctor.StatusFail {
		t.Fatalf("expected missing keyseal.yaml failure, got %#v", check)
	}
}

func TestDoctorFailsWhenKeysealConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	body := "version: 1\ndefaults:\n  file_mode: invalid\nvalidation:\n  key_pattern: \"[\"\n"
	if err := os.WriteFile(filepath.Join(dir, "keyseal.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, "keyseal.yaml")
	if check.Status != doctor.StatusFail {
		t.Fatalf("expected invalid keyseal.yaml failure, got %#v", check)
	}
}

func TestDoctorFailsWhenSOPSConfigMissing(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, ".sops.yaml")
	if check.Status != doctor.StatusFail {
		t.Fatalf("expected missing .sops.yaml failure, got %#v", check)
	}
}

func TestDoctorDetectsPlaceholderRecipients(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1REPLACE_ME,age1RECOVERY_REPLACE_ME\n")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, ".sops.yaml placeholders")
	if check.Status != doctor.StatusFail {
		t.Fatalf("expected placeholder failure, got %#v", check)
	}
	if !containsSubstring(check.Details, "age1REPLACE_ME") {
		t.Fatalf("expected placeholder detail, got %#v", check.Details)
	}
}

func TestDoctorFlagsPlaintextStarterFiles(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
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
	check := findCheck(t, result, "secret production/platform/app")
	if check.Status != doctor.StatusFail {
		t.Fatalf("expected plaintext failure, got %#v", check)
	}
	if !strings.Contains(check.Summary, "non-empty plaintext") {
		t.Fatalf("unexpected summary: %q", check.Summary)
	}
}

func TestDoctorWarnsOnEmptyPlaceholderSecretFiles(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
	secretPath := filepath.Join(dir, "production/platform/app.enc.yaml")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(secretPath, nil, 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, "secret production/platform/app")
	if check.Status != doctor.StatusWarn {
		t.Fatalf("expected empty placeholder warning, got %#v", check)
	}
	if !strings.Contains(check.Summary, "empty or uninitialized placeholder") {
		t.Fatalf("unexpected summary: %q", check.Summary)
	}
}

func TestDoctorWarnsOnWhitespaceOnlyPlaceholderSecretFiles(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
	secretPath := filepath.Join(dir, "production/platform/app.enc.yaml")
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(secretPath, []byte(" \n\t"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, "secret production/platform/app")
	if check.Status != doctor.StatusWarn {
		t.Fatalf("expected whitespace placeholder warning, got %#v", check)
	}
	if !strings.Contains(check.Summary, "empty or uninitialized placeholder") {
		t.Fatalf("unexpected summary: %q", check.Summary)
	}
}

func TestDoctorSkipsDecryptValidationWhenSOPSMissing(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
	writeEncryptedSecret(t, dir, "production/platform/app.enc.yaml")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	binaryCheck := findCheck(t, result, "sops binary")
	if binaryCheck.Status != doctor.StatusFail {
		t.Fatalf("expected missing sops failure, got %#v", binaryCheck)
	}
	skipCheck := findCheck(t, result, "decrypt validation")
	if skipCheck.Status != doctor.StatusSkip {
		t.Fatalf("expected decrypt validation skip, got %#v", skipCheck)
	}
}

func TestDoctorWarnsOnUnsafeFileMode(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir, "missing-sops", "0644")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, "defaults.file_mode")
	if check.Status != doctor.StatusWarn {
		t.Fatalf("expected unsafe file mode warning, got %#v", check)
	}
}

func TestDoctorWarnsOnDuplicateEnvKeys(t *testing.T) {
	dir := t.TempDir()
	writeFakeSOPS(t, dir, "fake-sops", "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo 'sops 3.9.0'\n  exit 0\nfi\nif [ \"$1\" = \"--decrypt\" ]; then\n  printf 'version: 1\\nkind: env\\nname: production/platform/app\\nvalues:\\n  APP_ENV: production\\n  APP_ENV: duplicate\\n'\n  exit 0\nfi\nexit 0\n")
	writeDoctorConfig(t, dir, "fake-sops", "0600")
	writeSOPSConfig(t, dir, "creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1realrecipient\n")
	writeEncryptedSecret(t, dir, "production/platform/app.enc.yaml")

	result, err := doctor.Run(dir)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	check := findCheck(t, result, "secret production/platform/app")
	if check.Status != doctor.StatusWarn {
		t.Fatalf("expected duplicate key warning, got %#v", check)
	}
	if !containsSubstring(check.Details, "APP_ENV") {
		t.Fatalf("expected duplicate key detail, got %#v", check.Details)
	}
}

func findCheck(t *testing.T, result doctor.Result, name string) doctor.CheckResult {
	t.Helper()
	for _, check := range result.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %q not found in %#v", name, result.Checks)
	return doctor.CheckResult{}
}

func containsSubstring(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func writeDoctorConfig(t *testing.T, dir, binary, mode string) {
	t.Helper()
	cfg := `version: 1
repository:
  root: .
  encrypted_extension: .enc.yaml
sops:
  binary: ` + binary + `
defaults:
  output_format: dotenv
  output_dir: /run/secrets
  file_mode: "` + mode + `"
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
}

func writeSOPSConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}
}

func writeEncryptedSecret(t *testing.T, dir, relativePath string) {
	t.Helper()
	secretPath := filepath.Join(dir, relativePath)
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	secret := "version: 1\nkind: env\nname: ENC[AES256_GCM,data:abc,type:str]\nvalues:\n  APP_ENV: ENC[AES256_GCM,data:def,type:str]\nsops:\n  version: 3.8.0\n"
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
}

func writeFakeSOPS(t *testing.T, dir, binaryName, script string) {
	t.Helper()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	path := filepath.Join(binDir, binaryName)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
