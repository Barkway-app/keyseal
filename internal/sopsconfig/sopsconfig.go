// Package sopsconfig inspects the small subset of .sops.yaml that Keyseal
// needs before invoking SOPS-backed workflows.
package sopsconfig

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// placeholderRecipientPattern catches the placeholder markers emitted by
// Keyseal's starter .sops.yaml template without trying to validate real keys.
var placeholderRecipientPattern = regexp.MustCompile(`\bage1[A-Za-z0-9_]*REPLACE_ME[A-Za-z0-9_]*\b|\bREPLACE_ME\b`)

// Info captures recipient readiness details from .sops.yaml.
type Info struct {
	// CreationRuleCount is the number of entries under creation_rules.
	CreationRuleCount int
	// UsableRuleCount is the number of creation rules with recipient material.
	UsableRuleCount int
	// Placeholders contains unique placeholder recipient markers found in the file.
	Placeholders []string
}

// Inspect parses .sops.yaml content and reports whether it contains creation
// rules, recipient material, and obvious placeholder recipients.
func Inspect(data []byte) (Info, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return Info{}, fmt.Errorf("parse yaml: %w", err)
	}
	root := rootMapping(&node)
	if root == nil || root.Kind != yaml.MappingNode {
		return Info{}, fmt.Errorf("expected a mapping document")
	}

	info := Info{
		Placeholders: findPlaceholderRecipients(string(data)),
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value != "creation_rules" {
			continue
		}
		value := root.Content[i+1]
		if value.Kind != yaml.SequenceNode {
			return Info{}, fmt.Errorf("creation_rules must be a sequence")
		}
		info.CreationRuleCount = len(value.Content)
		for _, rule := range value.Content {
			if rule.Kind != yaml.MappingNode {
				continue
			}
			if ruleHasUsableRecipients(rule) {
				info.UsableRuleCount++
			}
		}
		return info, nil
	}
	return info, nil
}

// rootMapping unwraps YAML document nodes so callers can inspect the mapping
// content regardless of whether yaml.v3 returned a document wrapper.
func rootMapping(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

// findPlaceholderRecipients returns unique placeholder recipient strings in
// first-seen order so user-facing diagnostics stay stable and readable.
func findPlaceholderRecipients(text string) []string {
	matches := placeholderRecipientPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		match = strings.TrimSpace(match)
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		out = append(out, match)
	}
	return out
}

// ruleHasUsableRecipients reports whether a creation rule contains at least
// one recipient field that SOPS can use for encryption.
func ruleHasUsableRecipients(rule *yaml.Node) bool {
	for i := 0; i < len(rule.Content)-1; i += 2 {
		key := rule.Content[i].Value
		value := rule.Content[i+1]
		switch key {
		case "age", "pgp", "kms", "gcp_kms", "azure_keyvault", "hc_vault_transit_uri":
			if nodeHasContent(value) {
				return true
			}
		case "key_groups":
			if keyGroupsHaveRecipients(value) {
				return true
			}
		}
	}
	return false
}

// keyGroupsHaveRecipients checks nested SOPS key group entries for the same
// recipient fields accepted at the top level of a creation rule.
func keyGroupsHaveRecipients(node *yaml.Node) bool {
	if node.Kind != yaml.SequenceNode {
		return false
	}
	for _, group := range node.Content {
		if group.Kind != yaml.MappingNode {
			continue
		}
		if ruleHasUsableRecipients(group) {
			return true
		}
	}
	return false
}

// nodeHasContent treats non-empty scalars and populated collections as usable
// recipient material while rejecting blank values.
func nodeHasContent(node *yaml.Node) bool {
	switch node.Kind {
	case yaml.ScalarNode:
		return strings.TrimSpace(node.Value) != ""
	case yaml.SequenceNode, yaml.MappingNode:
		return len(node.Content) > 0
	default:
		return false
	}
}
