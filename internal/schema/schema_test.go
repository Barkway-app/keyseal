package schema_test

import (
	"testing"

	"github.com/Barkway-app/keyseal/internal/schema"
)

func TestValidateDocument(t *testing.T) {
	doc := schema.EnvSecretDocument{
		Version: 1,
		Kind:    "env",
		Name:    "production/platform/app",
		Values: map[string]string{
			"APP_ENV": "production",
		},
	}
	if err := doc.Validate(schema.DefaultValidationOptions()); err != nil {
		t.Fatalf("expected valid doc, got %v", err)
	}
}

func TestValidateDocumentRejectsInvalidKey(t *testing.T) {
	doc := schema.EnvSecretDocument{
		Version: 1,
		Kind:    "env",
		Name:    "production/platform/app",
		Values: map[string]string{
			"app-env": "production",
		},
	}
	if err := doc.Validate(schema.DefaultValidationOptions()); err == nil {
		t.Fatal("expected invalid env key to fail validation")
	}
}

func TestParseYAMLDocumentFindsDuplicateValueKeys(t *testing.T) {
	body := []byte("version: 1\nkind: env\nname: staging/platform/api\nvalues:\n  APP_ENV: staging\n  APP_ENV: duplicate\n")
	_, dupes, err := schema.ParseYAMLDocument(body)
	if err != nil {
		t.Fatalf("ParseYAMLDocument returned error: %v", err)
	}
	if len(dupes) != 1 || dupes[0] != "APP_ENV" {
		t.Fatalf("expected duplicate APP_ENV, got %#v", dupes)
	}
}
