# Security policy

## Supported versions

Security fixes are applied to the latest release on the default branch.

## Reporting a vulnerability

Please **do not** file a public issue for security-sensitive reports.

Instead, contact the maintainers privately (e.g. via GitHub Security Advisories or the contact method listed on the repository). Include:

- Description of the issue and impact
- Steps to reproduce
- Affected version or commit

We aim to acknowledge reports within a few business days.

## Handling credentials

- Never commit passwords, tokens, or real `*.yaml` config files.
- Prefer `MAILCLI_PASSWORD` (or legacy `EWS_PASSWORD`) over `--password` in scripts.
- Use `--verbose` only in trusted environments (logs may contain mail metadata and SOAP bodies).
