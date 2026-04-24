# Changelog

All notable changes to Keyseal will be documented in this file.

The format is intentionally lightweight for now and optimized for early public release tags.

## v0.3.0

Tightens up Keyseal's day-to-day secret handling, especially around empty placeholders, SOPS checks, and keeping encrypted files in sync when recipients change.

Highlights:
- SOPS-backed commands now preflight the configured SOPS binary before decrypting or mutating files, and `keyseal doctor` now reports SOPS and age tool availability near the top of its output
- `keyseal updatekeys` now batch-syncs SOPS recipients for Keyseal-managed encrypted files from `.sops.yaml`, with placeholder/plaintext safety checks and optional explicit commits
- empty or whitespace-only `.enc.yaml` files are now treated as placeholder secrets so `render` and `exec` skip them when other requested secrets are usable, and fail clearly when every requested secret is still uninitialized
- `keyseal edit` now bootstraps placeholder secret files with an encrypted starter document before opening SOPS, making recovery from empty placeholder files a first-class workflow
- `keyseal doctor` now warns on empty placeholder secret files, continues to fail on non-empty plaintext content at encrypted paths, and includes regression coverage for the new secret-file classification behavior

## v0.2.0

Highlights:
- `keyseal add` now scaffolds via a secure temp file, runs `sops encrypt`, and atomically writes only encrypted output to the final `.enc.yaml` path by default
- `keyseal add` no longer exposes a plaintext scaffolding mode
- Keyseal now has first-class Git workflow support with `status`, `diff`, `history`, `commit`, and Git-based `rollback`
- `status` can now be scoped to one logical secret and `history` now supports a compact `--oneline` view
- `add`, `edit`, and `rollback` now support `--commit`, `-m/--message`, and config-driven `git.auto_commit`
- `sops.age_key_file` in `keyseal.yaml` is now passed through automatically to SOPS-backed commands, with `SOPS_AGE_KEY_FILE` remaining the higher-precedence override
- `keyseal doctor` now reports structured, actionable checks for config sanity, `.sops.yaml` readiness, placeholder recipients, SOPS availability, plaintext mistakes, decrypted schema validation, and JSON output for scripts
- `keyseal --version` and `keyseal version` now report stamped build metadata
- tagged `v*` GitHub releases now publish platform archives and SHA256 checksums
- Linux `.deb` and `.rpm` packaging now uses generated per-arch nfpm configs so release packaging does not depend on fragile env interpolation

## v0.1.0

Initial public release.

Highlights:
- Cobra-based CLI with `init`, `add`, `edit`, `render`, `exec`, and `doctor`
- SOPS subprocess integration for edit and decrypt flows
- deterministic logical-name to `.enc.yaml` mapping
- dotenv, JSON, and YAML render outputs
- doctor checks for config health, repo naming, schema validation, and plaintext starter-file detection
- GPL-3.0-only licensing

Known limitations:
- `keyseal add` creates a plaintext starter document until you encrypt it with `keyseal edit`
- only `kind: env` is supported
- rendering is limited to dotenv, JSON, and YAML
