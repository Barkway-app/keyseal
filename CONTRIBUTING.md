# Contributing

Thanks for your interest in Keyseal.

## Development workflow

Requirements:
- Go 1.27+
- `sops` installed locally if you want to exercise real edit/decrypt flows

`make check` also runs `go test -race`, which needs a C toolchain (`gcc`); on
systems without one, use `make fmt-check vet test build` or `make test` as the
non-race fast path.

Common commands:

```bash
make tidy
make fmt
make check
make build
```

## Scope

Keyseal is intentionally narrow:
- CLI-first
- repo-backed
- SOPS-driven

Please avoid broad feature proposals that turn it into a hosted platform, server, daemon, or custom crypto system.

## Pull requests

Before opening a PR:
- keep changes within the current product direction
- add or update targeted tests when behavior changes
- update README/help text if user-facing behavior changes
- prefer smaller, reviewable patches over broad refactors

## Reporting issues

Bug reports are most helpful when they include:
- the command you ran
- the observed behavior
- the expected behavior
- your OS and Go version
- whether `sops` was installed and available in `PATH`
