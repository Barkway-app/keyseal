package doctor

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestResultWarningAndVerifyHelpers verifies strict verification treats
// warnings as blockers even when there are no failures.
func TestResultWarningAndVerifyHelpers(t *testing.T) {
	result := Result{}
	result.Add(CheckResult{Name: "ok", Status: StatusOK, Summary: "ok"})
	result.Add(CheckResult{Name: "warn", Status: StatusWarn, Summary: "warn"})

	if !result.HasWarnings() {
		t.Fatal("expected HasWarnings to report warning")
	}
	if result.HasFailures() {
		t.Fatal("did not expect failure")
	}
	if result.VerifyPassed() {
		t.Fatal("expected warning to fail strict verification")
	}
}

// TestRenderVerifyTextFiltersOKChecks verifies failure output stays concise by
// omitting successful checks.
func TestRenderVerifyTextFiltersOKChecks(t *testing.T) {
	result := Result{}
	result.Add(CheckResult{Name: "ok", Status: StatusOK, Summary: "ok"})
	result.Add(CheckResult{Name: "warn", Status: StatusWarn, Summary: "warn", Details: []string{"detail"}, Remediation: []string{"fix it"}})
	result.Add(CheckResult{Name: "skip", Status: StatusSkip, Summary: "skip"})

	output := result.RenderVerifyText()
	if !strings.Contains(output, "verify failed: 0 failure(s), 1 warning(s), 1 skipped") {
		t.Fatalf("unexpected verify summary: %q", output)
	}
	if strings.Contains(output, "[OK]") {
		t.Fatalf("expected OK checks to be omitted, got %q", output)
	}
	if !strings.Contains(output, "[WARN] warn") || !strings.Contains(output, "[SKIP] skip") {
		t.Fatalf("expected warn and skip checks, got %q", output)
	}
	if !strings.Contains(output, "Fix: fix it") {
		t.Fatalf("expected remediation, got %q", output)
	}
}

// TestRenderVerifyJSONIncludesVerdict verifies CI consumers get an explicit
// strict verification verdict.
func TestRenderVerifyJSONIncludesVerdict(t *testing.T) {
	result := Result{}
	result.Add(CheckResult{Name: "ok", Status: StatusOK, Summary: "ok"})

	payload, err := result.RenderVerifyJSON()
	if err != nil {
		t.Fatalf("RenderVerifyJSON returned error: %v", err)
	}
	var decoded struct {
		Verified bool `json:"verified"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !decoded.Verified {
		t.Fatalf("expected verified=true, got %s", string(payload))
	}
}
