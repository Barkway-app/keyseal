package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrpbuilds/keyseal/internal/fsutil"
)

func TestIsSafeFileMode(t *testing.T) {
	mode, err := fsutil.ParseFileMode("0600")
	if err != nil {
		t.Fatalf("ParseFileMode returned error: %v", err)
	}
	if !fsutil.IsSafeFileMode(mode) {
		t.Fatal("expected 0600 to be safe")
	}

	mode, err = fsutil.ParseFileMode("0644")
	if err != nil {
		t.Fatalf("ParseFileMode returned error: %v", err)
	}
	if fsutil.IsSafeFileMode(mode) {
		t.Fatal("expected 0644 to be unsafe")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rendered.env")
	if err := fsutil.AtomicWriteFile(target, []byte("APP_ENV=production\n"), 0o600, false); err != nil {
		t.Fatalf("AtomicWriteFile returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "APP_ENV=production\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestValidateOutputPathRejectsTraversal(t *testing.T) {
	if err := fsutil.ValidateOutputPath("../secrets.env"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestAtomicWriteFileRejectsExistingFileWithoutForce(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rendered.env")
	if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := fsutil.AtomicWriteFile(target, []byte("new"), 0o600, false); err == nil {
		t.Fatal("expected existing file to be rejected without force")
	}
}
