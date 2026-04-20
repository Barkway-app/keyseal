package doctor

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var placeholderRecipientPattern = regexp.MustCompile(`\bage1[A-Za-z0-9_]*REPLACE_ME[A-Za-z0-9_]*\b|\bREPLACE_ME\b`)

// sopsConfigInfo captures only the small subset of .sops.yaml details the
// doctor needs in order to catch obvious setup mistakes.
type sopsConfigInfo struct {
	CreationRuleCount int
	UsableRuleCount   int
	Placeholders      []string
}

func inspectSOPSConfig(data []byte) (sopsConfigInfo, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return sopsConfigInfo{}, fmt.Errorf("parse yaml: %w", err)
	}
	root := doctorRootMapping(&node)
	if root == nil || root.Kind != yaml.MappingNode {
		return sopsConfigInfo{}, fmt.Errorf("expected a mapping document")
	}

	info := sopsConfigInfo{
		Placeholders: findPlaceholderRecipients(string(data)),
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value != "creation_rules" {
			continue
		}
		value := root.Content[i+1]
		if value.Kind != yaml.SequenceNode {
			return sopsConfigInfo{}, fmt.Errorf("creation_rules must be a sequence")
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

func doctorRootMapping(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

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
