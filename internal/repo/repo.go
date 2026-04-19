// Package repo owns logical-name validation and deterministic file mapping.
package repo

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ValidateLogicalName rejects paths that could escape the repository layout or
// blur the logical-name contract with raw filesystem paths.
func ValidateLogicalName(logical string) error {
	if logical == "" {
		return errors.New("logical name is required")
	}
	if filepath.IsAbs(logical) {
		return fmt.Errorf("logical name %q must be relative", logical)
	}
	if strings.Contains(logical, "\\") {
		return fmt.Errorf("logical name %q must use forward slashes", logical)
	}
	if strings.HasSuffix(logical, ".yaml") || strings.Contains(logical, ".enc.") {
		return fmt.Errorf("logical name %q must not include a file extension", logical)
	}
	clean := filepath.ToSlash(filepath.Clean(logical))
	if clean != logical {
		return fmt.Errorf("logical name %q is not normalized", logical)
	}
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return fmt.Errorf("logical name %q must not traverse directories", logical)
	}
	parts := strings.Split(clean, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("logical name %q contains an invalid path segment", logical)
		}
	}
	return nil
}

// LogicalNameToPath converts a logical name like production/platform/app into
// an encrypted file path like production/platform/app.enc.yaml.
func LogicalNameToPath(root, logical, ext string) (string, error) {
	if err := ValidateLogicalName(logical); err != nil {
		return "", err
	}
	if ext == "" {
		return "", errors.New("encrypted extension is required")
	}
	return filepath.Join(root, filepath.FromSlash(logical)+ext), nil
}

// PathToLogicalName converts an encrypted file path back into its logical name.
func PathToLogicalName(root, path, ext string) (string, error) {
	if !strings.HasSuffix(path, ext) {
		return "", fmt.Errorf("path %q does not end with %q", path, ext)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("make relative path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("path %q is outside repository root", path)
	}
	logical := strings.TrimSuffix(rel, ext)
	if err := ValidateLogicalName(logical); err != nil {
		return "", err
	}
	return logical, nil
}

// DiscoverEncryptedFiles walks the repository root and returns every file that
// matches the configured encrypted extension.
func DiscoverEncryptedFiles(root, ext string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ext) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
