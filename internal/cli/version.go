package cli

import (
	"fmt"

	"github.com/jrpbuilds/keyseal/internal/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Show build version information",
		Example: "  keyseal version\n  keyseal version --short",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				fmt.Fprintln(cmd.OutOrStdout(), buildinfo.Short())
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), buildinfo.Full())
			return nil
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Print only the version string")
	return cmd
}
