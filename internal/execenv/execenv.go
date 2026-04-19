// Package execenv runs child processes with merged decrypted environment values.
package execenv

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// MergeEnv overlays secret values on top of the current process environment.
//
// Existing variables are inherited first, then secret values override matching
// keys for the child process only.
func MergeEnv(base []string, overrides map[string]string) []string {
	values := map[string]string{}
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	merged := make([]string, 0, len(keys))
	for _, key := range keys {
		merged = append(merged, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return merged
}

// Run executes the provided command with the merged environment.
func Run(command []string, env map[string]string) error {
	if len(command) == 0 {
		return fmt.Errorf("command is required")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = MergeEnv(os.Environ(), env)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
