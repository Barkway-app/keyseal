package cli

import (
	"fmt"

	"github.com/Barkway-app/keyseal/internal/config"
	"github.com/Barkway-app/keyseal/internal/repo"
	"github.com/Barkway-app/keyseal/internal/schema"
	"github.com/Barkway-app/keyseal/internal/sopsutil"
)

func loadDocuments(cfg config.Config, cwd string, logicalNames []string) ([]schema.EnvSecretDocument, error) {
	opts := schema.DefaultValidationOptions()
	opts.RequireValues = cfg.Validation.RequireValues
	opts.KeyPattern = cfg.Validation.KeyPattern

	root := cfg.RepoRoot(cwd)
	docs := make([]schema.EnvSecretDocument, 0, len(logicalNames))
	for _, logical := range logicalNames {
		// Logical names are the only user-facing identifier. They must be mapped
		// through the repo package so every command applies the same path safety
		// rules and naming contract.
		path, err := repo.LogicalNameToPath(root, logical, cfg.Repository.EncryptedExtension)
		if err != nil {
			return nil, err
		}
		plaintext, err := sopsutil.DecryptFile(cfg.SOPS.Binary, path)
		if err != nil {
			return nil, err
		}
		doc, _, err := schema.ParseYAMLDocument(plaintext)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", logical, err)
		}
		if err := doc.Validate(opts); err != nil {
			return nil, fmt.Errorf("validate %s: %w", logical, err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
