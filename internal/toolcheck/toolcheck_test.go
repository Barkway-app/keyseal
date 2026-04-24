package toolcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProbeFindsBinaryFromPath verifies the common configuration case where a
// bare command name is resolved through PATH.
func TestProbeFindsBinaryFromPath(t *testing.T) {
	binDir := t.TempDir()
	writeTool(t, filepath.Join(binDir, "demo-tool"), "#!/bin/sh\nprintf 'demo 1.2.3\\n'\n")
	t.Setenv("PATH", binDir)

	probe, err := Probe("demo-tool", nil, "--version")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if probe.Configured != "demo-tool" {
		t.Fatalf("expected configured name to be preserved, got %#v", probe)
	}
	if probe.Resolved != filepath.Join(binDir, "demo-tool") {
		t.Fatalf("expected PATH-resolved binary, got %#v", probe)
	}
	if probe.Version != "demo 1.2.3" {
		t.Fatalf("expected version line, got %#v", probe)
	}
}

// TestProbeFindsExplicitBinaryPath verifies non-standard installs configured
// with an absolute path.
func TestProbeFindsExplicitBinaryPath(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "custom-tool")
	writeTool(t, binPath, "#!/bin/sh\nprintf 'custom 9.0.0\\n'\n")
	t.Setenv("PATH", "")

	probe, err := Probe(binPath, nil, "--version")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if probe.Resolved != binPath {
		t.Fatalf("expected explicit path to resolve, got %#v", probe)
	}
	if probe.Version != "custom 9.0.0" {
		t.Fatalf("expected explicit tool version, got %#v", probe)
	}
}

// TestProbeReportsMissingBinary verifies callers can surface a precise missing
// binary error before running workflow-specific commands.
func TestProbeReportsMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := Probe("missing-tool", nil, "--version")
	if err == nil {
		t.Fatal("expected missing binary to fail")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// writeTool creates an executable test binary script.
func writeTool(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write test tool: %v", err)
	}
}
