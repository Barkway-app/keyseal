package doctor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Barkway-app/keyseal/internal/config"
	"github.com/Barkway-app/keyseal/internal/fsutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/schema"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
)

// Run executes the doctor checks for the current repository.
func Run(cwd string) (Result, error) {
	var result Result

	cfg, cfgOK := runConfigChecks(cwd, &result)
	runSOPSConfigChecks(cwd, &result)

	if !cfgOK {
		result.Add(CheckResult{
			Name:     "secret validation",
			Status:   StatusSkip,
			Severity: SeverityInformational,
			Summary:  "Skipped repository-backed checks because keyseal.yaml could not be loaded",
			Remediation: []string{
				"Create or fix keyseal.yaml, then rerun `keyseal doctor`.",
			},
		})
		return result, nil
	}

	repoRoot := cfg.RepoRoot(cwd)
	repoRootReady := runRepositoryRootCheck(repoRoot, &result)
	runOutputPathChecks(cfg, &result)
	runRepoArtifactChecks(repoRoot, &result)

	sopsAvailable := runSOPSBinaryChecks(cfg, cwd, &result)
	if !repoRootReady {
		result.Add(CheckResult{
			Name:     "secret discovery",
			Status:   StatusSkip,
			Severity: SeverityInformational,
			Summary:  "Skipped secret discovery because repository.root does not resolve to a usable directory",
			Remediation: []string{
				"Fix repository.root in keyseal.yaml so it points at the repository working tree.",
			},
		})
		return result, nil
	}

	if err := runSecretChecks(cwd, repoRoot, cfg, sopsAvailable, &result); err != nil {
		return result, fmt.Errorf("run secret checks: %w", err)
	}
	return result, nil
}

func runConfigChecks(cwd string, result *Result) (config.Config, bool) {
	cfgPath := filepath.Join(cwd, config.DefaultConfigPath)
	if _, err := os.Stat(cfgPath); err != nil {
		result.Add(CheckResult{
			Name:     "keyseal.yaml",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "Missing keyseal.yaml",
			Details: []string{
				fmt.Sprintf("Expected configuration file at %s.", cfgPath),
				"Keyseal commands rely on this file for repository layout, SOPS binary settings, and validation rules.",
			},
			Remediation: []string{
				"Run `keyseal init` in the repository root to create a starter config.",
			},
		})
		return config.Config{}, false
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		result.Add(CheckResult{
			Name:     "keyseal.yaml",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "keyseal.yaml could not be parsed or validated",
			Details: []string{
				fmt.Sprintf("Load error: %v", err),
			},
			Remediation: []string{
				"Fix the YAML syntax or invalid values in keyseal.yaml, then rerun `keyseal doctor`.",
			},
		})
		return config.Config{}, false
	}

	result.Add(CheckResult{
		Name:     "keyseal.yaml",
		Status:   StatusOK,
		Severity: SeverityInformational,
		Summary:  "Parsed and validated keyseal.yaml",
		Details: []string{
			fmt.Sprintf("repository.root resolves from %q.", cfg.Repository.Root),
			fmt.Sprintf("Encrypted extension is %q.", cfg.Repository.EncryptedExtension),
			fmt.Sprintf("Configured SOPS binary is %q.", cfg.SOPS.Binary),
		},
	})
	return cfg, true
}

func runRepositoryRootCheck(repoRoot string, result *Result) bool {
	info, err := os.Stat(repoRoot)
	if err != nil {
		result.Add(CheckResult{
			Name:     "repository.root",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "repository.root does not resolve to an existing directory",
			Details: []string{
				fmt.Sprintf("Resolved repository root: %s", repoRoot),
				fmt.Sprintf("Filesystem error: %v", err),
			},
			Remediation: []string{
				"Set repository.root in keyseal.yaml to a directory that exists on disk.",
			},
		})
		return false
	}
	if !info.IsDir() {
		result.Add(CheckResult{
			Name:     "repository.root",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "repository.root resolves to a file instead of a directory",
			Details: []string{
				fmt.Sprintf("Resolved repository root: %s", repoRoot),
			},
			Remediation: []string{
				"Set repository.root in keyseal.yaml to a directory path.",
			},
		})
		return false
	}
	result.Add(CheckResult{
		Name:     "repository.root",
		Status:   StatusOK,
		Severity: SeverityInformational,
		Summary:  "repository.root resolves to a directory",
		Details: []string{
			fmt.Sprintf("Resolved repository root: %s", repoRoot),
		},
	})
	return true
}

