package cli

import (
	"errors"
	"fmt"

	"github.com/Barkway-app/keyseal/internal/gitutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/spf13/cobra"
)

func newRollbackCommand() *cobra.Command {
	var revision string
	var dryRun bool
	var commitConfig commitFlags

	cmd := &cobra.Command{
		Use:   "rollback <logical-name> --to <commit>",
		Short: "Restore a secret file from Git history",
		Long: "Restore the encrypted file for a logical name from a previous Git commit.\n" +
			"Rollback is Git-based file restore, not semantic secret reconstruction.\n" +
			"The command refuses to overwrite local changes for the target file.\n" +
			"Use --dry-run to preview the rollback target without changing the working tree, even if the file is dirty.",
		Example: "  keyseal rollback production/platform/app --to abc1234\n" +
			"  keyseal rollback production/platform/app --to abc1234 --dry-run\n" +
			"  keyseal rollback production/platform/app --to abc1234 -m \"Rollback app secret\"",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if revision == "" {
				return errors.New("--to is required")
			}

			ctx, err := loadGitWorkflowContext()
			if err != nil {
				return err
			}
			if err := ensureRepoReadyForCommit(ctx); err != nil {
				return err
			}
			commitDecision, err := resolveCommitDecision(cmd, ctx.Config, commitConfig)
			if err != nil {
				return err
			}

			target, err := repo.LogicalNameToPath(ctx.KeysealRoot, args[0], ctx.Config.Repository.EncryptedExtension)
			if err != nil {
				return err
			}
			exists, err := gitutil.RevisionContainsPath("git", ctx.GitRoot, revision, target)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("commit %s does not contain %s", revision, args[0])
			}
			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "would restore %s from %s\n", args[0], revision)
				return nil
			}
			dirty, err := gitutil.HasLocalChanges("git", ctx.GitRoot, target)
			if err != nil {
				return err
			}
			if dirty {
				return fmt.Errorf("%w: %s", gitutil.ErrFileHasLocalChanges, args[0])
			}
			if err := gitutil.RestoreFile("git", ctx.GitRoot, revision, target); err != nil {
				return err
			}
			if !commitDecision.Enabled {
				fmt.Fprintf(cmd.OutOrStdout(), "restored %s from %s\n", args[0], revision)
				return nil
			}
			changed, err := gitutil.HasLocalChanges("git", ctx.GitRoot, target)
			if err != nil {
				return err
			}
			if !changed {
				return gitutil.ErrNothingToCommit
			}
			message := commitDecision.Message
			if message == "" {
				message = defaultCommitMessageForSecretRollback(args[0], revision)
			}
			if err := commitPaths(ctx, []string{target}, message); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restored %s from %s\ncommitted Git change: %s\n", args[0], revision, message)
			return nil
		},
	}

	cmd.Flags().StringVar(&revision, "to", "", "Git commit to restore from")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the target rollback without modifying the working tree")
	bindCommitFlags(cmd, &commitConfig)
	return cmd
}
