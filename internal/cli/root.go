// Package cli wires the Cobra command tree to the internal workflow packages.
//
// The package intentionally keeps business logic out of command setup so the
// CLI surface stays readable and the underlying behavior remains testable.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Barkway-app/keyseal/internal/config"
	"github.com/spf13/cobra"
)

// Version is the user-facing CLI version string. Release builds can override it
// at link time with -ldflags.
var Version = "dev"

// Execute runs the Keyseal root command.
func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keyseal",
		Short: "Small CLI for SOPS + age repo-backed secrets workflows",
		Long: "Keyseal is a workflow layer over SOPS, age, and Git-backed encrypted files.\n" +
			"It helps bootstrap a repo layout, scaffold starter secret documents, render decrypted env data,\n" +
			"run child commands with injected secrets, and validate repository health.",
		Example: "  keyseal init\n" +
			"  keyseal add production/platform/app --template laravel\n" +
			"  keyseal edit production/platform/app\n" +
			"  keyseal render production/platform/app --stdout\n" +
			"  keyseal exec production/platform/app -- env | grep APP_\n" +
			"  keyseal doctor",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.AddCommand(
		newInitCommand(),
		newAddCommand(),
		newEditCommand(),
		newRenderCommand(),
		newExecCommand(),
		newDoctorCommand(),
	)
	return cmd
}

// loadConfigFromCWD loads keyseal.yaml from the current working directory.
//
// Commands operate relative to the active repository root, so the CLI fails
// early with a clear message when keyseal.yaml is missing.
func loadConfigFromCWD() (config.Config, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("get working directory: %w", err)
	}
	cfg, err := config.Load(filepath.Join(cwd, config.DefaultConfigPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.Config{}, "", fmt.Errorf("keyseal.yaml not found in %s; run `keyseal init` first", cwd)
		}
		return config.Config{}, "", err
	}
	return cfg, cwd, nil
}
