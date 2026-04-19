// Package schema defines and validates the decrypted secret document format.
package schema

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

const SupportedVersion = 1

// EnvSecretDocument is the only supported decrypted secret schema in v1 RC.
type EnvSecretDocument struct {
	Version     int               `yaml:"version" json:"version"`
	Kind        string            `yaml:"kind" json:"kind"`
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Values      map[string]string `yaml:"values" json:"values"`
}

// ValidationOptions controls required-field and env-key validation behavior.
type ValidationOptions struct {
	RequireValues bool
	KeyPattern    string
}

// DefaultValidationOptions returns the default schema validation settings.
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		RequireValues: true,
		KeyPattern:    "^[A-Z0-9_]+$",
	}
}

// Validate checks the required fields and env key constraints for a document.
func (d EnvSecretDocument) Validate(opts ValidationOptions) error {
	if d.Version == 0 {
		return fmt.Errorf("version is required")
	}
	if d.Version != SupportedVersion {
		return fmt.Errorf("unsupported version %d", d.Version)
	}
	if d.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if d.Kind != "env" {
		return fmt.Errorf("unsupported kind %q", d.Kind)
	}
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}
	if opts.RequireValues && len(d.Values) == 0 {
		return fmt.Errorf("values is required for kind env")
	}
	rx, err := regexp.Compile(opts.KeyPattern)
	if err != nil {
		return fmt.Errorf("invalid key pattern: %w", err)
	}
	for key := range d.Values {
		if !rx.MatchString(key) {
			return fmt.Errorf("invalid env key %q", key)
		}
	}
	return nil
}

// ParseYAMLDocument parses a decrypted YAML document and reports duplicate keys
// within the values mapping without losing the last-value-wins behavior of YAML.
func ParseYAMLDocument(data []byte) (EnvSecretDocument, []string, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return EnvSecretDocument{}, nil, fmt.Errorf("parse yaml: %w", err)
	}

	doc, err := decodeDocumentNode(&node)
	if err != nil {
		return EnvSecretDocument{}, nil, err
	}
	return doc, duplicateValueKeys(&node), nil
}

// MarshalYAMLDocument encodes a decrypted env document with stable indentation.
func MarshalYAMLDocument(doc EnvSecretDocument) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HasSOPSMetadata reports whether a YAML document contains the top-level sops
// metadata block that SOPS writes to encrypted files.
func HasSOPSMetadata(data []byte) bool {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return false
	}
	root := rootMapping(&node)
	if root == nil {
		return false
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "sops" {
			return true
		}
	}
	return false
}

func duplicateValueKeys(node *yaml.Node) []string {
	if node == nil {
		return nil
	}
	target := rootMapping(node)
	if target == nil {
		return nil
	}
	for i := 0; i < len(target.Content)-1; i += 2 {
		keyNode := target.Content[i]
		if keyNode.Value != "values" {
			continue
		}
		valueNode := target.Content[i+1]
		if valueNode.Kind != yaml.MappingNode {
			return nil
		}
		seen := map[string]int{}
		dupes := map[string]struct{}{}
		for j := 0; j < len(valueNode.Content)-1; j += 2 {
			k := valueNode.Content[j].Value
			seen[k]++
			if seen[k] > 1 {
				dupes[k] = struct{}{}
			}
		}
		if len(dupes) == 0 {
			return nil
		}
		out := make([]string, 0, len(dupes))
		for k := range dupes {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return nil
}

func rootMapping(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

// decodeDocumentNode decodes only the fields Keyseal cares about so duplicate
// mapping keys can still be detected from the original YAML node tree.
func decodeDocumentNode(node *yaml.Node) (EnvSecretDocument, error) {
	root := rootMapping(node)
	if root == nil || root.Kind != yaml.MappingNode {
		return EnvSecretDocument{}, fmt.Errorf("decode yaml: expected a mapping document")
	}

	doc := EnvSecretDocument{
		Values: map[string]string{},
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		key := root.Content[i].Value
		value := root.Content[i+1]
		switch key {
		case "version":
			version, err := parseIntNode(value)
			if err != nil {
				return EnvSecretDocument{}, fmt.Errorf("decode yaml: version: %w", err)
			}
			doc.Version = version
		case "kind":
			doc.Kind = value.Value
		case "name":
			doc.Name = value.Value
		case "description":
			doc.Description = value.Value
		case "values":
			if value.Kind != yaml.MappingNode {
				return EnvSecretDocument{}, fmt.Errorf("decode yaml: values must be a mapping")
			}
			for j := 0; j < len(value.Content)-1; j += 2 {
				doc.Values[value.Content[j].Value] = value.Content[j+1].Value
			}
		}
	}
	return doc, nil
}

func parseIntNode(node *yaml.Node) (int, error) {
	var value int
	if err := node.Decode(&value); err != nil {
		return 0, err
	}
	return value, nil
}
