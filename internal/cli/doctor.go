package cli

import (
	"fmt"
	"os"

	"github.com/Barkway-app/keyseal/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate repository and config health",
		Long: "Validate keyseal.yaml, .sops.yaml, SOPS availability, logical path mapping,\n" +
			"and decrypted env document shape without exposing secret values in normal output.",
		Example: "  keyseal doctor",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := doctor.Run(cwd)
			if err != nil {
				return err
			}
			for _, note := range result.Notes {
				fmt.Fprintf(cmd.OutOrStdout(), "note: %s\n", note)
			}
			for _, warning := range result.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", warning)
			}
			for _, failure := range result.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", failure)
			}
			if result.HasErrors() {
				return fmt.Errorf("doctor found %d error(s)", len(result.Errors))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "doctor checks passed")
			return nil
		},
	}
}
