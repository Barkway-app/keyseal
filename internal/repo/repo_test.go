package repo_test

import (
	"path/filepath"
	"testing"

	"github.com/Barkway-app/keyseal/internal/repo"
)

func TestLogicalNameToPath(t *testing.T) {
	got, err := repo.LogicalNameToPath("/repo", "production/platform/app", ".enc.yaml")
	if err != nil {
		t.Fatalf("LogicalNameToPath returned error: %v", err)
	}
	want := filepath.Join("/repo", "production/platform/app.enc.yaml")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestValidateLogicalNameRejectsSketchyPaths(t *testing.T) {
	cases := []string{
		"../prod/app",
		"/prod/app",
		"production//app",
		"production/app.enc.yaml",
		"production/../app",
		"./production/app",
		"production\\app",
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if err := repo.ValidateLogicalName(tc); err == nil {
				t.Fatalf("expected %q to be rejected", tc)
			}
		})
	}
}

func TestPathToLogicalName(t *testing.T) {
	got, err := repo.PathToLogicalName("/repo", filepath.Join("/repo", "staging/platform/api.enc.yaml"), ".enc.yaml")
	if err != nil {
		t.Fatalf("PathToLogicalName returned error: %v", err)
	}
	if got != "staging/platform/api" {
		t.Fatalf("expected logical name staging/platform/api, got %q", got)
	}
}
