@echo off
REM Local dev helper — copy docs/examples/providers/qq.yaml to .mailcli.yaml (gitignored)
REM Set MAILCLI_PASSWORD to your QQ authorization code before running send/search.

go build -o mailcli.exe .\cmd\mailcli
if errorlevel 1 exit /b 1

mailcli.exe profile show
mailcli.exe search --limit 3 --output table
REM mailcli.exe search "关键词"
REM mailcli.exe show --item-id 1460 --format text
REM mailcli.exe send --to you@example.com --subject "test" --text "body"
