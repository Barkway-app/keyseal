# Changelog

All notable changes to Keyseal will be documented in this file.

The format is intentionally lightweight for now and optimized for early public release tags.

## Unreleased

Highlights:
- `keyseal add` now scaffolds via a secure temp file, runs `sops encrypt`, and atomically writes only encrypted output to the final `.enc.yaml` path by default
- `keyseal add` no longer exposes a plaintext scaffolding mode
- `keyseal doctor` now reports structured, actionable checks for config sanity, `.sops.yaml` readiness, placeholder recipients, SOPS availability, plaintext mistakes, decrypted schema validation, and JSON output for scripts
- `keyseal --version` and `keyseal version` now report stamped build metadata
- tagged `v*` GitHub releases now publish platform archives and SHA256 checksums

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
