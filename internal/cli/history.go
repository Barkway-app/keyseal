package cli

import (
	"fmt"

	"github.com/Barkway-app/keyseal/internal/gitutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/spf13/cobra"
)

func newHistoryCommand() *cobra.Command {
	var oneline bool

	cmd := &cobra.Command{
		Use:   "history <logical-name>",
		Short: "Show Git history for one secret file",
		Long: "Resolve a logical name to its encrypted file path and show the Git history for that file.\n" +
			"The command uses Git as the source of truth for secret history.\n" +
			"Use --oneline for a compact day-to-day log view.",
		Example: "  keyseal history production/platform/app\n" +
			"  keyseal history production/platform/app --oneline",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadGitWorkflowContext()
			if err != nil {
				return err
			}
			target, err := repo.LogicalNameToPath(ctx.KeysealRoot, args[0], ctx.Config.Repository.EncryptedExtension)
			if err != nil {
				return err
			}
			historyText, err := gitutil.History("git", ctx.GitRoot, target, oneline)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), historyText)
			return nil
		},
	}
	cmd.Flags().BoolVar(&oneline, "oneline", false, "Show compact one-line Git history")
	return cmd
}
