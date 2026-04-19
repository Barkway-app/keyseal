package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/Barkway-app/keyseal/internal/execenv"
	"github.com/Barkway-app/keyseal/internal/render"
	"github.com/spf13/cobra"
)

func newExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <logical-name...> -- <command...>",
		Short: "Run a child process with decrypted env vars injected",
		Long: "Decrypt and merge one or more env documents, inherit the current process environment,\n" +
			"override matching keys with secret values, and execute the child command after `--`.",
		Example: "  keyseal exec production/platform/app -- env | grep APP_\n" +
			"  keyseal exec production/platform/app production/platform/stripe -- php artisan queue:work",
		Args: func(cmd *cobra.Command, args []string) error {
			sep := cmd.ArgsLenAtDash()
			if sep < 1 {
				return errors.New("require at least one logical name before `--`")
			}
			if len(args[sep:]) == 0 {
				return errors.New("require a child command after `--`")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cwd, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			sep := cmd.ArgsLenAtDash()
			docs, err := loadDocuments(cfg, cwd, args[:sep])
			if err != nil {
				return err
			}
			if err := execenv.Run(args[sep:], render.MergeEnvDocs(docs)); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return commandExitError{code: exitErr.ExitCode()}
				}
				return fmt.Errorf("run child command: %w", err)
			}
			return nil
		},
	}
}

type commandExitError struct {
	code int
}

func (e commandExitError) Error() string {
	return ""
}

func (e commandExitError) ExitCode() int {
	return e.code
}
