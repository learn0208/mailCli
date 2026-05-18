@echo off
REM Local dev helper — requires .mailcli.yml in repo root (gitignored; copy from docs/examples/mailcli.example.yaml)
go build -o mailcli.exe .\cmd\mailcli
if errorlevel 1 exit /b 1
mailcli.exe search --config .\.mailcli.yml --subject "test"
REM mailcli.exe send --config .\.mailcli.yml --to you@example.com --subject "test" --text "body"
