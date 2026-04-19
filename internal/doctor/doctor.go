// Package doctor performs repository health checks without exposing secret data.
package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Barkway-app/keyseal/internal/config"
	"github.com/Barkway-app/keyseal/internal/fsutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/schema"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
)

// Result collects doctor notes, warnings, and hard failures.
type Result struct {
	Errors   []string
	Warnings []string
	Notes    []string
}

func (r *Result) addError(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Result) addWarning(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Result) addNote(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// HasErrors reports whether the doctor run found any hard failures.
func (r Result) HasErrors() bool {
	return len(r.Errors) > 0
}

// Run executes the v1 RC doctor checks for the current repository.
func Run(cwd string) (Result, error) {
	var result Result

	cfgPath := filepath.Join(cwd, config.DefaultConfigPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		result.addError("config: %v", err)
		return result, nil
	}
	result.addNote("loaded %s", cfgPath)

	sopsPath := filepath.Join(cwd, ".sops.yaml")
	if _, err := os.Stat(sopsPath); err != nil {
		result.addError(".sops.yaml: %v", err)
	} else {
		result.addNote("found %s", sopsPath)
	}

	if _, err := sopsutil.LookPath(cfg.SOPS.Binary); err != nil {
		result.addError("sops: %v", err)
	}
	sopsAvailable := !containsPrefix(result.Errors, "sops:")

	mode, err := fsutil.ParseFileMode(cfg.Defaults.FileMode)
	if err != nil {
		result.addError("defaults.file_mode: %v", err)
	} else if !fsutil.IsSafeFileMode(mode) {
		result.addWarning("defaults.file_mode %s is broader than owner-only access", cfg.Defaults.FileMode)
	}
	if err := fsutil.ValidateOutputPath(cfg.Defaults.OutputDir); err != nil {
		result.addWarning("defaults.output_dir: %v", err)
	}

	repoRoot := cfg.RepoRoot(cwd)
	files, err := repo.DiscoverEncryptedFiles(repoRoot, cfg.Repository.EncryptedExtension)
	if err != nil {
		return result, fmt.Errorf("discover encrypted files: %w", err)
	}
	if len(files) == 0 {
		result.addNote("no encrypted files discovered")
	}

	opts := schema.DefaultValidationOptions()
	opts.RequireValues = cfg.Validation.RequireValues
	opts.KeyPattern = cfg.Validation.KeyPattern

	for _, file := range files {
		logical, err := repo.PathToLogicalName(repoRoot, file, cfg.Repository.EncryptedExtension)
		if err != nil {
			result.addError("%s: %v", file, err)
			continue
		}
		roundTrip, err := repo.LogicalNameToPath(repoRoot, logical, cfg.Repository.EncryptedExtension)
		if err != nil || roundTrip != file {
			result.addError("%s: logical path resolution mismatch", file)
			continue
		}
		rawFile, readErr := os.ReadFile(file)
		if readErr == nil {
			doc, _, parseErr := schema.ParseYAMLDocument(rawFile)
			if parseErr == nil && !schema.HasSOPSMetadata(rawFile) {
				if validateErr := doc.Validate(opts); validateErr == nil {
					result.addError("%s: appears to be a plaintext starter document; run `keyseal edit %s` to encrypt it with SOPS", file, logical)
					continue
				}
			}
		}
		if !sopsAvailable {
			result.addNote("skipped decrypt validation for %s because the configured sops binary is unavailable", file)
			continue
		}
		plaintext, err := sopsutil.DecryptFile(cfg.SOPS.Binary, file)
		if err != nil {
			result.addError("%s: decrypt failed (%v)", file, classifyDecryptError(err))
			continue
		}
		doc, dupes, err := schema.ParseYAMLDocument(plaintext)
		if err != nil {
			result.addError("%s: invalid yaml (%v)", file, err)
			continue
		}
		if err := doc.Validate(opts); err != nil {
			result.addError("%s: schema validation failed (%v)", file, err)
			continue
		}
		if len(dupes) > 0 {
			result.addWarning("%s: duplicate env keys in values: %v", file, dupes)
		}
	}

	return result, nil
}

func classifyDecryptError(err error) string {
	if errors.Is(err, sopsutil.ErrBinaryNotFound) {
		return "sops binary not found"
	}
	return "sops command failed"
}

func containsPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
