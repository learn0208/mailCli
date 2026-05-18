# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **IMAP/SMTP** — `protocol: imap` profiles with `search`, `show` (UID), and `send` (SMTP).
- **Provider presets** — `provider: qq|163|gmail|…` and domain inference; `mailcli providers list|show|doc`, `mailcli profile show`.
- `search --query` and positional `search "keyword"` (IMAP: subject/from/body).
- Nested `imap` / `smtp` config blocks; `MAILCLI_IMAP_HOST`, `MAILCLI_SMTP_HOST`, `MAILCLI_PROVIDER`.
- Per-provider docs under [docs/providers/](docs/providers/) and YAML examples in [docs/examples/providers/](docs/examples/providers/).
- [ROADMAP.md](ROADMAP.md) — completed vs planned capabilities.
- GitHub Actions release workflow (GoReleaser on `v*` tags).
- Open-source layout: `cmd/mailcli`, `internal/protocol/ews`, docs, CI.
- `protocol` field in config profiles (default `ews`).
- `MAILCLI_*` environment variables (`EWS_*` still supported).

### Fixed

- IMAP search: use **UidSearch** / **UidFetch** (UID vs sequence number).
- IMAP search: avoid empty `SearchCriteria.Header` panic; non-ASCII keywords filtered client-side.
- IMAP `show`: fetch by UID; folder must match search.

### Changed

- CLI product name **mailCli**; command/binary **mailcli** (formerly **ews-cli**).
- Config lookup: cwd `.mailcli.yaml` → `~/.mailcli.yaml` → `~/.ews-cli.yaml`.
- `search --default-days` default **30** (was 7).
- Go module path: `github.com/learn0208/mailcli`.

## [0.1.0] - 2026-05-14

### Added

- EWS search, send, show, discover commands.
- YAML profiles, Basic/NTLM/OAuth auth, JSON/table output.