func runOutputPathChecks(cfg config.Config, result *Result) {
	mode, err := fsutil.ParseFileMode(cfg.Defaults.FileMode)
	if err != nil {
		result.Add(CheckResult{
			Name:     "defaults.file_mode",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "defaults.file_mode is invalid",
			Details: []string{
				fmt.Sprintf("Configured value: %q", cfg.Defaults.FileMode),
				fmt.Sprintf("Parse error: %v", err),
			},
			Remediation: []string{
				`Set defaults.file_mode to a four-digit owner-only mode such as "0600".`,
			},
		})
	} else if !fsutil.IsSafeFileMode(mode) {
		result.Add(CheckResult{
			Name:     "defaults.file_mode",
			Status:   StatusWarn,
			Severity: SeverityWarning,
			Summary:  fmt.Sprintf("defaults.file_mode %s allows group or world access", cfg.Defaults.FileMode),
			Details: []string{
				"Rendered secrets written with this mode may be readable by other users on the system.",
			},
			Remediation: []string{
				`Change defaults.file_mode to "0600" in keyseal.yaml.`,
			},
		})
	} else {
		result.Add(CheckResult{
			Name:     "defaults.file_mode",
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  fmt.Sprintf("defaults.file_mode %s is owner-only", cfg.Defaults.FileMode),
		})
	}

	pathDetails := []string{}
	pathFixes := []string{}
	pathStatus := StatusOK
	pathSeverity := SeverityInformational

	checkOutputPath := func(label, path string) {
		if err := fsutil.ValidateOutputPath(path); err != nil {
			pathStatus = StatusWarn
			pathSeverity = SeverityWarning
			pathDetails = append(pathDetails, fmt.Sprintf("%s is invalid: %v", label, err))
			pathFixes = append(pathFixes, fmt.Sprintf("Set %s to an absolute runtime path or a safe relative path inside the repo.", label))
			return
		}

		// Relative output paths are valid, but they are easy to commit by mistake
		// when users render secrets into the repository working tree.
		if !filepath.IsAbs(path) {
			pathStatus = StatusWarn
			pathSeverity = SeverityWarning
			pathDetails = append(pathDetails, fmt.Sprintf("%s is relative: %s", label, path))
			pathFixes = append(pathFixes, fmt.Sprintf("Consider changing %s to an absolute runtime path like /run/secrets/... to reduce accidental commits.", label))
		}
	}

	checkOutputPath("defaults.output_dir", cfg.Defaults.OutputDir)
	for profileName, profile := range cfg.Profiles {
		for idx, renderProfile := range profile.Renders {
			if renderProfile.Out == "" {
				continue
			}
			checkOutputPath(fmt.Sprintf("profiles.%s.renders[%d].out", profileName, idx), renderProfile.Out)
		}
	}

	if len(pathDetails) == 0 {
		result.Add(CheckResult{
			Name:     "render output paths",
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  "Configured render output paths look sane",
			Details: []string{
				fmt.Sprintf("defaults.output_dir: %s", cfg.Defaults.OutputDir),
			},
		})
		return
	}

	result.Add(CheckResult{
		Name:        "render output paths",
		Status:      pathStatus,
		Severity:    pathSeverity,
		Summary:     "Some configured render output paths are risky or invalid",
		Details:     pathDetails,
		Remediation: dedupeStrings(pathFixes),
	})
}

