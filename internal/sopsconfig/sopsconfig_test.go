package sopsconfig

import (
	"strings"
	"testing"
)

// TestInspectFindsUsableRulesAndPlaceholders verifies that Inspect reports
// both recipient readiness and template placeholder values.
func TestInspectFindsUsableRulesAndPlaceholders(t *testing.T) {
	info, err := Inspect([]byte("creation_rules:\n  - path_regex: production/.*\\.enc\\.yaml$\n    age: age1REPLACE_ME,age1realrecipient\n"))
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.CreationRuleCount != 1 {
		t.Fatalf("expected 1 creation rule, got %#v", info)
	}
	if info.UsableRuleCount != 1 {
		t.Fatalf("expected usable recipients, got %#v", info)
	}
	if len(info.Placeholders) != 1 || info.Placeholders[0] != "age1REPLACE_ME" {
		t.Fatalf("expected placeholder recipient, got %#v", info.Placeholders)
	}
}

// TestInspectRejectsInvalidShape verifies that creation_rules must be a YAML
// sequence, matching the shape SOPS expects.
func TestInspectRejectsInvalidShape(t *testing.T) {
	_, err := Inspect([]byte("creation_rules: {}\n"))
	if err == nil {
		t.Fatal("expected invalid creation_rules shape to fail")
	}
	if !strings.Contains(err.Error(), "creation_rules must be a sequence") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInspectDetectsKeyGroupRecipients verifies that key_groups count as
// usable recipient material during preflight.
func TestInspectDetectsKeyGroupRecipients(t *testing.T) {
	info, err := Inspect([]byte("creation_rules:\n  - key_groups:\n      - age:\n          - age1realrecipient\n"))
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info.UsableRuleCount != 1 {
		t.Fatalf("expected key_groups recipient to be usable, got %#v", info)
	}
}
