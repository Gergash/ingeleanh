# Cross-compile C2 agent for lab targets (MVP-07 multi-OS).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Out = Join-Path $Root "dist"
New-Item -ItemType Directory -Force -Path $Out | Out-Null

Write-Host "Building agents into $Out"
$env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -o (Join-Path $Out "c2-agent-linux-amd64") (Join-Path $Root "cmd/agent")
$env:GOOS = "linux"; $env:GOARCH = "arm64"
go build -o (Join-Path $Out "c2-agent-linux-arm64") (Join-Path $Root "cmd/agent")
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -o (Join-Path $Out "c2-agent-windows-amd64.exe") (Join-Path $Root "cmd/agent")
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Get-ChildItem $Out
