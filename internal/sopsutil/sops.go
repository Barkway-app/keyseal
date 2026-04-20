// Package sopsutil wraps subprocess interaction with the external sops binary.
package sopsutil

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrBinaryNotFound is returned when the configured sops binary cannot be found.
var ErrBinaryNotFound = errors.New("sops binary not found")

// LookPath resolves the configured sops binary in PATH.
func LookPath(binary string) (string, error) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, binary)
	}
	return path, nil
}

// DecryptFile decrypts a file by invoking `sops --decrypt <file>`.
//
// Stdout and stderr are captured separately so successful decryptions cannot be
// polluted by warning output and failure messages can be sanitized without
// accidentally echoing plaintext to the caller.
func DecryptFile(binary, path string) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat encrypted file: %w", err)
	}
	if _, err := LookPath(binary); err != nil {
		return nil, err
	}
	cmd := exec.Command(binary, "--decrypt", path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("decrypt %s with sops: %s", path, sanitizeOutput(stderr.Bytes(), stdout.Bytes()))
	}
	return stdout.Bytes(), nil
}

// EditFile launches `sops <file>` attached to the current terminal.
func EditFile(binary, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("stat encrypted file: %w", err)
	}
	if _, err := LookPath(binary); err != nil {
		return err
	}
	cmd := exec.Command(binary, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run sops editor for %s: %w", path, err)
	}
	return nil
}

// Version returns the first non-empty line from `sops --version`.
func Version(binary string) (string, error) {
	if _, err := LookPath(binary); err != nil {
		return "", err
	}
	cmd := exec.Command(binary, "--version")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run sops --version: %s", sanitizeOutput(stderr.Bytes(), stdout.Bytes()))
	}
	for _, candidate := range []string{stdout.String(), stderr.String()} {
		for _, line := range strings.Split(candidate, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				return line, nil
			}
		}
	}
	return "", nil
}

// EncryptFile encrypts plaintext by writing it to a secure temp file and
// invoking `sops encrypt --filename-override <target> <tempfile>`.
//
// The caller owns the final write path so encrypted mode can keep plaintext out
// of the destination `.enc.yaml` file entirely.
func EncryptFile(binary string, plaintext []byte, filenameOverride string) ([]byte, error) {
	if _, err := LookPath(binary); err != nil {
		return nil, err
	}

	temp, err := os.CreateTemp("", "keyseal-plaintext-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return nil, fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := temp.Write(plaintext); err != nil {
		temp.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	cmd := exec.Command(binary, "encrypt", "--filename-override", filenameOverride, tempPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("encrypt %s with sops: %s", filepath.Base(filenameOverride), sanitizeOutput(stderr.Bytes(), stdout.Bytes()))
	}
	return stdout.Bytes(), nil
}

func sanitizeOutput(primary, fallback []byte) string {
	text := strings.TrimSpace(string(primary))
	if text == "" {
		text = strings.TrimSpace(string(fallback))
	}
	if text == "" {
		return "sops command failed"
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 3 {
		lines = lines[:3]
	}
	return strings.Join(lines, "; ")
}
