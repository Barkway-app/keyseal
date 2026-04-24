package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateKeysRequiresTargetsOrAll verifies that accidental whole-repo
// updates require the explicit --all flag.
func TestUpdateKeysRequiresTargetsOrAll(t *testing.T) {
	output, err := runRootCommand(t, t.TempDir(), "updatekeys")
	if err == nil {
		t.Fatal("expected updatekeys without targets or --all to fail")
	}
	if !strings.Contains(err.Error(), "pass one or more logical names, or use --all") {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "" {
		t.Fatalf("expected no output, got %q", output)
	}
}

// TestUpdateKeysTargetsLogicalNames verifies that explicit logical names
// restrict the batch to only those encrypted files.
func TestUpdateKeysTargetsLogicalNames(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeEncryptedFixture(t, repoRoot, "production/platform/api.enc.yaml")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	appBody := readCLIFile(t, filepath.Join(repoRoot, "production/platform/app.enc.yaml"))
	apiBody := readCLIFile(t, filepath.Join(repoRoot, "production/platform/api.enc.yaml"))
	if !strings.Contains(appBody, "updated-by-fake-sops") {
		t.Fatalf("expected targeted file to be updated, got %q", appBody)
	}
	if strings.Contains(apiBody, "updated-by-fake-sops") {
		t.Fatalf("did not expect untargeted file to change, got %q", apiBody)
	}
	if !strings.Contains(output, "updatekeys summary: 1 updated, 0 unchanged, 0 skipped, 0 failed") {
		t.Fatalf("unexpected output: %q", output)
	}
}

// TestUpdateKeysLogicalNamesOverrideAll locks the rule that explicit logical
// names take precedence when --all is also present.
func TestUpdateKeysLogicalNamesOverrideAll(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeEncryptedFixture(t, repoRoot, "production/platform/api.enc.yaml")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--all", "--yes")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	appBody := readCLIFile(t, filepath.Join(repoRoot, "production/platform/app.enc.yaml"))
	apiBody := readCLIFile(t, filepath.Join(repoRoot, "production/platform/api.enc.yaml"))
	if !strings.Contains(appBody, "updated-by-fake-sops") {
		t.Fatalf("expected explicit target to be updated, got %q", appBody)
	}
	if strings.Contains(apiBody, "updated-by-fake-sops") {
		t.Fatalf("did not expect --all to expand when logical names are present, got %q", apiBody)
	}
	if !strings.Contains(output, "updatekeys summary: 1 updated, 0 unchanged, 0 skipped, 0 failed") {
		t.Fatalf("unexpected output: %q", output)
	}
}

// TestUpdateKeysAllDiscoversEncryptedFiles verifies that --all walks the repo
// root and processes each encrypted file it discovers.
func TestUpdateKeysAllDiscoversEncryptedFiles(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeEncryptedFixture(t, repoRoot, "staging/platform/app.enc.yaml")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "--all", "--yes")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "updated production/platform/app") || !strings.Contains(output, "updated staging/platform/app") {
		t.Fatalf("expected --all to process both files, got %q", output)
	}
	if !strings.Contains(output, "updatekeys summary: 2 updated, 0 unchanged, 0 skipped, 0 failed") {
		t.Fatalf("unexpected output: %q", output)
	}
}

// TestUpdateKeysSkipsPlaceholdersAndFailsPlaintextAndMissing covers the
// shared secret-file classification semantics used by other commands.
func TestUpdateKeysSkipsPlaceholdersAndFailsPlaintextAndMissing(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " \n")
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/plain.enc.yaml"), "version: 1\nkind: env\n")

	output, err := runRootCommand(
		t,
		repoRoot,
		"updatekeys",
		"production/platform/app",
		"production/platform/empty",
		"production/platform/plain",
		"production/platform/missing",
		"--yes",
	)
	if err == nil {
		t.Fatal("expected mixed failure batch to return error")
	}
	for _, snippet := range []string{
		"updated production/platform/app",
		"skipped production/platform/empty: placeholder",
		"failed production/platform/plain: non-empty plaintext at encrypted path",
		"failed production/platform/missing: missing",
		"updatekeys summary: 1 updated, 0 unchanged, 1 skipped, 2 failed",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected output to contain %q, got %q", snippet, output)
		}
	}
}

