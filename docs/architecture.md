# Architecture

## Goals

- **Stable CLI surface** — `search`, `send`, `show` keep similar flags across protocols where possible.
- **Pluggable backends** — each protocol lives under `internal/protocol/<name>`.
- **Automation-first** — JSON output, env-based secrets, exit codes suitable for scripts.

## Layering

```text
cmd/mailcli/main.go
        │
        ▼
internal/app          ← Cobra, flag parsing, profile merge
        │
        ├── internal/config
        ├── internal/domain     ← protocol id, future shared models
        └── internal/protocol/
                └── ews/        ← today: search, send, show, ewshttp, message
                └── imap/       ← search, show
                └── smtp/       ← send path (paired with imap profile)
```

## EWS (current)

| Package | Role |
|---------|------|
| `protocol/ews/ewshttp` | HTTPS SOAP client, retries, NTLM |
| `protocol/ews/search` | FindItem, enrichment via GetItem |
| `protocol/ews/send` | CreateItem / SendAndSaveCopy |
| `protocol/ews/show` | GetItem for one message |
| `protocol/ews/message` | Shared GetItem request/parse helpers |
| `protocol/ews` | `NewHTTPClient`, `ValidateProfile` |

Commands in `internal/app` select the backend using `profile.protocol` (default `ews`). Unsupported protocols fail fast with a clear error.

## IMAP / SMTP (current)

| Package | Role |
|---------|------|
| `protocol/imap` | Dial, login, folder name mapping |
| `protocol/imap/search` | IMAP SEARCH + FETCH, client-side filters |
| `protocol/imap/show` | FETCH one message by UID |
| `protocol/smtp` | Host/TLS settings |
| `protocol/smtp/send` | MIME build + SMTP delivery, optional Sent-folder verify via IMAP |

`protocol: imap` profiles use **IMAP** for `search`/`show` and **SMTP** for `send`. `item_id` in JSON output is the **UID** within the selected folder. Search uses **UidSearch** + client-side filters (`--query`, Chinese keywords); keyword scans default to the newest ~800 messages in the folder.

## Configuration

EWS fields live at the profile top level (`endpoint`, `user`, `auth_type`, …). IMAP/SMTP can use **`provider`** presets (`internal/config/provider.go`) so `imap.host` / `smtp.host` are filled from `qq`, `163`, `gmail`, etc. Nested blocks override presets:

```yaml
profiles:
  gmail:
    protocol: imap
    imap:
      host: imap.gmail.com:993
      tls: true
    smtp:
      host: smtp.gmail.com:587
```

Existing EWS profiles remain valid without `protocol` (implicit `ews`).

## Adding a protocol

1. Implement search/send/show (or a subset) under `internal/protocol/<proto>`.
2. Register in `internal/app` command `RunE` after `requireSupportedProtocol`.
3. Extend `config.Profile` and example YAML.
4. Document env vars and limitations in `docs/使用说明.md` and README.

## Cross-compilation

Static Go binary; example:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o mailcli-linux-amd64 ./cmd/mailcli
```

No cgo — straightforward builds for Windows, Linux, and macOS (amd64/arm64).
