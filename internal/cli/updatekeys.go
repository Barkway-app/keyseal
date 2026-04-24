package cli

import (
	"fmt"
	"os"

	"github.com/Barkway-app/keyseal/internal/updatekeys"
	"github.com/spf13/cobra"
)

// newUpdateKeysCommand wires the Cobra surface for recipient synchronization
// while leaving batch behavior in the updatekeys package.
func newUpdateKeysCommand() *cobra.Command {
	var all bool
	var yes bool
	var commit bool
	var message string

	cmd := &cobra.Command{
		Use:   "updatekeys [logical-name...]",
		Short: "Sync SOPS recipients for encrypted files",
		Long: "Run `sops updatekeys` for one or more Keyseal-managed encrypted files using the current .sops.yaml.\n" +
			"This syncs recipients only; it does not rotate secret values or the SOPS data encryption key.\n" +
			"Pass logical names explicitly, or use --all to process every discovered encrypted file.",
		Example: "  keyseal updatekeys production/platform/app\n" +
			"  keyseal updatekeys production/platform/app staging/platform/app --yes\n" +
			"  keyseal updatekeys --all --yes\n" +
			"  keyseal updatekeys --all --yes -m \"Sync SOPS recipients\"",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !all {
				return fmt.Errorf("pass one or more logical names, or use --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			if message != "" {
				commit = true
			}
			_, err = updatekeys.Run(updatekeys.Options{
				CWD:      cwd,
				Logicals: args,
				All:      all,
				Yes:      yes,
				Commit:   commit,
				Message:  message,
				Stdout:   cmd.OutOrStdout(),
				Stderr:   cmd.ErrOrStderr(),
			})
			return err
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Update keys for every discovered encrypted secret file")
	cmd.Flags().BoolVar(&yes, "yes", false, "Pass -y to sops updatekeys for non-interactive confirmation")
	cmd.Flags().BoolVar(&commit, "commit", false, "Stage changed encrypted files and create a Git commit")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Git commit message; implies --commit")
	return cmd
}
