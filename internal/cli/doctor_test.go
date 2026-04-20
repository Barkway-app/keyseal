package cli

import (
	"encoding/json"
	"testing"
)

func TestDoctorJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()

	output, err := runRootCommand(t, repoRoot, "doctor", "--json")
	if err == nil {
		t.Fatal("expected doctor to exit non-zero when keyseal.yaml is missing")
	}

	var payload struct {
		Summary struct {
			Fail int `json:"fail"`
		} `json:"summary"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", unmarshalErr, output)
	}
	if payload.Summary.Fail == 0 {
		t.Fatalf("expected failure count in json output, got %#v", payload)
	}
	if len(payload.Checks) == 0 {
		t.Fatalf("expected checks in json output, got %#v", payload)
	}
}