// TestUpdateKeysCountsUnchangedEncryptedFile verifies byte comparison after
// SOPS exits so unchanged files are not treated as updates.
func TestUpdateKeysCountsUnchangedEncryptedFile(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, false)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "unchanged production/platform/app") {
		t.Fatalf("expected unchanged output, got %q", output)
	}
	if !strings.Contains(output, "updatekeys summary: 0 updated, 1 unchanged, 0 skipped, 0 failed") {
		t.Fatalf("unexpected output: %q", output)
	}
}

// TestUpdateKeysPreflightFailures ensures invalid .sops.yaml states fail
// before any encrypted file is touched.
func TestUpdateKeysPreflightFailures(t *testing.T) {
	tests := []struct {
		name    string
		sopsYML *string
		want    string
	}{
		{name: "missing", sopsYML: nil, want: "read .sops.yaml"},
		{name: "invalid", sopsYML: strPtr("creation_rules: {}\n"), want: "creation_rules must be a sequence"},
		{name: "empty rules", sopsYML: strPtr("creation_rules: []\n"), want: "does not define any creation rules"},
		{name: "placeholder", sopsYML: strPtr("creation_rules:\n  - age: age1REPLACE_ME\n"), want: "placeholder recipient"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeTestConfig(t, repoRoot)
			if tt.sopsYML != nil {
				writeCLIFile(t, filepath.Join(repoRoot, ".sops.yaml"), *tt.sopsYML)
			}
			configureFakeUpdateKeysSOPS(t, repoRoot, true)
			writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")

			_, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes")
			if err == nil {
				t.Fatal("expected preflight failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error to contain %q, got %v", tt.want, err)
			}
			body := readCLIFile(t, filepath.Join(repoRoot, "production/platform/app.enc.yaml"))
			if strings.Contains(body, "updated-by-fake-sops") {
				t.Fatalf("expected preflight to avoid mutation, got %q", body)
			}
		})
	}
}

// TestUpdateKeysPreflightFailsWhenSOPSMissing verifies that the configured
// SOPS binary is resolved before the batch mutates files.
func TestUpdateKeysPreflightFailsWhenSOPSMissing(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	t.Setenv("PATH", gitOnlyPathDir(t))
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")

	_, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes")
	if err == nil {
		t.Fatal("expected missing sops to fail")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	body := readCLIFile(t, filepath.Join(repoRoot, "production/platform/app.enc.yaml"))
	if strings.Contains(body, "updated-by-fake-sops") {
		t.Fatalf("expected missing sops to avoid mutation, got %q", body)
	}
}

// TestUpdateKeysCommitOnlyWhenSuccessfulAndChanged verifies explicit commit
// behavior for a fully successful batch with file changes.
func TestUpdateKeysCommitOnlyWhenSuccessfulAndChanged(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "add", "--", "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "seed secret")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes", "-m", "Sync recipients")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "committed Git change: Sync recipients") {
		t.Fatalf("expected commit output, got %q", output)
	}
	if got := strings.TrimSpace(gitOutput(t, repoRoot, "log", "-1", "--pretty=%s")); got != "Sync recipients" {
		t.Fatalf("expected updatekeys commit, got %q", got)
	}
}

// TestUpdateKeysCommitSkippedWhenNothingChanged verifies that a successful but
// unchanged batch does not create an empty Git commit.
func TestUpdateKeysCommitSkippedWhenNothingChanged(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, false)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "add", "--", "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "seed secret")

	output, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes", "--commit")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "no files changed; no Git commit created") {
		t.Fatalf("expected no-change commit message, got %q", output)
	}
	if got := strings.TrimSpace(gitOutput(t, repoRoot, "log", "-1", "--pretty=%s")); got != "seed secret" {
		t.Fatalf("expected no new commit, got %q", got)
	}
}

