package cli

import (
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
	"github.com/spf13/cobra"
)

func newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "edit <logical-name>",
		Short:   "Open an encrypted file with SOPS",
		Long:    "Resolve a logical name to <logical-name>.enc.yaml and run `sops <file>` using the configured SOPS binary.",
		Example: "  keyseal edit production/platform/app",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cwd, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			target, err := repo.LogicalNameToPath(cfg.RepoRoot(cwd), args[0], cfg.Repository.EncryptedExtension)
			if err != nil {
				return err
			}
			return sopsutil.EditFile(cfg.SOPS.Binary, target)
		},
	}
}
