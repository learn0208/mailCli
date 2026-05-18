# mailCli

[![CI](https://github.com/learn0208/mailCli/actions/workflows/ci.yml/badge.svg)](https://github.com/learn0208/mailCli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Cross-platform CLI for programmatic mail **search**, **send**, and **read** — built for automation (CI/CD, scripts, monitoring).

The command name is **`mailcli`** (lowercase, typical for shell tools). Product name: **mailCli**. Supports **[EWS](https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/ews-operations-in-exchange)** (Exchange) and **IMAP/SMTP** (generic mail providers). See **[ROADMAP.md](ROADMAP.md)** for planned work.

### Install from GitHub Releases

Download the archive for your OS/arch from **[Releases](https://github.com/learn0208/mailCli/releases)** (`mailcli_x.y.z_os_arch.zip` / `.tar.gz`), unpack, and put `mailcli` on your `PATH`. Checksums are in `checksums.txt` on each release.

## Features

- **search** — EWS FindItem or IMAP SEARCH + filters (subject, body, from, dates, folder, read state, attachments)
- **send** — EWS CreateItem or SMTP (plain/HTML, attachments, optional Sent folder verification)
- **show** — fetch one message by EWS ItemId or IMAP UID (text / html / json)
- **discover** — print common EWS or IMAP/SMTP host hints for a domain
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

Copy [docs/examples/mailcli.example.yaml](docs/examples/mailcli.example.yaml) to `~/.mailcli.yaml` (or project `.mailcli.yaml`). Legacy `~/.ews-cli.yaml` is still detected if present.

**Exchange (EWS):**

```yaml
profiles:
  default:
    protocol: ews
    endpoint: https://mail.example.com/EWS/Exchange.asmx
    user: you@example.com
    auth_type: basic
```

**QQ / Gmail / 163 (IMAP)** — set `provider` or use a known email domain; hosts are filled automatically:

```yaml
profiles:
  qq:
    protocol: imap
    provider: qq
    user: 123456789@qq.com
```

See [docs/providers/README.md](docs/providers/README.md) for authorization codes (QQ/163) and step-by-step setup.

Set password via environment (recommended):

```bash
export MAILCLI_PASSWORD='your-secret'
```

### Examples

```bash
# EWS or IMAP
mailcli search --subject "weekly report" --output json
mailcli search "keyword"                    # IMAP: subject/from/body
mailcli send --to a@b.com --subject "Hello" --text "Body"
mailcli show --item-id "AAMkAG..." --format json   # EWS ItemId
mailcli show --item-id 1460 --format text          # IMAP UID

mailcli providers doc qq
mailcli profile show
mailcli discover --user you@163.com
```

## Documentation

| Document | Description |
|----------|-------------|
| [ROADMAP.md](ROADMAP.md) | **Completed features & planned work** |
| [docs/使用说明.md](docs/使用说明.md) | Full user guide (Chinese) |
| [docs/architecture.md](docs/architecture.md) | Code layout and protocol plug-ins |
| [docs/providers/README.md](docs/providers/README.md) | **Per-provider setup guides** (QQ, 163, Gmail, …) |
| [docs/examples/providers/](docs/examples/providers/) | Copy-paste YAML profile examples |
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
| `MAILCLI_IMAP_HOST` | IMAP `host:port` |
| `MAILCLI_SMTP_HOST` | SMTP `host:port` |
| `MAILCLI_PROVIDER` | Preset id (`qq`, `163`, `gmail`, …) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
