package render_test

import (
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/render"
	"github.com/Barkway-app/keyseal/internal/schema"
)

func TestMergeEnvDocsLaterFilesWin(t *testing.T) {
	docs := []schema.EnvSecretDocument{
		{Values: map[string]string{"APP_ENV": "staging", "DB_HOST": "db-a"}},
		{Values: map[string]string{"APP_ENV": "production"}},
	}
	merged := render.MergeEnvDocs(docs)
	if merged["APP_ENV"] != "production" {
		t.Fatalf("expected later value to win, got %q", merged["APP_ENV"])
	}
	if merged["DB_HOST"] != "db-a" {
		t.Fatalf("expected DB_HOST to be preserved")
	}
}

func TestRenderDotenv(t *testing.T) {
	out, err := render.RenderDotenv(map[string]string{
		"APP_ENV": "production",
		"MULTI":   "hello\n\"quoted\"",
	})
	if err != nil {
		t.Fatalf("RenderDotenv returned error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "APP_ENV=\"production\"") {
		t.Fatalf("expected dotenv output to include APP_ENV, got %q", got)
	}
	if !strings.Contains(got, "MULTI=\"hello\\n\\\"quoted\\\"\"") {
		t.Fatalf("expected escaped multiline value, got %q", got)
	}
}

func TestRenderJSON(t *testing.T) {
	out, err := render.RenderJSON(map[string]string{"APP_ENV": "production"})
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}
	if !strings.Contains(string(out), "\"APP_ENV\": \"production\"") {
		t.Fatalf("unexpected json output: %q", string(out))
	}
}

func TestRenderYAML(t *testing.T) {
	out, err := render.RenderYAML(map[string]string{"DB_HOST": "db", "APP_ENV": "production"})
	if err != nil {
		t.Fatalf("RenderYAML returned error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "APP_ENV: production") || !strings.Contains(got, "DB_HOST: db") {
		t.Fatalf("unexpected yaml output: %q", string(out))
	}
	if strings.Index(got, "APP_ENV") > strings.Index(got, "DB_HOST") {
		t.Fatalf("expected yaml output to be sorted, got %q", got)
	}
}
