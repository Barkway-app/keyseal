package sopsutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/sopsutil"
)

func TestDecryptFileWithStubBinary(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	scriptPath := filepath.Join(binDir, "fake-sops")
	script := "#!/bin/sh\nif [ \"$1\" = \"--decrypt\" ]; then\n  cat \"$2\"\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	secretPath := filepath.Join(dir, "app.enc.yaml")
	body := "version: 1\nkind: env\nname: production/platform/app\nvalues:\n  APP_ENV: production\n"
	if err := os.WriteFile(secretPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	out, err := sopsutil.DecryptFile("fake-sops", secretPath)
	if err != nil {
		t.Fatalf("DecryptFile returned error: %v", err)
	}
	if !strings.Contains(string(out), "APP_ENV: production") {
		t.Fatalf("unexpected decrypt output: %q", string(out))
	}
}

func TestDecryptFileDoesNotMixWarningStderrIntoPlaintext(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	scriptPath := filepath.Join(binDir, "fake-sops")
	script := "#!/bin/sh\nif [ \"$1\" = \"--decrypt\" ]; then\n  echo 'warning: test' >&2\n  cat \"$2\"\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	secretPath := filepath.Join(dir, "app.enc.yaml")
	body := "version: 1\nkind: env\nname: production/platform/app\nvalues:\n  APP_ENV: production\n"
	if err := os.WriteFile(secretPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	out, err := sopsutil.DecryptFile("fake-sops", secretPath)
	if err != nil {
		t.Fatalf("DecryptFile returned error: %v", err)
	}
	if strings.Contains(string(out), "warning: test") {
		t.Fatalf("expected stderr to stay out of plaintext output, got %q", string(out))
	}
}

func TestEncryptFileWithStubBinary(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	scriptPath := filepath.Join(binDir, "fake-sops")
	script := "#!/bin/sh\nif [ \"$1\" = \"encrypt\" ] && [ \"$2\" = \"--filename-override\" ]; then\n  printf 'version: 1\\nkind: env\\nname: ENC[AES256_GCM,data:name,type:str]\\nvalues:\\n  APP_ENV: ENC[AES256_GCM,data:value,type:str]\\nsops:\\n  version: 3.9.0\\n'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := sopsutil.EncryptFile("fake-sops", []byte("version: 1\nkind: env\nname: production/platform/app\n"), "production/platform/app.enc.yaml")
	if err != nil {
		t.Fatalf("EncryptFile returned error: %v", err)
	}
	if !strings.Contains(string(out), "sops:") {
		t.Fatalf("expected encrypted output, got %q", string(out))
	}
	if strings.Contains(string(out), "production/platform/app") {
		t.Fatalf("expected encrypted output to avoid raw plaintext, got %q", string(out))
	}
}

func TestEncryptFileCleansUpTempPlaintext(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	trackerPath := filepath.Join(dir, "temp-path.txt")
	scriptPath := filepath.Join(binDir, "fake-sops")
	script := "#!/bin/sh\nif [ \"$1\" = \"encrypt\" ] && [ \"$2\" = \"--filename-override\" ]; then\n  printf '%s' \"$3\" > \"" + trackerPath + "\"\n  printf 'ENC[test]\\n'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake sops: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := sopsutil.EncryptFile("fake-sops", []byte("version: 1\n"), "production/platform/app.enc.yaml"); err != nil {
		t.Fatalf("EncryptFile returned error: %v", err)
	}

	tempPathBytes, err := os.ReadFile(trackerPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	tempPath := string(tempPathBytes)
	if tempPath == "" {
		t.Fatal("expected fake sops to record temp path")
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp plaintext file to be removed, stat err = %v", err)
	}
}
