package cli

import (
	"encoding/json"
	"fmt"

	"github.com/Barkway-app/keyseal/internal/gitutil"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status [logical-name]",
		Short: "Show Git status for Keyseal-managed files",
		Long: "Show a focused Git status view for Keyseal-managed files.\n" +
			"This includes encrypted secret files plus keyseal.yaml and .sops.yaml when they are changed.\n" +
			"When a logical name is provided, the output is limited to that secret file.\n" +
			"Use --json for script-friendly structured output.",
		Example: "  keyseal status\n" +
			"  keyseal status production/platform/app\n" +
			"  keyseal status --json",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadGitWorkflowContext()
			if err != nil {
				return err
			}
			entries, err := relevantStatusEntries(ctx)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				target, err := repo.LogicalNameToPath(ctx.KeysealRoot, args[0], ctx.Config.Repository.EncryptedExtension)
				if err != nil {
					return err
				}
				pathspec, err := gitutil.Pathspec(ctx.GitRoot, target)
				if err != nil {
					return err
				}
				entries = filterStatusEntriesByRepoPath(entries, pathspec)
			}
			if jsonOutput {
				return writeStatusJSON(cmd, entries)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatStatusEntries(entries))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write Keyseal-managed Git status as JSON")
	return cmd
}

func filterStatusEntriesByRepoPath(entries []gitutil.StatusEntry, repoPath string) []gitutil.StatusEntry {
	filtered := make([]gitutil.StatusEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Path == repoPath || entry.OriginalPath == repoPath {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type statusJSONEntry struct {
	Code         string `json:"code"`
	Staged       string `json:"staged"`
	Unstaged     string `json:"unstaged"`
	Path         string `json:"path"`
	OriginalPath string `json:"original_path,omitempty"`
}

type statusJSONPayload struct {
	Count   int               `json:"count"`
	Entries []statusJSONEntry `json:"entries"`
}

func writeStatusJSON(cmd *cobra.Command, entries []gitutil.StatusEntry) error {
	payload := statusJSONPayload{
		Count:   len(entries),
		Entries: make([]statusJSONEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		payload.Entries = append(payload.Entries, statusJSONEntry{
			Code:         entry.Code(),
			Staged:       string(entry.Staged),
			Unstaged:     string(entry.Unstaged),
			Path:         entry.Path,
			OriginalPath: entry.OriginalPath,
		})
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
