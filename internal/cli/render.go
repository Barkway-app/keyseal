package cli

import (
	"errors"
	"fmt"

	"github.com/Barkway-app/keyseal/internal/fsutil"
	"github.com/Barkway-app/keyseal/internal/render"
	"github.com/spf13/cobra"
)

func newRenderCommand() *cobra.Command {
	var format string
	var out string
	var stdout bool
	var mode string
	var force bool

	cmd := &cobra.Command{
		Use:   "render <logical-name...>",
		Short: "Decrypt, merge, and render secret files",
		Long: "Decrypt and merge one or more env documents, then render them as dotenv, JSON, or YAML.\n" +
			"Later files override earlier ones. Use --stdout to print secret values, or --out to write atomically to a file.",
		Example: "  keyseal render production/platform/app --stdout\n" +
			"  keyseal render production/platform/app staging/platform/stripe --format json --out ./runtime/app-secrets.json",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case stdout && out != "":
				return errors.New("choose either --out or --stdout, not both")
			case !stdout && out == "":
				return errors.New("require either --out or --stdout")
			}
			cfg, cwd, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			if format == "" {
				format = cfg.Defaults.OutputFormat
			}
			if mode == "" {
				mode = cfg.Defaults.FileMode
			}

			docs, err := loadDocuments(cfg, cwd, args)
			if err != nil {
				return err
			}
			merged := render.MergeEnvDocs(docs)
			body, err := render.Render(format, merged)
			if err != nil {
				return err
			}

			if stdout {
				_, err = cmd.OutOrStdout().Write(body)
				return err
			}

			fileMode, err := fsutil.ParseFileMode(mode)
			if err != nil {
				return err
			}
			if !fsutil.IsSafeFileMode(fileMode) && !force {
				return fmt.Errorf("mode %s is not owner-only; use --force to override", mode)
			}
			if err := fsutil.AtomicWriteFile(out, body, fileMode, force); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "rendered %d secret file(s) to %s\n", len(args), out)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "", "Render format: dotenv, json, yaml (defaults to keyseal.yaml)")
	cmd.Flags().StringVar(&out, "out", "", "Output file path for atomic plaintext writes")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Write decrypted secret data to stdout")
	cmd.Flags().StringVar(&mode, "mode", "", "Output file mode as an octal string (defaults to keyseal.yaml)")
	cmd.Flags().BoolVar(&force, "force", false, "Allow unsafe modes and overwrite existing output")
	return cmd
}
