// Package templates contains the built-in starter documents for `keyseal add`.
package templates

import (
	"fmt"
	"sort"

	"github.com/Barkway-app/keyseal/internal/schema"
)

var supported = map[string]map[string]string{
	"laravel": {
		"APP_NAME":         "ExampleApp",
		"APP_ENV":          "production",
		"APP_KEY":          "base64:REPLACE_ME",
		"APP_DEBUG":        "false",
		"APP_URL":          "https://example.com",
		"DB_HOST":          "127.0.0.1",
		"DB_PORT":          "3306",
		"DB_DATABASE":      "app",
		"DB_USERNAME":      "app",
		"DB_PASSWORD":      "REPLACE_ME",
		"CACHE_STORE":      "redis",
		"QUEUE_CONNECTION": "redis",
		"SESSION_DRIVER":   "redis",
	},
	"stripe": {
		"STRIPE_SECRET_KEY":        "sk_live_REPLACE_ME",
		"STRIPE_WEBHOOK_SECRET":    "whsec_REPLACE_ME",
		"STRIPE_CONNECT_CLIENT_ID": "ca_REPLACE_ME",
	},
	"mail": {
		"MAIL_MAILER":       "smtp",
		"MAIL_HOST":         "smtp.example.com",
		"MAIL_PORT":         "587",
		"MAIL_USERNAME":     "mailer",
		"MAIL_PASSWORD":     "REPLACE_ME",
		"MAIL_FROM_ADDRESS": "noreply@example.com",
	},
	"mysql-app": {
		"DB_HOST":     "127.0.0.1",
		"DB_PORT":     "3306",
		"DB_DATABASE": "app",
		"DB_USERNAME": "app",
		"DB_PASSWORD": "REPLACE_ME",
	},
}

// SupportedNames returns the stable list of built-in template names.
func SupportedNames() []string {
	names := make([]string, 0, len(supported))
	for name := range supported {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Build returns a starter env document for the requested logical name.
func Build(logicalName, templateName string) (schema.EnvSecretDocument, error) {
	doc := schema.EnvSecretDocument{
		Version:     schema.SupportedVersion,
		Kind:        "env",
		Name:        logicalName,
		Description: fmt.Sprintf("Secrets for %s", logicalName),
		Values:      map[string]string{},
	}
	if templateName == "" {
		doc.Values["EXAMPLE_KEY"] = "REPLACE_ME"
		return doc, nil
	}
	values, ok := supported[templateName]
	if !ok {
		return schema.EnvSecretDocument{}, fmt.Errorf("unknown template %q", templateName)
	}
	for key, value := range values {
		doc.Values[key] = value
	}
	return doc, nil
}
