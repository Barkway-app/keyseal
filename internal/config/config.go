// Package config loads and validates the small keyseal.yaml configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	DefaultConfigPath      = "keyseal.yaml"
	DefaultEncryptedExt    = ".enc.yaml"
	DefaultSOPSBinary      = "sops"
	DefaultAgeBinary       = "age"
	DefaultAgeKeyFile      = "~/.config/sops/age/keys.txt"
	DefaultOutputFormat    = "dotenv"
	DefaultOutputDir       = "/run/secrets"
	DefaultFileMode        = "0600"
	DefaultRequireValues   = true
	DefaultEnvKeyPattern   = "^[A-Z0-9_]+$"
	SupportedConfigVersion = 1
)

var allowedOutputFormats = map[string]struct{}{
	"dotenv": {},
	"json":   {},
	"yaml":   {},
}

// Config is the root representation of keyseal.yaml.
type Config struct {
	Version    int                `yaml:"version"`
	Repository RepositoryConfig   `yaml:"repository"`
	SOPS       SOPSConfig         `yaml:"sops"`
	Git        GitConfig          `yaml:"git"`
	Defaults   DefaultsConfig     `yaml:"defaults"`
	Validation ValidationConfig   `yaml:"validation"`
	Profiles   map[string]Profile `yaml:"profiles"`
}

// RepositoryConfig controls repository-root and encrypted file naming behavior.
type RepositoryConfig struct {
	Root               string `yaml:"root"`
	EncryptedExtension string `yaml:"encrypted_extension"`
}

// SOPSConfig controls how Keyseal locates and invokes the external sops binary.
type SOPSConfig struct {
	Binary     string `yaml:"binary"`
	AgeBinary  string `yaml:"age_binary"`
	AgeKeyFile string `yaml:"age_key_file"`
}

// GitConfig controls small Git workflow defaults used by mutating commands.
type GitConfig struct {
	AutoCommit bool `yaml:"auto_commit"`
}

// DefaultsConfig contains render-time defaults used when flags are omitted.
type DefaultsConfig struct {
	OutputFormat string `yaml:"output_format"`
	OutputDir    string `yaml:"output_dir"`
	FileMode     string `yaml:"file_mode"`
}

// ValidationConfig controls schema checks applied to decrypted env documents.
type ValidationConfig struct {
	RequireValues bool   `yaml:"require_values"`
	KeyPattern    string `yaml:"key_pattern"`
}

// Profile is reserved for small grouped render definitions in future versions.
type Profile struct {
	Renders []RenderProfile `yaml:"renders"`
}

// RenderProfile describes a named render entry in configuration.
type RenderProfile struct {
	Name   string   `yaml:"name,omitempty"`
	Inputs []string `yaml:"inputs,omitempty"`
	Format string   `yaml:"format,omitempty"`
	Out    string   `yaml:"out,omitempty"`
	Mode   string   `yaml:"mode,omitempty"`
}

// Default returns the built-in configuration defaults for a new repository.
func Default() Config {
	return Config{
		Version: SupportedConfigVersion,
		Repository: RepositoryConfig{
			Root:               ".",
			EncryptedExtension: DefaultEncryptedExt,
		},
		SOPS: SOPSConfig{
			Binary:     DefaultSOPSBinary,
			AgeBinary:  DefaultAgeBinary,
			AgeKeyFile: DefaultAgeKeyFile,
		},
		Git: GitConfig{
			AutoCommit: false,
		},
		Defaults: DefaultsConfig{
			OutputFormat: DefaultOutputFormat,
			OutputDir:    DefaultOutputDir,
			FileMode:     DefaultFileMode,
		},
		Validation: ValidationConfig{
			RequireValues: DefaultRequireValues,
			KeyPattern:    DefaultEnvKeyPattern,
		},
		Profiles: map[string]Profile{
			"default": {Renders: []RenderProfile{}},
		},
	}
}

// Load reads keyseal.yaml, applies defaults for omitted optional fields, and
// validates the final configuration.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := yamlUnmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	def := Default()

	if c.Version == 0 {
		c.Version = def.Version
	}
	if c.Repository.Root == "" {
		c.Repository.Root = def.Repository.Root
	}
	if c.Repository.EncryptedExtension == "" {
		c.Repository.EncryptedExtension = def.Repository.EncryptedExtension
	}
	if c.SOPS.Binary == "" {
		c.SOPS.Binary = def.SOPS.Binary
	}
	if c.SOPS.AgeBinary == "" {
		c.SOPS.AgeBinary = def.SOPS.AgeBinary
	}
	if c.SOPS.AgeKeyFile == "" {
		c.SOPS.AgeKeyFile = def.SOPS.AgeKeyFile
	}
	if c.Defaults.OutputFormat == "" {
		c.Defaults.OutputFormat = def.Defaults.OutputFormat
	}
	if c.Defaults.OutputDir == "" {
		c.Defaults.OutputDir = def.Defaults.OutputDir
	}
	if c.Defaults.FileMode == "" {
		c.Defaults.FileMode = def.Defaults.FileMode
	}
	if c.Validation.KeyPattern == "" {
		c.Validation.KeyPattern = def.Validation.KeyPattern
	}
	if c.Profiles == nil {
		c.Profiles = def.Profiles
	}
	if _, ok := c.Profiles["default"]; !ok {
		c.Profiles["default"] = Profile{Renders: []RenderProfile{}}
	}
}

// Validate checks the semantic constraints of a loaded config file.
func (c Config) Validate() error {
	if c.Version != SupportedConfigVersion {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Repository.Root == "" {
		return errors.New("repository.root is required")
	}
	if !strings.HasPrefix(c.Repository.EncryptedExtension, ".") || !strings.Contains(c.Repository.EncryptedExtension, ".yaml") {
		return fmt.Errorf("repository.encrypted_extension must look like %q", DefaultEncryptedExt)
	}
	if c.SOPS.Binary == "" {
		return errors.New("sops.binary is required")
	}
	if c.SOPS.AgeBinary == "" {
		return errors.New("sops.age_binary is required")
	}
	if _, ok := allowedOutputFormats[c.Defaults.OutputFormat]; !ok {
		return fmt.Errorf("defaults.output_format must be one of dotenv, json, yaml")
	}
	if _, err := parseFileMode(c.Defaults.FileMode); err != nil {
		return fmt.Errorf("defaults.file_mode: %w", err)
	}
	if c.Validation.KeyPattern == "" {
		return errors.New("validation.key_pattern is required")
	}
	if _, err := regexp.Compile(c.Validation.KeyPattern); err != nil {
		return fmt.Errorf("validation.key_pattern: %w", err)
	}
	return nil
}

// RepoRoot resolves the repository root relative to the current working tree.
func (c Config) RepoRoot(base string) string {
	if filepath.IsAbs(c.Repository.Root) {
		return c.Repository.Root
	}
	return filepath.Clean(filepath.Join(base, c.Repository.Root))
}

// ExpandUser expands a leading "~" in a path when a home directory is known.
func ExpandUser(path string) string {
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

// ResolvePath expands "~" and resolves relative paths from the provided base.
func ResolvePath(base, path string) string {
	path = ExpandUser(path)
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}
