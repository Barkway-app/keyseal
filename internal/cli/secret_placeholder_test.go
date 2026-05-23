package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSkipsEmptyPlaceholderWhenOtherSecretsAreUsable(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestConfig(t, repoRoot)
	configureDecryptOnlySOPS(t, repoRoot)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " \n\t")

	output, err := runRootCommand(t, repoRoot, "render", "production/platform/app", "production/platform/empty", "--stdout")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	if !strings.Contains(output, "APP_ENV=\"production\"") {
		t.Fatalf("expected rendered dotenv output, got %q", output)
	}
	if strings.Contains(output, "EXAMPLE_KEY") {
		t.Fatalf("did not expect placeholder-only content in output, got %q", output)
	}
}

func TestRenderFailsWhenAllRequestedSecretsAreEmptyPlaceholders(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestConfig(t, repoRoot)
	configureVersionOnlySOPS(t, repoRoot)
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), "\n \t \n")

	_, err := runRootCommand(t, repoRoot, "render", "production/platform/empty", "--stdout")
	if err == nil {
		t.Fatal("expected render to fail when every requested secret is a placeholder")
	}
	if !strings.Contains(err.Error(), "all requested secrets are empty or uninitialized placeholder files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecSkipsEmptyPlaceholderWhenOtherSecretsAreUsable(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestConfig(t, repoRoot)
	configureDecryptOnlySOPS(t, repoRoot)
	writeEncryptedFixture(t, repoRoot, "production/platform/app.enc.yaml")
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " ")
	outPath := filepath.Join(repoRoot, "child-output.txt")

	_, err := runRootCommand(
		t,
		repoRoot,
		"exec",
		"production/platform/app",
		"production/platform/empty",
		"--",
		"/bin/sh",
		"-c",
		"printf %s \"$APP_ENV\" > \"$1\"",
		"sh",
		outPath,
	)
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(body) != "production" {
		t.Fatalf("expected child process to receive APP_ENV, got %q", string(body))
	}
}

func TestExecFailsWhenAllRequestedSecretsAreEmptyPlaceholders(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestConfig(t, repoRoot)
	configureVersionOnlySOPS(t, repoRoot)
	writeCLIFile(t, filepath.Join(repoRoot, "production/platform/empty.enc.yaml"), " \n")

	_, err := runRootCommand(t, repoRoot, "exec", "production/platform/empty", "--", "/bin/sh", "-c", "exit 0")
	if err == nil {
		t.Fatal("expected exec to fail when every requested secret is a placeholder")
	}
	if !strings.Contains(err.Error(), "all requested secrets are empty or uninitialized placeholder files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditBootstrapsEmptyPlaceholderBeforeOpeningSOPS(t *testing.T) {
	repoRoot := seedKeysealRepo(t, false)
	target := filepath.Join(repoRoot, "production/platform/app.enc.yaml")
	writeCLIFile(t, target, " \n")
	configureFakeEditBootstrapSOPS(t, repoRoot)

	_, err := runRootCommand(t, repoRoot, "edit", "production/platform/app")
	if err != nil {
		t.Fatalf("runRootCommand returned error: %v", err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "sops:") {
		t.Fatalf("expected encrypted starter to be written before edit, got %q", text)
	}
	if !strings.Contains(text, "edited") {
		t.Fatalf("expected fake editor to run after bootstrap, got %q", text)
	}
}

func configureDecryptOnlySOPS(t *testing.T, root string) {
	t.Helper()

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	writeFakeSOPS(t, filepath.Join(binDir, "sops"), "#!/bin/sh\nif [ \"$1\" = \"--decrypt\" ]; then\n  case \"$2\" in\n    *app.enc.yaml)\n      printf 'version: 1\\nkind: env\\nname: production/platform/app\\nvalues:\\n  APP_ENV: production\\n'\n      exit 0\n      ;;\n  esac\nfi\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nexit 1\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func configureVersionOnlySOPS(t *testing.T, root string) {
	t.Helper()

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	writeFakeSOPS(t, filepath.Join(binDir, "sops"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nexit 1\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func configureFakeEditBootstrapSOPS(t *testing.T, root string) {
	t.Helper()

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	writeFakeSOPS(t, filepath.Join(binDir, "sops"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf 'sops 3.9.0\\n'\n  exit 0\nfi\nif [ \"$1\" = \"encrypt\" ] && [ \"$2\" = \"--filename-override\" ]; then\n  printf 'version: 1\\nkind: env\\nname: ENC[test]\\nvalues:\\n  EXAMPLE_KEY: ENC[test]\\nsops:\\n  version: 3.9.0\\n'\n  exit 0\nfi\nprintf 'edited\\n' >> \"$1\"\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeEncryptedFixture(t *testing.T, root, rel string) {
	t.Helper()
	configureLibraryDecryptFixture(t, root)
	writeCLIFile(t, filepath.Join(root, rel), cliTestEncryptedYAML)
}

func writeCLIFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
