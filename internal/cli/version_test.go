package cli

import (
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/buildinfo"
)

func TestRootVersionFlag(t *testing.T) {
	restoreBuildInfo(t, "v0.2.0", "abc1234", "2026-04-19T13:10:00Z")

	output, err := runRootCommand(t, t.TempDir(), "--version")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if output != "keyseal v0.2.0 (abc1234)\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestVersionCommand(t *testing.T) {
	restoreBuildInfo(t, "v0.2.0", "abc1234", "2026-04-19T13:10:00Z")

	output, err := runRootCommand(t, t.TempDir(), "version")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	want := "keyseal v0.2.0 (abc1234)\ntag: v0.2.0\ncommit: abc1234\nbuilt: 2026-04-19T13:10:00Z\n"
	if output != want {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestVersionCommandShort(t *testing.T) {
	restoreBuildInfo(t, "v0.2.0", "abc1234", "2026-04-19T13:10:00Z")

	output, err := runRootCommand(t, t.TempDir(), "version", "--short")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if strings.TrimSpace(output) != "v0.2.0" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func restoreBuildInfo(t *testing.T, version, commit, date string) {
	t.Helper()

	origVersion, origCommit, origDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	buildinfo.Version = version
	buildinfo.Commit = commit
	buildinfo.Date = date
	t.Cleanup(func() {
		buildinfo.Version = origVersion
		buildinfo.Commit = origCommit
		buildinfo.Date = origDate
	})
}
