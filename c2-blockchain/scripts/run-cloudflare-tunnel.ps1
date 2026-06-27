# Quick Tunnel para portal C2 (sin dominio). Requiere: go run ./cmd/server en :8443
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Config = Join-Path $Root "cloudflared\config.quick.yml"
$Origin = if ($env:C2_TUNNEL_ORIGIN) { $env:C2_TUNNEL_ORIGIN } else { "http://localhost:8443" }

$Cf = "${env:ProgramFiles(x86)}\cloudflared\cloudflared.exe"
if (-not (Test-Path $Cf)) {
	$CfCmd = Get-Command cloudflared -ErrorAction SilentlyContinue
	if ($CfCmd) { $Cf = $CfCmd.Source } else {
		Write-Error "cloudflared no encontrado. Instala: winget install Cloudflare.cloudflared"
	}
}

$Extra = @()
$CaBundle = "C:\Program Files\Git\usr\ssl\certs\ca-bundle.crt"
if (Test-Path $CaBundle) {
	$Extra += "--origin-ca-pool"
	$Extra += $CaBundle
}

Write-Host "Origen local: $Origin"
Write-Host "Config: $Config"
Write-Host "Espera la URL https://....trycloudflare.com y abre /portal/"
Write-Host ""

$Args = @(
	"tunnel",
	"--config", $Config,
	"--url", $Origin,
	"--no-prechecks",
	"--loglevel", "info",
	"--transport-loglevel", "error",
	"--management-diagnostics=false",
	"--metrics", "127.0.0.1:0"
) + $Extra

& $Cf @Args
