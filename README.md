# mailCli

[![CI](https://github.com/learn0208/mailCli/actions/workflows/ci.yml/badge.svg)](https://github.com/learn0208/mailCli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Cross-platform CLI for programmatic mail **search**, **send**, and **read** — built for automation (CI/CD, scripts, monitoring).

The command name is **`mailcli`** (lowercase, typical for shell tools). Product name: **mailCli**. **Today:** [EWS](https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/ews-operations-in-exchange) is supported. **Next:** IMAP/SMTP — see **[ROADMAP.md](ROADMAP.md)** (done vs planned).

### Install from GitHub Releases

Download the archive for your OS/arch from **[Releases](https://github.com/learn0208/mailCli/releases)** (`mailcli_x.y.z_os_arch.zip` / `.tar.gz`), unpack, and put `mailcli` on your `PATH`. Checksums are in `checksums.txt` on each release.

## Features

- **search** — server-side FindItem with filters (subject, body, from, dates, folder, read state, attachments)
- **send** — plain/HTML, attachments, optional Sent Items verification
- **show** — fetch one message by EWS ItemId (text / html / json)
- **discover** — print common EWS endpoint URL hints for a domain
- **JSON output** for scripting; table output for humans
- Static binary, no JVM/.NET runtime

## Quick start

### Build

```bash
go build -o mailcli ./cmd/mailcli
```

Windows:

```powershell
go build -o mailcli.exe .\cmd\mailcli
```

### Config

Copy [docs/examples/mailcli.example.yaml](docs/examples/mailcli.example.yaml) to `~/.mailcli.yaml` and set your profile. Legacy `~/.ews-cli.yaml` is still detected if present.

```yaml
profiles:
  default:
    protocol: ews
    endpoint: https://mail.example.com/EWS/Exchange.asmx
    user: you@example.com
    auth_type: basic
```

Set password via environment (recommended):

```bash
export MAILCLI_PASSWORD='your-secret'
```

### Examples

```bash
mailcli search --subject "weekly report" --output json
mailcli send --to a@b.com --subject "Hello" --text "Body"
mailcli show --item-id "AAMkAG..." --format json
```

## Documentation

| Document | Description |
|----------|-------------|
| [ROADMAP.md](ROADMAP.md) | **Completed features & planned work** |
| [docs/使用说明.md](docs/使用说明.md) | Full user guide (Chinese) |
| [docs/architecture.md](docs/architecture.md) | Code layout and protocol plug-ins |
| [docs/RELEASING.md](docs/RELEASING.md) | How maintainers cut a release (tags + CI) |
| [docs/design/prd-ews.md](docs/design/prd-ews.md) | Original EWS product requirements |

## Environment variables

`MAILCLI_*` is preferred. `EWS_*` still works for backward compatibility with early builds.

| Variable | Purpose |
|----------|---------|
| `MAILCLI_ENDPOINT` | EWS URL |
| `MAILCLI_USER` | Username / mailbox |
| `MAILCLI_PASSWORD` | Password |
| `MAILCLI_TOKEN` | OAuth access token |
| `MAILCLI_AUTH_TYPE` | `basic`, `ntlm`, `oauth` |
| `MAILCLI_DOMAIN` | NTLM domain |
| `MAILCLI_TIMEOUT` | HTTP timeout (seconds) |
| `MAILCLI_SERVER_VERSION` | SOAP `RequestServerVersion` (default `Exchange2016`) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
