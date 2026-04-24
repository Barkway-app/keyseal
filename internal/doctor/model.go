// Package doctor performs repository health checks without exposing secret data.
package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Status describes the outcome of a single doctor check.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Severity describes how urgent a doctor finding is.
type Severity string

const (
	SeverityInformational Severity = "informational"
	SeverityWarning       Severity = "warning"
	SeverityError         Severity = "error"
)

// CheckResult is one structured doctor finding.
type CheckResult struct {
	Name        string   `json:"name"`
	Status      Status   `json:"status"`
	Severity    Severity `json:"severity"`
	Summary     string   `json:"summary"`
	Details     []string `json:"details,omitempty"`
	Remediation []string `json:"remediation,omitempty"`
}

// Counts summarizes the number of checks in each status bucket.
type Counts struct {
	OK    int `json:"ok"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
	Skip  int `json:"skip"`
	Total int `json:"total"`
}

// Result contains every doctor check and the computed summary counts.
type Result struct {
	Checks []CheckResult `json:"checks"`
}

// Add appends a check result.
func (r *Result) Add(check CheckResult) {
	r.Checks = append(r.Checks, check)
}

// HasFailures reports whether any doctor check failed.
func (r Result) HasFailures() bool {
	return r.Counts().Fail > 0
}

// HasWarnings reports whether any doctor check produced a warning.
func (r Result) HasWarnings() bool {
	return r.Counts().Warn > 0
}

// VerifyPassed reports whether the result is clean enough for strict CI
// verification, where both failures and warnings are release blockers.
func (r Result) VerifyPassed() bool {
	counts := r.Counts()
	return counts.Fail == 0 && counts.Warn == 0
}

// Counts computes the current per-status totals.
func (r Result) Counts() Counts {
	var counts Counts
	for _, check := range r.Checks {
		switch check.Status {
		case StatusOK:
			counts.OK++
		case StatusWarn:
			counts.Warn++
		case StatusFail:
			counts.Fail++
		case StatusSkip:
			counts.Skip++
		}
	}
	counts.Total = len(r.Checks)
	return counts
}

// RenderText formats the doctor report for humans.
func (r Result) RenderText() string {
	counts := r.Counts()
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "doctor summary: %d ok, %d warning(s), %d failure(s), %d skipped\n", counts.OK, counts.Warn, counts.Fail, counts.Skip)
	for _, check := range r.Checks {
		fmt.Fprintf(buf, "[%s] %s: %s\n", displayStatus(check.Status), check.Name, check.Summary)
		for _, detail := range check.Details {
			fmt.Fprintf(buf, "  - %s\n", detail)
		}
		for _, remediation := range check.Remediation {
			fmt.Fprintf(buf, "  Fix: %s\n", remediation)
		}
	}
	return buf.String()
}

// RenderVerifyText formats a concise strict-verification report for CI logs.
func (r Result) RenderVerifyText() string {
	counts := r.Counts()
	buf := &bytes.Buffer{}
	if r.VerifyPassed() {
		fmt.Fprintf(buf, "verify passed: %d check(s) ok", counts.OK)
		if counts.Skip > 0 {
			fmt.Fprintf(buf, ", %d skipped", counts.Skip)
		}
		buf.WriteByte('\n')
		return buf.String()
	}

	fmt.Fprintf(buf, "verify failed: %d failure(s), %d warning(s), %d skipped\n", counts.Fail, counts.Warn, counts.Skip)
	for _, check := range r.Checks {
		if check.Status == StatusOK {
			continue
		}
		fmt.Fprintf(buf, "[%s] %s: %s\n", displayStatus(check.Status), check.Name, check.Summary)
		for _, detail := range check.Details {
			fmt.Fprintf(buf, "  - %s\n", detail)
		}
		for _, remediation := range check.Remediation {
			fmt.Fprintf(buf, "  Fix: %s\n", remediation)
		}
	}
	return buf.String()
}

// RenderJSON formats the doctor report for machine-readable consumers.
func (r Result) RenderJSON() ([]byte, error) {
	payload := struct {
		Summary Counts        `json:"summary"`
		Checks  []CheckResult `json:"checks"`
	}{
		Summary: r.Counts(),
		Checks:  r.Checks,
	}
	return json.MarshalIndent(payload, "", "  ")
}

// RenderVerifyJSON formats strict verification results for machine-readable CI
// consumers while preserving the doctor summary and checks shape.
func (r Result) RenderVerifyJSON() ([]byte, error) {
	payload := struct {
		Verified bool          `json:"verified"`
		Summary  Counts        `json:"summary"`
		Checks   []CheckResult `json:"checks"`
	}{
		Verified: r.VerifyPassed(),
		Summary:  r.Counts(),
		Checks:   r.Checks,
	}
	return json.MarshalIndent(payload, "", "  ")
}

func displayStatus(status Status) string {
	switch status {
	case StatusOK:
		return "OK"
	case StatusWarn:
		return "WARN"
	case StatusFail:
		return "FAIL"
	case StatusSkip:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}
