![Keyseal logo](https://github.com/Barkway-app/keyseal/wiki/keyseal-logo.png)

# Keyseal

[![CI](https://github.com/Barkway-app/keyseal/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Barkway-app/keyseal/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/Barkway-app/keyseal)](https://github.com/Barkway-app/keyseal/releases)
[![License: GPL-3.0-only](https://img.shields.io/badge/license-GPL--3.0--only-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8)](https://go.dev/)

> **Default flow:** `keyseal add` scaffolds and encrypts a new secret document immediately, without writing plaintext starter content to the final `.enc.yaml` path.

Keyseal is a small Go CLI that standardizes encrypted file workflows around `sops`, `age`, and Git. Barkway uses it in production for Git-backed SOPS secret workflows.

It helps teams:
- keep encrypted secret files in a repository
- scaffold new secret documents
- open encrypted files in `sops`
- inspect Git status, diffs, and file history by logical name
- commit and roll back Keyseal-managed changes without staging the whole repo
- render decrypted values into runtime formats
- run commands with decrypted environment variables injected
- validate repository layout and config health
- enforce strict CI verification before release or deployment

Keyseal does not implement encryption itself, replace SOPS, run a server, expose an API, or behave like a hosted secrets platform.

Keyseal shells out to the installed `sops` binary for editing and decryption. It does not vendor, embed, or replace SOPS.

## Why it exists

SOPS already solves encryption and editing well. Keyseal stays one layer above that and makes a repo-backed workflow more predictable:
- consistent file naming and layout
- Git-aware workflows around the encrypted files that already live in the repo
- small config with strong defaults
- repeatable render and exec flows
- clear validation for common mistakes

## Requirements

- Go 1.22+ to build the CLI
- `sops` installed locally and available in `PATH`
- age recipients configured in `.sops.yaml`

## Installation

Pre-built binaries and Linux packages are available on the [GitHub Releases](https://github.com/Barkway-app/keyseal/releases) page.

Each release includes:
- tar.gz archives for Linux (amd64, arm64) and macOS (amd64, arm64)
- `.deb` packages for Linux amd64 and arm64
- `.rpm` packages for Linux amd64 and arm64
- SHA256 checksums for all artifacts

**Linux (Debian/Ubuntu):**
```bash
sudo dpkg -i keyseal_<version>_amd64.deb
```

**Linux (RHEL/Fedora/SUSE):**
```bash
sudo rpm -i keyseal-<version>-1.x86_64.rpm
```

**Other platforms:** Extract the binary from the appropriate tar.gz archive and place it somewhere in your `PATH`.

Verify downloads against the `checksums.txt` file included in each release.

## Quick start

```bash
make build
./bin/keyseal init
./bin/keyseal add production/platform/app --template laravel
./bin/keyseal edit production/platform/app
./bin/keyseal status
./bin/keyseal commit -m "Add production app secret"
./bin/keyseal render production/platform/app --stdout
./bin/keyseal doctor
```

For a fuller walkthrough, see the wiki:
- [Quick Start](https://github.com/Barkway-app/keyseal/wiki/Quick-Start)
- [Concepts](https://github.com/Barkway-app/keyseal/wiki/Concepts)
- [Command Reference](https://github.com/Barkway-app/keyseal/wiki/Command-Reference)
- [Configuration Reference](https://github.com/Barkway-app/keyseal/wiki/Configuration-Reference)

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
- `keyseal add <logical-name>` creates and encrypts a starter env secret document at the final `.enc.yaml` path, with optional immediate Git commit support
- `keyseal edit <logical-name>` opens the target file with `sops`, bootstrapping empty placeholder files first when needed
- `keyseal updatekeys [logical-name...]` syncs SOPS recipients from `.sops.yaml` for encrypted files, with optional explicit Git commit support
- `keyseal status [logical-name]` shows Git status for Keyseal-managed files, optionally narrowed to one secret
- `keyseal diff <logical-name>` shows `git diff` for one secret file
- `keyseal history <logical-name>` shows file-scoped Git history for one secret file, with optional `--oneline` output
- `keyseal commit` stages current Keyseal-managed changes and creates a Git commit
- `keyseal rollback <logical-name> --to <commit>` restores one secret file from Git history
- `keyseal render <logical-name...>` decrypts, merges, and renders secret values, skipping empty placeholder files
- `keyseal exec <logical-name...> -- <command...>` runs a child process with merged env vars, skipping empty placeholder files
- `keyseal doctor` validates config sanity, SOPS availability, age availability warnings, `.sops.yaml` readiness, placeholder recipients, plaintext mistakes, empty placeholders, and decrypted document shape
- `keyseal verify` runs strict CI checks and fails on any doctor warning or failure
- `keyseal version` reports version, commit, and build date metadata

Detailed flags, examples, and behavior notes live in the wiki:
- [Command Reference](https://github.com/Barkway-app/keyseal/wiki/Command-Reference)
- [Troubleshooting](https://github.com/Barkway-app/keyseal/wiki/Troubleshooting)

## What Keyseal is not

- not a crypto implementation
- not a Vault replacement
- not a secret hosting service
- not a daemon or web UI
- not a Kubernetes controller

## Example config

```yaml
version: 1

repository:
  root: .
  encrypted_extension: .enc.yaml

sops:
  binary: sops
  age_binary: age
  age_key_file: ~/.config/sops/age/keys.txt

git:
  auto_commit: false

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

For full schema and config details, see:
- [Configuration Reference](https://github.com/Barkway-app/keyseal/wiki/Configuration-Reference)
- [Secret File Format](https://github.com/Barkway-app/keyseal/wiki/Secret-File-Format)
- [Repository Layout](https://github.com/Barkway-app/keyseal/wiki/Repository-Layout)

## Usage

```bash
keyseal add production/platform/app --template laravel
keyseal edit production/platform/app
keyseal updatekeys production/platform/app --yes
keyseal status production/platform/app
keyseal history production/platform/app --oneline
keyseal commit -m "Update production app secret"
keyseal render production/platform/app --stdout --format json
keyseal exec production/platform/app -- php artisan migrate
keyseal rollback production/platform/app --to <commit> --dry-run
keyseal doctor
keyseal verify
```

Key workflow details:
- `-m, --message` implies commit on mutating commands
- `git.auto_commit` is off by default
- `sops.age_key_file` is used as the default age key path unless `SOPS_AGE_KEY_FILE` is already set
- SOPS-backed commands check the configured `sops.binary` before decrypting or mutating secret files
- `updatekeys` uses `.sops.yaml` as the recipient source of truth and does not rotate secret values or data encryption keys
- `keyseal commit` stages only Keyseal-managed files, not the whole repo
- `rollback` restores the encrypted file from Git history, while `--dry-run` previews safely without modifying the working tree

For exact command behavior and more examples, see:
- [Command Reference](https://github.com/Barkway-app/keyseal/wiki/Command-Reference)
- [Configuration Reference](https://github.com/Barkway-app/keyseal/wiki/Configuration-Reference)
- [Troubleshooting](https://github.com/Barkway-app/keyseal/wiki/Troubleshooting)

## Version reporting

```bash
keyseal --version
keyseal version
keyseal version --short
```

Release builds stamp version, commit, and build date metadata via Go linker flags. Local builds default to `dev`, `unknown`, and `unknown` unless you pass `VERSION`, `COMMIT`, and `DATE`.
When Git metadata is available, local builds default to the latest `v*` tag and the short current commit, so `keyseal --version` reports output like `keyseal v1.0.0 (abc1234)`.

## Release artifacts

```bash
make dist VERSION=v1.0.0 COMMIT=$(git rev-parse --short HEAD) DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

`make dist` builds tar.gz archives for the supported release platforms, generates `.deb` and `.rpm` packages for Linux (amd64, arm64), and writes a single SHA256 checksums file covering all artifacts to `dist/`.

Linux packages require [nfpm](https://nfpm.goreleaser.com/install/) to be installed. To build only the Linux packages without archives:

```bash
make packages VERSION=v1.0.0
```

Tagged `v*` pushes publish all artifacts — archives, packages, and checksums — as GitHub Releases.

## Development

```bash
make fmt
make check
make test
make build
```

For contributor-oriented detail, see:
- [Contributing](https://github.com/Barkway-app/keyseal/wiki/Contributing)
- [Build and Release](https://github.com/Barkway-app/keyseal/wiki/Build-and-Release)
- [Templates](https://github.com/Barkway-app/keyseal/wiki/Templates)

## CI

GitHub Actions runs:
- `gofmt -l` verification
- `go test ./...`
- `go build ./cmd/keyseal`
- tagged `v*` release builds that publish tar.gz archives, `.deb` packages, `.rpm` packages, and checksums

For repositories managed by Keyseal, use `keyseal verify` as the strict health gate in release and deploy workflows.

## Built by Barkway

Keyseal is built by the team at Barkway.

We created it to solve a practical infrastructure problem in our own stack: managing encrypted secret files safely and predictably with `sops`, `age`, and Git.

Learn more about Barkway and our open-source work: https://www.barkway.app/open-source

## Licensing

This package is licensed under `GPL-3.0-only` (GNU General Public License v3.0 only).

If you distribute this package, modifications, or derivative works, review the GPLv3 obligations first to make sure your usage and distribution model remain compliant.

See [LICENSE](./LICENSE) for the full license text.
