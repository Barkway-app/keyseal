package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifyPassesOnCleanRepository verifies strict verification succeeds when
// doctor reports no warnings or failures.
func TestVerifyPassesOnCleanRepository(t *testing.T) {
	repoRoot := seedVerifyRepo(t)
	configureVersionOnlySOPS(t, repoRoot)
	configureFakeAge(t, repoRoot)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")

	output, err := runRootCommand(t, repoRoot, "verify")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "verify passed:") {
		t.Fatalf("expected verify pass output, got %q", output)
	}
	if strings.Contains(output, "[OK]") {
		t.Fatalf("expected concise output without OK checks, got %q", output)
	}
}

// TestVerifyFailsOnWarning verifies warnings are strict CI failures even when
// doctor would still exit successfully.
func TestVerifyFailsOnWarning(t *testing.T) {
	repoRoot := seedVerifyRepo(t)
	configureVersionOnlySOPS(t, repoRoot)
	configureFakeAge(t, repoRoot)
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " \n")

	doctorOutput, doctorErr := runRootCommand(t, repoRoot, "doctor")
	if doctorErr != nil {
		t.Fatalf("doctor should not fail on warning-only repo: %v", doctorErr)
	}
	if !strings.Contains(doctorOutput, "[WARN]") {
		t.Fatalf("expected doctor warning output, got %q", doctorOutput)
	}

	verifyOutput, verifyErr := runRootCommand(t, repoRoot, "verify")
	if verifyErr == nil {
		t.Fatal("expected verify to fail on warning")
	}
	if !strings.Contains(verifyOutput, "verify failed: 0 failure(s), 1 warning(s)") {
		t.Fatalf("expected strict warning failure, got %q", verifyOutput)
	}
	if strings.Contains(verifyOutput, "[OK]") {
		t.Fatalf("expected verify output to omit OK checks, got %q", verifyOutput)
	}
}

// TestVerifyFailsOnFailure verifies normal doctor failures also fail strict
// verification.
func TestVerifyFailsOnFailure(t *testing.T) {
	repoRoot := t.TempDir()

	output, err := runRootCommand(t, repoRoot, "verify")
	if err == nil {
		t.Fatal("expected verify to fail when keyseal.yaml is missing")
	}
	if !strings.Contains(output, "verify failed:") || !strings.Contains(output, "[FAIL] keyseal.yaml") {
		t.Fatalf("unexpected verify failure output: %q", output)
	}
}

// TestVerifyJSONOutput verifies the machine-readable verdict and strict exit
// behavior.
func TestVerifyJSONOutput(t *testing.T) {
	repoRoot := seedVerifyRepo(t)
	configureVersionOnlySOPS(t, repoRoot)
	configureFakeAge(t, repoRoot)
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " \n")

	output, err := runRootCommand(t, repoRoot, "verify", "--json")
	if err == nil {
		t.Fatal("expected verify --json to fail on warning")
	}

	var payload struct {
		Verified bool `json:"verified"`
		Summary  struct {
			Warn int `json:"warn"`
		} `json:"summary"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("Unmarshal returned error: %v\noutput=%s", unmarshalErr, output)
	}
	if payload.Verified {
		t.Fatalf("expected verified=false, got %#v", payload)
	}
	if payload.Summary.Warn == 0 {
		t.Fatalf("expected warning count, got %#v", payload)
	}
	if len(payload.Checks) == 0 {
		t.Fatalf("expected checks in verify json, got %#v", payload)
	}
}

// seedVerifyRepo creates a Keyseal repo with real .sops.yaml recipients for
// verify tests.
func seedVerifyRepo(t *testing.T) string {
	t.Helper()
	repoRoot := seedKeysealRepo(t, false)
	writeValidSOPSConfig(t, repoRoot)
	return repoRoot
}

// configureFakeAge installs a version-capable age stub so clean verify tests do
// not inherit host tool availability.
func configureFakeAge(t *testing.T, root string) {
	t.Helper()
	writeFakeSOPS(t, filepath.Join(root, "bin", "age"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'age 1.2.0\\n'\n  exit 0\nfi\nexit 0\n")
}
