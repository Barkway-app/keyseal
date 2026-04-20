package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Barkway-app/keyseal/internal/config"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
	"github.com/spf13/cobra"
)

func newEditCommand() *cobra.Command {
	var commitConfig commitFlags

	cmd := &cobra.Command{
		Use:   "edit <logical-name>",
		Short: "Open an encrypted file with SOPS",
		Long: "Resolve a logical name to <logical-name>.enc.yaml and run `sops <file>` using the configured SOPS binary.\n" +
			"Use --commit or -m to commit the edited file after SOPS exits successfully. -m implies --commit.",
		Example: "  keyseal edit production/platform/app\n" +
			"  keyseal edit production/platform/app --commit\n" +
			"  keyseal edit production/platform/app -m \"Update app secret\"",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadGitWorkflowContext()
			if err != nil {
				return err
			}
			commitDecision, err := resolveCommitDecision(cmd, ctx.Config, commitConfig)
			if err != nil {
				return err
			}
			if commitDecision.Enabled {
				if err := ensureRepoReadyForCommit(ctx); err != nil {
					return err
				}
			}
			ageKeyFile := config.ResolvePath(ctx.CWD, ctx.Config.SOPS.AgeKeyFile)

			target, err := repo.LogicalNameToPath(ctx.KeysealRoot, args[0], ctx.Config.Repository.EncryptedExtension)
			if err != nil {
				return err
			}
			before, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("read encrypted file before edit: %w", err)
			}
			if err := sopsutil.EditFile(ctx.Config.SOPS.Binary, ageKeyFile, target); err != nil {
				return err
			}
			after, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("read encrypted file after edit: %w", err)
			}
			if !commitDecision.Enabled || bytes.Equal(before, after) {
				if commitDecision.Enabled && cmd.OutOrStdout() != nil && bytes.Equal(before, after) {
					fmt.Fprintln(cmd.OutOrStdout(), "file unchanged; no Git commit created")
				}
				return nil
			}
			message := commitDecision.Message
			if message == "" {
				message = defaultCommitMessageForSecretEdit(args[0])
			}
			if err := commitPaths(ctx, []string{target}, message); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "committed Git change: %s\n", message)
			return nil
		},
	}
	bindCommitFlags(cmd, &commitConfig)
	return cmd
}
