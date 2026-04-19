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
