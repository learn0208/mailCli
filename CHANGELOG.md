# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- [ROADMAP.md](ROADMAP.md) — completed vs planned capabilities.
- GitHub Actions release workflow (GoReleaser on `v*` tags).
- Open-source layout: `cmd/mailcli`, `internal/protocol/ews`, docs, CI.
- `protocol` field in config profiles (default `ews`).
- `MAILCLI_*` environment variables (`EWS_*` still supported).

### Changed

- CLI product name **mailCli**; command/binary **mailcli** (formerly **ews-cli**).
- Default config path: `~/.mailcli.yaml` (falls back to `~/.ews-cli.yaml`).
- Go module path: `github.com/learn0208/mailcli`.

## [0.1.0] - 2026-05-14

### Added

- EWS search, send, show, discover commands.
- YAML profiles, Basic/NTLM/OAuth auth, JSON/table output.
