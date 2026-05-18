# Contributing to mailCli

Thank you for your interest in contributing.

## Development setup

- Go 1.21+ (see `go.mod` for the version used in CI)
- Clone the repo and run from the root:

```bash
go test ./...
go build -o mailcli ./cmd/mailcli
```

## Project layout

- `cmd/mailcli` — CLI entrypoint
- `internal/app` — Cobra commands (protocol-agnostic orchestration)
- `internal/config` — profiles and environment variables
- `internal/domain` — shared types and protocol identifiers
- `internal/protocol/ews` — EWS implementation (search, send, show, HTTP client)
- `internal/timeparse` — shared date parsing for CLI flags

When adding a new protocol (e.g. IMAP/SMTP), implement it under `internal/protocol/<name>` and wire it from `internal/app` via `config.Profile.Protocol`. Prefer extending shared options in `internal/domain` rather than duplicating CLI flags.

## Roadmap

Before large features, check [ROADMAP.md](ROADMAP.md) and open an issue to align scope (especially IMAP/SMTP).

## Pull requests

1. Fork and create a feature branch.
2. Add tests for behavior changes (`go test ./...` must pass).
3. Update [ROADMAP.md](ROADMAP.md) if you complete or reprioritize user-visible capabilities.
4. Update user-facing docs if CLI flags or config schema change.
5. Do not commit real credentials or personal config files.

## Releases

Maintainers: see [docs/RELEASING.md](docs/RELEASING.md) (tag `v*` → GitHub Actions → Releases).

## Code style

- Match existing naming and error wrapping (`fmt.Errorf("...: %w", err)`).
- Keep changes focused; avoid unrelated refactors in the same PR.

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md). Do not open public issues with secrets or live mailbox data.
