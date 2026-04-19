// Package buildinfo centralizes user-facing build metadata and formatting.
package buildinfo

import "fmt"

// These values are overridden in release builds with -ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// CLIName is the stable executable name shown in user-facing version output.
const CLIName = "keyseal"

// DisplayVersion returns the user-facing semantic version string.
func DisplayVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// ShortCommit returns a concise commit identifier for display.
func ShortCommit() string {
	if Commit == "" || Commit == "unknown" {
		return "unknown"
	}
	if len(Commit) <= 7 {
		return Commit
	}
	return Commit[:7]
}

// OneLine returns the `keyseal --version` output.
func OneLine() string {
	if shortCommit := ShortCommit(); shortCommit != "unknown" {
		return fmt.Sprintf("%s %s (%s)", CLIName, DisplayVersion(), shortCommit)
	}
	return fmt.Sprintf("%s %s", CLIName, DisplayVersion())
}

// Short returns the concise version string for scripting.
func Short() string {
	return DisplayVersion()
}

// Full returns the multiline version report.
func Full() string {
	return fmt.Sprintf("%s\ntag: %s\ncommit: %s\nbuilt: %s", OneLine(), Version, ShortCommit(), Date)
}
