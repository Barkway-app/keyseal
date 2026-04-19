package execenv_test

import (
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/execenv"
)

func TestMergeEnvOverridesExistingValues(t *testing.T) {
	merged := execenv.MergeEnv(
		[]string{"APP_ENV=local", "HOME=/tmp/home"},
		map[string]string{"APP_ENV": "production", "DB_HOST": "db"},
	)
	got := strings.Join(merged, "\n")
	if !strings.Contains(got, "APP_ENV=production") {
		t.Fatalf("expected override to win, got %q", got)
	}
	if !strings.Contains(got, "HOME=/tmp/home") {
		t.Fatalf("expected inherited env to remain, got %q", got)
	}
	if !strings.Contains(got, "DB_HOST=db") {
		t.Fatalf("expected injected env key, got %q", got)
	}
}
