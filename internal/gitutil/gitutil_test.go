package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Barkway-app/keyseal/internal/gitutil"
)

func TestRepoRootDetectsRepository(t *testing.T) {
	repoRoot := newGitRepo(t)

	got, err := gitutil.RepoRoot("git", repoRoot)
	if err != nil {
		t.Fatalf("RepoRoot returned error: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, got)
	}
}

func TestRepoRootRejectsDirectoryOutsideRepository(t *testing.T) {
	dir := t.TempDir()

	_, err := gitutil.RepoRoot("git", dir)
	if err == nil {
		t.Fatal("expected RepoRoot to fail outside a Git repository")
	}
	if !strings.Contains(err.Error(), gitutil.ErrNotRepository.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLookPathReportsMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := gitutil.LookPath("git")
	if err == nil {
		t.Fatal("expected LookPath to fail when git is missing")
	}
	if !strings.Contains(err.Error(), gitutil.ErrBinaryNotFound.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusFiltersTrackedAndUntrackedChanges(t *testing.T) {
	repoRoot := newGitRepo(t)
	writeFile(t, filepath.Join(repoRoot, "tracked.enc.yaml"), "v1\n")
	writeFile(t, filepath.Join(repoRoot, "plain.txt"), "v1\n")
	gitRun(t, repoRoot, "add", "--", "tracked.enc.yaml", "plain.txt")
	gitRun(t, repoRoot, "commit", "-m", "seed")

	writeFile(t, filepath.Join(repoRoot, "tracked.enc.yaml"), "v2\n")
	writeFile(t, filepath.Join(repoRoot, "untracked.enc.yaml"), "new\n")

	entries, err := gitutil.Status("git", repoRoot)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 status entries, got %#v", entries)
	}
}

func TestStagePathsStagesOnlyExplicitFiles(t *testing.T) {
	repoRoot := newGitRepo(t)
	writeFile(t, filepath.Join(repoRoot, "secret.enc.yaml"), "old\n")
	writeFile(t, filepath.Join(repoRoot, "notes.txt"), "old\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml", "notes.txt")
	gitRun(t, repoRoot, "commit", "-m", "seed")

	writeFile(t, filepath.Join(repoRoot, "secret.enc.yaml"), "new\n")
	writeFile(t, filepath.Join(repoRoot, "notes.txt"), "new\n")

	if err := gitutil.StagePaths("git", repoRoot, []string{filepath.Join(repoRoot, "secret.enc.yaml")}); err != nil {
		t.Fatalf("StagePaths returned error: %v", err)
	}

	entries, err := gitutil.Status("git", repoRoot)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	var secret, notes gitutil.StatusEntry
	for _, entry := range entries {
		switch entry.Path {
		case "secret.enc.yaml":
			secret = entry
		case "notes.txt":
			notes = entry
		}
	}
	if secret.Staged != 'M' || secret.Unstaged != ' ' {
		t.Fatalf("expected secret.enc.yaml to be staged only, got %#v", secret)
	}
	if notes.Staged != ' ' || notes.Unstaged != 'M' {
		t.Fatalf("expected notes.txt to remain unstaged, got %#v", notes)
	}
}

func TestCommitReturnsNothingToCommit(t *testing.T) {
	repoRoot := newGitRepo(t)

	err := gitutil.Commit("git", repoRoot, "no-op")
	if err == nil {
		t.Fatal("expected Commit to fail without staged changes")
	}
	if err != gitutil.ErrNothingToCommit {
		t.Fatalf("expected ErrNothingToCommit, got %v", err)
	}
}

func TestHistoryAndDiffOperateOnSingleFile(t *testing.T) {
	repoRoot := newGitRepo(t)
	target := filepath.Join(repoRoot, "secret.enc.yaml")
	writeFile(t, target, "first\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "first")
	writeFile(t, target, "second\n")

	diffText, err := gitutil.Diff("git", repoRoot, target)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	if !strings.Contains(diffText, "secret.enc.yaml") {
		t.Fatalf("expected diff output to mention the file, got %q", diffText)
	}

	historyText, err := gitutil.History("git", repoRoot, target, false)
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if !strings.Contains(historyText, "first") {
		t.Fatalf("expected history output to include the commit message, got %q", historyText)
	}

	onelineHistory, err := gitutil.History("git", repoRoot, target, true)
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if !strings.Contains(onelineHistory, "first") || strings.Contains(onelineHistory, "Author:") {
		t.Fatalf("expected one-line history output, got %q", onelineHistory)
	}
}

func TestHasLocalChangesDetectsDirtyFile(t *testing.T) {
	repoRoot := newGitRepo(t)
	target := filepath.Join(repoRoot, "secret.enc.yaml")
	writeFile(t, target, "first\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "first")
	writeFile(t, target, "second\n")

	dirty, err := gitutil.HasLocalChanges("git", repoRoot, target)
	if err != nil {
		t.Fatalf("HasLocalChanges returned error: %v", err)
	}
	if !dirty {
		t.Fatal("expected file to be reported as dirty")
	}
}

func TestRestoreFileRestoresHistoricalContent(t *testing.T) {
	repoRoot := newGitRepo(t)
	target := filepath.Join(repoRoot, "secret.enc.yaml")
	writeFile(t, target, "first\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "first")
	firstCommit := strings.TrimSpace(gitOutput(t, repoRoot, "rev-parse", "HEAD"))

	writeFile(t, target, "second\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "second")

	if err := gitutil.RestoreFile("git", repoRoot, firstCommit, target); err != nil {
		t.Fatalf("RestoreFile returned error: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(body) != "first\n" {
		t.Fatalf("expected restored content %q, got %q", "first\n", string(body))
	}
}

func TestRevisionContainsPathChecksHistoricalPresence(t *testing.T) {
	repoRoot := newGitRepo(t)
	writeFile(t, filepath.Join(repoRoot, "secret.enc.yaml"), "first\n")
	gitRun(t, repoRoot, "add", "--", "secret.enc.yaml")
	gitRun(t, repoRoot, "commit", "-m", "first")
	commit := strings.TrimSpace(gitOutput(t, repoRoot, "rev-parse", "HEAD"))

	ok, err := gitutil.RevisionContainsPath("git", repoRoot, commit, filepath.Join(repoRoot, "secret.enc.yaml"))
	if err != nil {
		t.Fatalf("RevisionContainsPath returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected file to exist in historical revision")
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	repoRoot := t.TempDir()
	gitRun(t, repoRoot, "init")
	gitRun(t, repoRoot, "config", "user.name", "Keyseal Test")
	gitRun(t, repoRoot, "config", "user.email", "keyseal@example.com")
	return repoRoot
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return string(output)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
