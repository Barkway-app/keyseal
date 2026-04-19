package templates_test

import (
	"testing"

	"github.com/Barkway-app/keyseal/internal/templates"
)

func TestBuildKnownTemplates(t *testing.T) {
	for _, name := range templates.SupportedNames() {
		t.Run(name, func(t *testing.T) {
			doc, err := templates.Build("production/platform/app", name)
			if err != nil {
				t.Fatalf("Build returned error: %v", err)
			}
			if doc.Kind != "env" {
				t.Fatalf("expected env kind, got %q", doc.Kind)
			}
			if len(doc.Values) == 0 {
				t.Fatal("expected template values")
			}
		})
	}
}