func runSOPSConfigChecks(cwd string, result *Result) {
	sopsPath := filepath.Join(cwd, ".sops.yaml")
	data, err := os.ReadFile(sopsPath)
	if err != nil {
		result.Add(CheckResult{
			Name:     ".sops.yaml",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "Missing .sops.yaml",
			Details: []string{
				fmt.Sprintf("Expected SOPS creation rules at %s.", sopsPath),
				"Encrypted add/edit flows depend on valid creation rules to choose recipients.",
			},
			Remediation: []string{
				"Run `keyseal init` to create a starter .sops.yaml, then replace the placeholder recipients.",
			},
		})
		return
	}

	info, err := inspectSOPSConfig(data)
	if err != nil {
		result.Add(CheckResult{
			Name:     ".sops.yaml",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  ".sops.yaml could not be parsed",
			Details: []string{
				fmt.Sprintf("Parse error: %v", err),
			},
			Remediation: []string{
				"Fix the YAML syntax or recreate the file with `keyseal init`.",
			},
		})
		return
	}

	result.Add(CheckResult{
		Name:     ".sops.yaml",
		Status:   StatusOK,
		Severity: SeverityInformational,
		Summary:  "Parsed .sops.yaml",
		Details: []string{
			fmt.Sprintf("Discovered %d creation rule(s).", info.CreationRuleCount),
		},
	})

	switch {
	case info.CreationRuleCount == 0:
		result.Add(CheckResult{
			Name:     ".sops.yaml creation rules",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  ".sops.yaml does not define any creation rules",
			Details: []string{
				"SOPS needs at least one creation rule to decide how new encrypted files should be protected.",
			},
			Remediation: []string{
				"Add at least one real creation rule with valid recipients to .sops.yaml.",
			},
		})
	case info.UsableRuleCount == 0:
		result.Add(CheckResult{
			Name:     ".sops.yaml creation rules",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  ".sops.yaml creation rules do not contain usable recipients",
			Details: []string{
				"Rules exist, but none of them include recipient material such as age, pgp, kms, or key_groups entries.",
			},
			Remediation: []string{
				"Add real recipients to each creation rule before relying on encrypted workflows.",
			},
		})
	default:
		result.Add(CheckResult{
			Name:     ".sops.yaml creation rules",
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  fmt.Sprintf("%d creation rule(s) include recipient material", info.UsableRuleCount),
		})
	}

	if len(info.Placeholders) > 0 {
		result.Add(CheckResult{
			Name:     ".sops.yaml placeholders",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  ".sops.yaml still contains placeholder recipient values",
			Details: []string{
				fmt.Sprintf("Found placeholders: %s", strings.Join(info.Placeholders, ", ")),
				"New encrypted files created with these placeholders will not be configured for real recipients.",
			},
			Remediation: []string{
				"Replace the placeholder recipients in .sops.yaml with real age recipients before relying on encrypted add/edit flows.",
			},
		})
		return
	}

	result.Add(CheckResult{
		Name:     ".sops.yaml placeholders",
		Status:   StatusOK,
		Severity: SeverityInformational,
		Summary:  "No obvious placeholder recipients were found in .sops.yaml",
	})
}

func runSOPSBinaryChecks(cfg config.Config, cwd string, result *Result) bool {
	binary := cfg.SOPS.Binary
	resolved, err := sopsutil.LookPath(binary)
	if err != nil {
		result.Add(CheckResult{
			Name:     "sops binary",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "Configured sops binary could not be resolved",
			Details: []string{
				fmt.Sprintf("Configured value: %q", binary),
				fmt.Sprintf("Lookup error: %v", err),
				"Encrypted add, edit, render, exec, and decrypt validation all depend on a working SOPS binary.",
			},
			Remediation: []string{
				fmt.Sprintf("Install SOPS and make %q available in PATH, or update sops.binary in keyseal.yaml.", binary),
			},
		})
		return false
	}

	version, err := sopsutil.Version(binary, config.ResolvePath(cwd, cfg.SOPS.AgeKeyFile))
	if err != nil {
		result.Add(CheckResult{
			Name:     "sops binary",
			Status:   StatusFail,
			Severity: SeverityError,
			Summary:  "Configured sops binary was found but could not be executed",
			Details: []string{
				fmt.Sprintf("Resolved path: %s", resolved),
				fmt.Sprintf("Version check error: %v", err),
			},
			Remediation: []string{
				"Ensure the configured SOPS binary is executable and runs successfully with `sops --version`.",
			},
		})
		return false
	}

	details := []string{
		fmt.Sprintf("Resolved path: %s", resolved),
	}
	status := StatusOK
	severity := SeverityInformational
	summary := "SOPS binary is available"
	if version != "" {
		details = append(details, fmt.Sprintf("Version: %s", version))
	} else {
		status = StatusWarn
		severity = SeverityWarning
		summary = "SOPS binary is available, but version output was empty"
	}
	result.Add(CheckResult{
		Name:     "sops binary",
		Status:   status,
		Severity: severity,
		Summary:  summary,
		Details:  details,
	})
	return true
}

