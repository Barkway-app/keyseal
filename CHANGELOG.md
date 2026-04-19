# Changelog

All notable changes to Keyseal will be documented in this file.

The format is intentionally lightweight for now and optimized for early public release tags.

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
