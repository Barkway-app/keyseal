package sopsutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// UpdateKeys runs `sops updatekeys` for an encrypted file.
func UpdateKeys(binary, ageKeyFile, path string, yes bool) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("stat encrypted file: %w", err)
	}
	if _, err := LookPath(binary); err != nil {
		return err
	}
	args := []string{"updatekeys"}
	if yes {
		args = append(args, "-y")
	}
	args = append(args, path)
	cmd := exec.Command(binary, args...)
	cmd.Env = commandEnv(ageKeyFile)
	if yes {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("update keys for %s with sops: %s", path, sanitizeOutput(stderr.Bytes(), stdout.Bytes()))
		}
		return nil
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update keys for %s with sops: %w", path, err)
	}
	return nil
}