// TestUpdateKeysPartialFailurePreventsCommit ensures changed files are left
// uncommitted when any target in the batch fails.
func TestUpdateKeysPartialFailurePreventsCommit(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	configureFakeUpdateKeysSOPS(t, repoRoot, true)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "add", "--", "production/platform/app.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "seed secret")

	_, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "production/platform/missing", "--yes", "--commit")
	if err == nil {
		t.Fatal("expected partial failure")
	}
	if got := strings.TrimSpace(gitOutput(t, repoRoot, "log", "-1", "--pretty=%s")); got != "seed secret" {
		t.Fatalf("expected partial failure to prevent commit, got %q", got)
	}
	status := gitOutput(t, repoRoot, "status", "--short")
	if !strings.Contains(status, " M production/platform/app.enc.yaml") {
		t.Fatalf("expected changed file to remain uncommitted, got %q", status)
	}
}

// TestUpdateKeysYesFlagReachesSOPS verifies that --yes maps to SOPS -y and the
// default interactive mode omits that flag.
func TestUpdateKeysYesFlagReachesSOPS(t *testing.T) {
	repoRoot := seedUpdateKeysRepo(t)
	argsPath := configureFakeUpdateKeysRecorderSOPS(t, repoRoot)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")

	_, err := runRootCommand(t, repoRoot, "updatekeys", "production/platform/app", "--yes")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	body := readCLIFile(t, argsPath)
	if !strings.Contains(body, "updatekeys -y ") {
		t.Fatalf("expected --yes to pass -y, got %q", body)
	}

	repoRoot = seedUpdateKeysRepo(t)
	argsPath = configureFakeUpdateKeysRecorderSOPS(t, repoRoot)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	_, err = runRootCommand(t, repoRoot, "updatekeys", "production/platform/app")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	body = readCLIFile(t, argsPath)
	if strings.Contains(body, " -y ") {
		t.Fatalf("did not expect default mode to pass -y, got %q", body)
	}
}

// seedUpdateKeysRepo creates a Git-backed Keyseal fixture with a ready
// .sops.yaml for updatekeys tests.
func seedUpdateKeysRepo(t *testing.T) string {
	t.Helper()
	repoRoot := seedKeysealRepo(t, false)
	writeValidSOPSConfig(t, repoRoot)
	return repoRoot
}

// writeValidSOPSConfig replaces the default empty test config with a creation
// rule that passes updatekeys preflight.
func writeValidSOPSConfig(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, filepath.Join(root, ".sops.yaml"), "creation_rules:\n  - path_regex: .*\\.enc\\.yaml$\n    age: age1realrecipient\n")
}

// configureFakeUpdateKeysSOPS installs a stub sops binary that optionally
// appends a marker to the target file.
func configureFakeUpdateKeysSOPS(t *testing.T, root string, mutate bool) {
	t.Helper()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nif [ \"$1\" = \"updatekeys\" ]; then\n  target=\"$2\"\n  if [ \"$2\" = \"-y\" ]; then target=\"$3\"; fi\n"
	if mutate {
		script += "  printf '# updated-by-fake-sops\\n' >> \"$target\"\n"
	}
	script += "  exit 0\nfi\nexit 1\n"
	writeFakeSOPS(t, filepath.Join(binDir, "sops"), script)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+gitBinDir(t))
}

// configureFakeUpdateKeysRecorderSOPS installs a stub sops binary that records
// its arguments so tests can assert flag translation.
func configureFakeUpdateKeysRecorderSOPS(t *testing.T, root string) string {
	t.Helper()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	argsPath := filepath.Join(root, "sops-args.txt")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nprintf '%s\\n' \"$*\" > \"" + argsPath + "\"\nif [ \"$1\" = \"updatekeys\" ]; then exit 0; fi\nexit 1\n"
	writeFakeSOPS(t, filepath.Join(binDir, "sops"), script)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+gitBinDir(t))
	return argsPath
}

// readCLIFile reads a fixture file and fails the test on any error.
func readCLIFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	return string(body)
}

// strPtr returns a stable string pointer for table-driven test fixtures.
func strPtr(value string) *string {
	return &value
}
