package buildinfo_test

import (
	"testing"

	"github.com/Barkway-app/keyseal/internal/buildinfo"
)

func TestOneLine(t *testing.T) {
	origVersion, origCommit, origDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	t.Cleanup(func() {
		buildinfo.Version = origVersion
		buildinfo.Commit = origCommit
		buildinfo.Date = origDate
	})

	buildinfo.Version = "v0.2.0"
	buildinfo.Commit = "abc1234"
	if got := buildinfo.OneLine(); got != "keyseal v0.2.0 (abc1234)" {
		t.Fatalf("OneLine() = %q", got)
	}
}

func TestShort(t *testing.T) {
	origVersion := buildinfo.Version
	t.Cleanup(func() {
		buildinfo.Version = origVersion
	})

	buildinfo.Version = "v0.2.0"
	if got := buildinfo.Short(); got != "v0.2.0" {
		t.Fatalf("Short() = %q", got)
	}
}

func TestFull(t *testing.T) {
	origVersion, origCommit, origDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	t.Cleanup(func() {
		buildinfo.Version = origVersion
		buildinfo.Commit = origCommit
		buildinfo.Date = origDate
	})

	buildinfo.Version = "v0.2.0"
	buildinfo.Commit = "abc123456789"
	buildinfo.Date = "2026-04-19T13:10:00Z"

	want := "keyseal v0.2.0 (abc1234)\ntag: v0.2.0\ncommit: abc1234\nbuilt: 2026-04-19T13:10:00Z"
	if got := buildinfo.Full(); got != want {
		t.Fatalf("Full() = %q", got)
	}
}
