![Keyseal logo](https://github.com/Barkway-app/keyseal/wiki/keyseal-logo.png)

# Keyseal

[![CI](https://github.com/Barkway-app/keyseal/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Barkway-app/keyseal/actions/workflows/ci.yml)

> **Not production ready** — this project is under active development and not yet suitable for production use.
>
> **Default flow:** `keyseal add` scaffolds and encrypts a new secret document immediately, without writing plaintext starter content to the final `.enc.yaml` path.

Keyseal is a small Go CLI that standardizes encrypted file workflows around `sops`, `age`, and Git.

It helps teams:
- keep encrypted secret files in a repository
- scaffold new secret documents
- open encrypted files in `sops`
- render decrypted values into runtime formats
- run commands with decrypted environment variables injected
- validate repository layout and config health

Keyseal does not implement encryption itself, replace SOPS, run a server, expose an API, or behave like a hosted secrets platform.

Keyseal shells out to the installed `sops` binary for editing and decryption. It does not vendor, embed, or replace SOPS.

## Why it exists

SOPS already solves encryption and editing well. Keyseal stays one layer above that and makes a repo-backed workflow more predictable:
- consistent file naming and layout
- small config with strong defaults
- repeatable render and exec flows
- clear validation for common mistakes

## Requirements

- Go 1.22+ to build the CLI
- `sops` installed locally and available in `PATH`
- age recipients configured in `.sops.yaml`

## Quick start

```bash
make build
./bin/keyseal init
./bin/keyseal add production/platform/app --template laravel
./bin/keyseal edit production/platform/app
./bin/keyseal render production/platform/app --stdout
./bin/keyseal doctor
```

## Build

```bash
make tidy
make check
make build
./bin/keyseal --version
./bin/keyseal --help
```

## Command overview

- `keyseal init` bootstraps a repository layout, `keyseal.yaml`, and `.sops.yaml`
- `keyseal add <logical-name>` creates and encrypts a starter env secret document at the final `.enc.yaml` path
- `keyseal edit <logical-name>` opens the target file with `sops`
- `keyseal render <logical-name...>` decrypts, merges, and renders secret values
- `keyseal exec <logical-name...> -- <command...>` runs a child process with merged env vars
- `keyseal doctor` validates config sanity, `.sops.yaml` readiness, placeholder recipients, SOPS availability, plaintext mistakes, and decrypted document shape
- `keyseal version` reports version, commit, and build date metadata

## What Keyseal is not

- not a crypto implementation
- not a Vault replacement
- not a secret hosting service
- not a daemon or web UI
- not a Kubernetes controller

## Default repository structure

```text
keyseal.yaml
.sops.yaml

production/
  platform/
  infra/
  tenants/

staging/
  platform/
  infra/
  tenants/
```

## `keyseal.yaml`

```yaml
version: 1

repository:
  root: .
  encrypted_extension: .enc.yaml

sops:
  binary: sops
  age_key_file: ~/.config/sops/age/keys.txt

defaults:
  output_format: dotenv
  output_dir: /run/secrets
  file_mode: "0600"

validation:
  require_values: true
  key_pattern: '^[A-Z0-9_]+$'

profiles:
  default:
    renders: []
```

## Secret document schema

Keyseal v1 RC supports one secret kind: `kind: env`.

```yaml
version: 1
kind: env
name: production/platform/app
description: Core Laravel app secrets for production
values:
  APP_NAME: Barkway
  APP_ENV: production
  APP_KEY: base64:xxxxx
  APP_DEBUG: "false"
  DB_HOST: 10.0.0.10
  DB_PORT: "3306"
  DB_DATABASE: barkway
  DB_USERNAME: barkway_app
  DB_PASSWORD: super-secret
```

Rules:
- `version`, `kind`, `name`, and `values` are required
- env keys must match `^[A-Z0-9_]+$`
- encrypted files always end with `.enc.yaml`

## `.sops.yaml`

Keyseal generates a starter file with placeholder recipients you must replace:

```yaml
creation_rules:
  - path_regex: production/.*\.enc\.yaml$
    age: age1REPLACE_ME,age1RECOVERY_REPLACE_ME

  - path_regex: staging/.*\.enc\.yaml$
    age: age1REPLACE_ME,age1RECOVERY_REPLACE_ME
```

## Usage

### Initialize a repo

```bash
keyseal init
keyseal init --dry-run
keyseal init --force
```

### Add a secret document

```bash
keyseal add production/platform/app --template laravel
```

This recommended flow is non-interactive: Keyseal writes starter YAML to a secure temp file, runs `sops encrypt`, and atomically writes only encrypted output to `production/platform/app.enc.yaml`.

### Edit with SOPS

```bash
keyseal edit production/platform/app
```

Keyseal does not reimplement SOPS. It resolves the logical name to a file path and runs `sops <file>`.

### Render decrypted output

Render to stdout:

```bash
keyseal render production/platform/app --format dotenv --stdout
```

`render` requires exactly one of `--stdout` or `--out`. If you use `--stdout`, decrypted values are printed directly to the terminal.

Render multiple files to a target file with later files overriding earlier files:

```bash
keyseal render staging/platform/app staging/platform/stripe \
  --format json \
  --out ./runtime/app-secrets.json
```

### Execute a command with injected env vars

```bash
keyseal exec production/platform/app production/platform/stripe -- php artisan queue:work
```

The current process environment is inherited first, then secret values override matching keys for the child process only.

### Run diagnostics

```bash
keyseal doctor
keyseal doctor --json
```

Checks include:
- `keyseal.yaml` exists, parses, and has sane values
- `.sops.yaml` exists, parses, and contains usable creation rules
- placeholder SOPS recipients like `age1REPLACE_ME` are flagged
- the configured `sops` binary can be resolved and executed
- the detected SOPS version is reported when available
- encrypted file naming still matches the expected logical mapping
- plaintext starter files left behind by older Keyseal versions are flagged until they are encrypted with `keyseal edit`
- decrypted documents validate against the env schema when the configured `sops` binary is available
- duplicate env keys are surfaced
- unsafe file modes, risky output paths, and obvious generated repo artifacts are surfaced

`doctor --json` emits the same checks in a structured JSON format for CI or scripts.

## Version reporting

```bash
keyseal --version
keyseal version
keyseal version --short
```

Release builds stamp version, commit, and build date metadata via Go linker flags. Local builds default to `dev`, `unknown`, and `unknown` unless you pass `VERSION`, `COMMIT`, and `DATE`.
When Git metadata is available, local builds default to the latest `v*` tag and the short current commit, so `keyseal --version` reports output like `keyseal v0.1.0 (abc1234)`.

## Release artifacts

```bash
make dist VERSION=v0.2.0 COMMIT=$(git rev-parse --short HEAD) DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

`make dist` builds tar.gz archives for the supported release platforms, includes `README.md` and `LICENSE` in each archive, and writes a single SHA256 checksums file to `dist/`. Tagged `v*` pushes publish those artifacts as GitHub Releases.

## Templates

Built-in starter templates:
- `laravel`
- `stripe`
- `mail`
- `mysql-app`

These templates are intentionally small and hardcoded for v1 RC.

## Development

```bash
make fmt
make check
make test
make build
```

## CI

GitHub Actions runs:
- `gofmt -l` verification
- `go test ./...`
- `go build ./cmd/keyseal`
- tagged `v*` release builds that publish archives and checksums

## Built by Barkway

Keyseal is built by the team at Barkway.

We created it to solve a practical infrastructure problem in our own stack: managing encrypted secret files safely and predictably with `sops`, `age`, and Git.

Learn more about Barkway and our open-source work: https://www.barkway.app/open-source

## Licensing

This package is licensed under `GPL-3.0-only` (GNU General Public License v3.0 only).

If you distribute this package, modifications, or derivative works, review the GPLv3 obligations first to make sure your usage and distribution model remain compliant.

See [LICENSE](./LICENSE) for the full license text.
