package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Barkway-app/keyseal/internal/fsutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/schema"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
	"github.com/Barkway-app/keyseal/internal/templates"
	"github.com/spf13/cobra"
)

func newAddCommand() *cobra.Command {
	var kind string
	var templateName string
	var force bool

	cmd := &cobra.Command{
		Use:   "add <logical-name>",
		Short: "Create a starter env document",
		Args:  cobra.ExactArgs(1),
		Long: "Create a starter secret document at <logical-name>.enc.yaml.\n" +
			"Keyseal uses a non-interactive SOPS flow that encrypts the starter document before writing the final file.\n" +
			"No plaintext starter content is written to the final .enc.yaml path.",
		Example: "  keyseal add production/platform/app\n" +
			"  keyseal add production/platform/app --template laravel\n" +
			"  keyseal edit production/platform/app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if kind != "env" {
				return fmt.Errorf("unsupported kind %q; only env is supported in v1 RC", kind)
			}

			cfg, cwd, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			root := cfg.RepoRoot(cwd)

			logical := args[0]
			target, err := repo.LogicalNameToPath(root, logical, cfg.Repository.EncryptedExtension)
			if err != nil {
				return err
			}

			doc, err := templates.Build(logical, templateName)
			if err != nil {
				return err
			}
			opts := schema.DefaultValidationOptions()
			opts.RequireValues = cfg.Validation.RequireValues
			opts.KeyPattern = cfg.Validation.KeyPattern
			if err := doc.Validate(opts); err != nil {
				return err
			}
			body, err := schema.MarshalYAMLDocument(doc)
			if err != nil {
				return fmt.Errorf("marshal starter document: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent directory: %w", err)
			}

			if err := fsutil.CheckWritableFilePath(target, force); err != nil {
				return err
			}
			ciphertext, err := sopsutil.EncryptFile(cfg.SOPS.Binary, body, target)
			if err != nil {
				return fmt.Errorf("encrypt starter document with %q: %w", cfg.SOPS.Binary, err)
			}
			if err := fsutil.AtomicWriteFile(target, ciphertext, 0o600, force); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created encrypted starter %s\nnext: run `keyseal edit %s` to review or update it with SOPS\n", target, logical)
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "env", "Secret document kind (only env is supported)")
	cmd.Flags().StringVar(&templateName, "template", "", "Starter template: "+strings.Join(templates.SupportedNames(), ", "))
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing target file")
	return cmd
}
