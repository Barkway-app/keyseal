package cli

import (
	"fmt"

	"github.com/jrpbuilds/keyseal/internal/gitutil"
	"github.com/jrpbuilds/keyseal/internal/repo"
	"github.com/spf13/cobra"
)

func newDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <logical-name>",
		Short: "Show Git diff for one secret file",
		Long: "Resolve a logical name to its encrypted file path and show the Git diff for that file.\n" +
			"The command is read-only and works in a dirty repository.",
		Example: "  keyseal diff production/platform/app",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadGitWorkflowContext()
			if err != nil {
				return err
			}
			target, err := repo.LogicalNameToPath(ctx.KeysealRoot, args[0], ctx.Config.Repository.EncryptedExtension)
			if err != nil {
				return err
			}
			diffText, err := gitutil.Diff("git", ctx.GitRoot, target)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), diffText)
			return nil
		},
	}
}
