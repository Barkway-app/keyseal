// Package toolcheck resolves and probes external command-line tools used by
// Keyseal without tying the checks to a specific workflow package.
package toolcheck

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrBinaryNotFound identifies failures where a configured executable could
// not be resolved as either a PATH command or an explicit filesystem path.
var ErrBinaryNotFound = errors.New("binary not found")

// ProbeResult describes a successfully resolved and optionally version-checked
// external tool.
type ProbeResult struct {
	// Configured is the command or path value supplied by configuration.
	Configured string
	// Resolved is the executable path returned by exec.LookPath.
	Resolved string
	// Version is the first non-empty line from the tool's version output.
	Version string
}

// Resolve finds a configured binary using the same rules exec.Command uses for
// PATH lookups and explicit paths.
func Resolve(binary string) (string, error) {
	if strings.TrimSpace(binary) == "" {
		return "", fmt.Errorf("%w: empty binary name", ErrBinaryNotFound)
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, binary)
	}
	return path, nil
}

// Probe resolves a binary and runs it with the supplied version arguments,
// returning the first non-empty line emitted on stdout or stderr.
func Probe(binary string, env []string, versionArgs ...string) (ProbeResult, error) {
	resolved, err := Resolve(binary)
	if err != nil {
		return ProbeResult{}, err
	}
	result := ProbeResult{
		Configured: binary,
		Resolved:   resolved,
	}
	if len(versionArgs) == 0 {
		return result, nil
	}
	cmd := exec.Command(binary, versionArgs...)
	if env != nil {
		cmd.Env = env
	} else {
		cmd.Env = os.Environ()
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("run %s %s: %s", binary, strings.Join(versionArgs, " "), sanitizeOutput(stderr.Bytes(), stdout.Bytes(), err))
	}
	result.Version = firstOutputLine(stdout.String(), stderr.String())
	return result, nil
}

// firstOutputLine returns the first non-empty line from stdout candidates and
// then stderr candidates, matching tools that print versions to either stream.
func firstOutputLine(candidates ...string) string {
	for _, candidate := range candidates {
		for _, line := range strings.Split(candidate, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				return line
			}
		}
	}
	return ""
}

// sanitizeOutput condenses command failure output into a short diagnostic that
// avoids flooding command and doctor messages.
func sanitizeOutput(primary, fallback []byte, err error) string {
	text := strings.TrimSpace(string(primary))
	if text == "" {
		text = strings.TrimSpace(string(fallback))
	}
	if text == "" {
		text = err.Error()
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 3 {
		lines = lines[:3]
	}
	return strings.Join(lines, "; ")
}