func runRepoArtifactChecks(repoRoot string, result *Result) {
	var details []string
	var fixes []string

	rootBinary := filepath.Join(repoRoot, "keyseal")
	if info, err := os.Stat(rootBinary); err == nil && info.Mode().IsRegular() {
		details = append(details, fmt.Sprintf("Built binary present in repository root: %s", rootBinary))
		fixes = append(fixes, "Remove the root-level binary and use ./bin/keyseal instead of building into the repository root.")
	}

	distDir := filepath.Join(repoRoot, "dist")
	if entries, err := os.ReadDir(distDir); err == nil && len(entries) > 0 {
		details = append(details, fmt.Sprintf("dist/ contains %d artifact(s).", len(entries)))
		fixes = append(fixes, "Clean dist/ before committing release artifacts or local packaging output.")
	}

	if len(details) == 0 {
		result.Add(CheckResult{
			Name:     "repo artifacts",
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  "No obvious generated artifacts were found in the repository root",
		})
		return
	}

	result.Add(CheckResult{
		Name:        "repo artifacts",
		Status:      StatusWarn,
		Severity:    SeverityWarning,
		Summary:     "Found generated artifacts that are easy to commit by mistake",
		Details:     details,
		Remediation: dedupeStrings(fixes),
	})
}

func runSecretChecks(cwd, repoRoot string, cfg config.Config, sopsAvailable bool, result *Result) error {
	files, err := repo.DiscoverEncryptedFiles(repoRoot, cfg.Repository.EncryptedExtension)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		result.Add(CheckResult{
			Name:     "secret discovery",
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  "No encrypted secret files were discovered yet",
			Details: []string{
				fmt.Sprintf("Looked for files ending in %s under %s.", cfg.Repository.EncryptedExtension, repoRoot),
			},
		})
		return nil
	}

	result.Add(CheckResult{
		Name:     "secret discovery",
		Status:   StatusOK,
		Severity: SeverityInformational,
		Summary:  fmt.Sprintf("Discovered %d encrypted secret file(s)", len(files)),
	})

	opts := schema.DefaultValidationOptions()
	opts.RequireValues = cfg.Validation.RequireValues
	opts.KeyPattern = cfg.Validation.KeyPattern

	skippedDecrypts := 0
	for _, file := range files {
		logical, err := repo.PathToLogicalName(repoRoot, file, cfg.Repository.EncryptedExtension)
		if err != nil {
			result.Add(CheckResult{
				Name:     "secret naming",
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  fmt.Sprintf("%s does not map cleanly back to a logical name", filepath.ToSlash(file)),
				Details: []string{
					fmt.Sprintf("Mapping error: %v", err),
				},
				Remediation: []string{
					fmt.Sprintf("Rename %s so it matches the configured %s naming convention.", filepath.ToSlash(file), cfg.Repository.EncryptedExtension),
				},
			})
			continue
		}
		roundTrip, err := repo.LogicalNameToPath(repoRoot, logical, cfg.Repository.EncryptedExtension)
		if err != nil || roundTrip != file {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  "Logical name round-trip validation failed",
				Details: []string{
					fmt.Sprintf("Expected path: %s", filepath.ToSlash(roundTrip)),
					fmt.Sprintf("Actual path: %s", filepath.ToSlash(file)),
				},
				Remediation: []string{
					"Rename the file so the logical name maps back to the same path consistently.",
				},
			})
			continue
		}

		rawFile, readErr := os.ReadFile(file)
		if readErr != nil {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  "Secret file could not be read",
				Details: []string{
					fmt.Sprintf("Read error: %v", readErr),
				},
				Remediation: []string{
					fmt.Sprintf("Fix file permissions or recreate %s.", filepath.ToSlash(file)),
				},
			})
			continue
		}

		// A valid env document without top-level sops metadata is usually a legacy
		// plaintext starter file or a manual edit that bypassed SOPS entirely.
		doc, _, parseErr := schema.ParseYAMLDocument(rawFile)
		if parseErr == nil && !schema.HasSOPSMetadata(rawFile) {
			if validateErr := doc.Validate(opts); validateErr == nil {
				result.Add(CheckResult{
					Name:     fmt.Sprintf("secret %s", logical),
					Status:   StatusFail,
					Severity: SeverityError,
					Summary:  fmt.Sprintf("%s appears to be plaintext rather than SOPS-encrypted content", filepath.ToSlash(file)),
					Details: []string{
						"This file parses as a valid env document but does not contain the top-level sops metadata block written by SOPS.",
						"This usually means the file was created by an older plaintext scaffold flow or was edited manually outside SOPS.",
					},
					Remediation: []string{
						fmt.Sprintf("Encrypt the file with `keyseal edit %s` or recreate it with `keyseal add %s`.", logical, logical),
					},
				})
				continue
			}
		}

		if !sopsAvailable {
			skippedDecrypts++
			continue
		}

		plaintext, err := sopsutil.DecryptFile(cfg.SOPS.Binary, config.ResolvePath(cwd, cfg.SOPS.AgeKeyFile), file)
		if err != nil {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  fmt.Sprintf("Failed to decrypt %s", filepath.ToSlash(file)),
				Details: []string{
					fmt.Sprintf("Decrypt error: %v", classifyDecryptError(err)),
					"Without successful decryption, Keyseal cannot validate the secret schema or env keys.",
				},
				Remediation: []string{
					"Confirm the SOPS recipients, local age key setup, and file access, then rerun `keyseal doctor`.",
				},
			})
			continue
		}

		doc, dupes, err := schema.ParseYAMLDocument(plaintext)
		if err != nil {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  fmt.Sprintf("Decrypted %s is not valid YAML", filepath.ToSlash(file)),
				Details: []string{
					fmt.Sprintf("Parse error: %v", err),
				},
				Remediation: []string{
					fmt.Sprintf("Open the file with `keyseal edit %s` and fix the YAML structure.", logical),
				},
			})
			continue
		}
		if err := doc.Validate(opts); err != nil {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusFail,
				Severity: SeverityError,
				Summary:  fmt.Sprintf("Decrypted %s does not match the supported env schema", filepath.ToSlash(file)),
				Details: []string{
					fmt.Sprintf("Validation error: %v", err),
				},
				Remediation: []string{
					fmt.Sprintf("Fix the secret with `keyseal edit %s` so it includes a supported kind: env document and valid env keys.", logical),
				},
			})
			continue
		}
		if len(dupes) > 0 {
			result.Add(CheckResult{
				Name:     fmt.Sprintf("secret %s", logical),
				Status:   StatusWarn,
				Severity: SeverityWarning,
				Summary:  fmt.Sprintf("%s contains duplicate env keys", filepath.ToSlash(file)),
				Details: []string{
					fmt.Sprintf("Duplicate keys in values: %s", strings.Join(dupes, ", ")),
					"YAML keeps the last value, which can hide earlier mistakes during review.",
				},
				Remediation: []string{
					fmt.Sprintf("Edit %s and remove the duplicate keys so each env var is defined once.", filepath.ToSlash(file)),
				},
			})
			continue
		}
		result.Add(CheckResult{
			Name:     fmt.Sprintf("secret %s", logical),
			Status:   StatusOK,
			Severity: SeverityInformational,
			Summary:  fmt.Sprintf("%s decrypted and validated successfully", filepath.ToSlash(file)),
		})
	}

	if !sopsAvailable && len(files) > 0 {
		result.Add(CheckResult{
			Name:     "decrypt validation",
			Status:   StatusSkip,
			Severity: SeverityInformational,
			Summary:  fmt.Sprintf("Skipped decrypt validation for %d file(s) because the configured SOPS binary is unavailable", skippedDecrypts),
			Remediation: []string{
				"Install or fix the configured SOPS binary, then rerun `keyseal doctor` for full secret validation.",
			},
		})
	}
	return nil
}

func classifyDecryptError(err error) string {
	if errors.Is(err, sopsutil.ErrBinaryNotFound) {
		return "sops binary not found"
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "sops command failed"
	}
	return err.Error()
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
