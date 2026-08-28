package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jrpbuilds/keyseal/internal/config"
)

const profilesTestBaseConfig = `version: 1
repository:
  root: .
  encrypted_extension: .enc.yaml
sops:
  binary: sops
  age_binary: age
  age_key_file: ~/.config/sops/age/keys.txt
defaults:
  output_format: dotenv
  output_dir: /run/secrets
  file_mode: "0600"
validation:
  require_values: true
  key_pattern: "^[A-Z0-9_]+$"
`

func writeProfilesTestConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "keyseal.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// TestConfigValidateProfiles covers each profile rejection with its specific
// error text, plus the passing cases required by the render-profiles spec.
func TestConfigValidateProfiles(t *testing.T) {
	tests := []struct {
		name     string
		profiles string
		wantErr  string
	}{
		{
			name: "valid multi-render profile passes",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        format: dotenv
        out: /run/secrets/app.env
        mode: "0600"
      - name: stripe
        inputs:
          - production/platform/stripe
        format: json
        out: /run/secrets/stripe.json
        mode: "0600"
`,
		},
		{
			name: "no profiles section passes",
		},
		{
			name: "default-only profile passes",
			profiles: `profiles:
  default:
    renders: []
`,
		},
		{
			name: "empty render name rejected",
			profiles: `profiles:
  prod:
    renders:
      - inputs:
          - production/platform/app
        out: /run/secrets/app.env
`,
			wantErr: `render name is required`,
		},
		{
			name: "empty inputs rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs: []
        out: /run/secrets/app.env
`,
			wantErr: `inputs must list at least one logical secret name`,
		},
		{
			name: "invalid logical-name shape rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - ../escape
        out: /run/secrets/app.env
`,
			wantErr: `must not traverse directories`,
		},
		{
			name: "unknown explicit format rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        format: xml
        out: /run/secrets/app.env
`,
			wantErr: `unknown format "xml"`,
		},
		{
			name: "malformed mode rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        mode: "999"
        out: /run/secrets/app.env
`,
			wantErr: `invalid mode "999"`,
		},
		{
			name: "empty out rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        out: ""
`,
			wantErr: `out is required`,
		},
		{
			name: "duplicate out after normalization rejected",
			profiles: `profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        out: /run/secrets/a//app.env
      - name: stripe
        inputs:
          - production/platform/stripe
        out: /run/secrets/./a/app.env
`,
			wantErr: `both write output`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeProfilesTestConfig(t, profilesTestBaseConfig+"\n"+tt.profiles)
			cfg, err := config.Load(path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Load returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected Load to fail, got cfg %#v", cfg)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error to contain %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// TestConfigValidateProfilesValidMultiRenderRetained verifies a valid profile
// keeps both renders for execution.
func TestConfigValidateProfilesValidMultiRenderRetained(t *testing.T) {
	path := writeProfilesTestConfig(t, profilesTestBaseConfig+`
profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        out: /run/secrets/app.env
      - name: stripe
        inputs:
          - production/platform/stripe
        format: json
        out: /run/secrets/stripe.json
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := len(cfg.Profiles["prod"].Renders); got != 2 {
		t.Fatalf("expected 2 retained renders, got %d", got)
	}
}

// TestConfigValidateProfilesDefaultFallback verifies empty format/mode fields
// fall back to Defaults for validation without being written back, and that an
// invalid fallback value is still rejected.
func TestConfigValidateProfilesDefaultFallback(t *testing.T) {
	t.Run("empty fields fall back to valid defaults", func(t *testing.T) {
		path := writeProfilesTestConfig(t, strings.Replace(profilesTestBaseConfig,
			`output_format: dotenv`, `output_format: json`, 1)+`
profiles:
  prod:
    renders:
      - name: app
        inputs:
          - production/platform/app
        out: /run/secrets/app.env
`)
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		// Fallback must not be written back to the struct.
		render := cfg.Profiles["prod"].Renders[0]
		if render.Format != "" || render.Mode != "" {
			t.Fatalf("expected empty format/mode to stay empty in the struct, got %q/%q", render.Format, render.Mode)
		}
	})

	t.Run("invalid fallback format rejected", func(t *testing.T) {
		cfg := config.Default()
		cfg.Defaults.OutputFormat = ""
		cfg.Profiles["prod"] = config.Profile{Renders: []config.RenderProfile{
			{Name: "app", Inputs: []string{"production/platform/app"}, Out: "/run/secrets/app.env"},
		}}
		err := cfg.ValidateProfiles()
		if err == nil {
			t.Fatal("expected unknown fallback format to be rejected")
		}
		if !strings.Contains(err.Error(), `unknown format ""`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid fallback mode rejected", func(t *testing.T) {
		cfg := config.Default()
		cfg.Defaults.FileMode = ""
		cfg.Profiles["prod"] = config.Profile{Renders: []config.RenderProfile{
			{Name: "app", Inputs: []string{"production/platform/app"}, Out: "/run/secrets/app.env"},
		}}
		err := cfg.ValidateProfiles()
		if err == nil {
			t.Fatal("expected invalid fallback mode to be rejected")
		}
		if !strings.Contains(err.Error(), `invalid mode ""`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestLoadLenientSkipsProfileValidation verifies doctor's lenient load keeps
// malformed profiles enumerable while strict Load rejects them.
func TestLoadLenientSkipsProfileValidation(t *testing.T) {
	body := profilesTestBaseConfig + `
profiles:
  prod:
    renders:
      - name: app
        inputs: []
        out: /run/secrets/app.env
`
	path := writeProfilesTestConfig(t, body)

	if _, err := config.Load(path); err == nil {
		t.Fatal("expected strict Load to reject the malformed profile")
	}

	cfg, err := config.LoadLenient(path)
	if err != nil {
		t.Fatalf("LoadLenient returned error: %v", err)
	}
	if got := len(cfg.Profiles["prod"].Renders); got != 1 {
		t.Fatalf("expected the malformed render to remain enumerable, got %d renders", got)
	}
}

// TestLoadLenientStillValidatesBase verifies the lenient path keeps enforcing
// non-profile validation.
func TestLoadLenientStillValidatesBase(t *testing.T) {
	path := writeProfilesTestConfig(t, strings.Replace(profilesTestBaseConfig,
		`file_mode: "0600"`, `file_mode: not-a-mode`, 1))
	if _, err := config.LoadLenient(path); err == nil {
		t.Fatal("expected LoadLenient to reject an invalid defaults.file_mode")
	}
}
