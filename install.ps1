# install.ps1 — Windows counterpart of install.sh, for Docker Desktop. Also the
# upgrade path: `git pull` then `.\install.ps1` again pulls the newer prebuilt
# images and restarts the stack on them. It installs Docker Desktop if
# missing, sets the logs password (the one value the setup wizard cannot
# collect, because OpenObserve reads it at its own first boot), pulls images,
# starts the stack, and prints where to open the setup wizard.
# Re-runnable: an existing logs password in .env is kept.
# $env:NEXUL_VERSION="<tag>" pins to that release, or rolls back to one
# already pulled; NEXUL_VERSION="beta" tracks the beta channel (a new beta per merge to master).
# Unset, it pulls `latest`: the newest stable release, or the newest beta until one exists.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$logsEmail = "nexul@nexul.local"
$devPassword = "DevLogs-rotate-me-1"

function New-Secret {
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  $bytes = New-Object byte[] 18
  $rng.GetBytes($bytes)
  return ([Convert]::ToBase64String($bytes) -replace '[/+=]', '')
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  Write-Host "Docker Desktop is not installed; installing it with winget."
  winget install --id Docker.DockerDesktop -e --accept-package-agreements --accept-source-agreements
  Write-Host "Start Docker Desktop once, then run this script again."
  exit 1
}
docker compose version | Out-Null

$lines = @()
if (Test-Path .env) { $lines = @(Get-Content .env) }
$existing = $lines | Where-Object { $_ -match '^NEXUL_LOGS_PASSWORD=(.+)$' -and $Matches[1] -ne $devPassword }
$generated = $false
if ($existing) {
  Write-Host "Keeping the logs password already in .env."
} else {
  $secure = Read-Host -AsSecureString "Password for the logs UI (user $logsEmail), blank to generate one"
  $password = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
  if (-not $password) {
    $password = New-Secret
    $generated = $true
  }
  $lines = @($lines | Where-Object { $_ -notmatch '^NEXUL_LOGS_' })
  $lines += "NEXUL_LOGS_EMAIL=$logsEmail"
  $lines += "NEXUL_LOGS_PASSWORD=$password"
  $lines += "NEXUL_LOGS_TOKEN=$(New-Secret)"
  Set-Content -Path .env -Value $lines
}

docker compose pull
docker compose up -d

Write-Host ""
Write-Host "Nexul is starting."
Write-Host "  Setup wizard:  http://localhost/"
Write-Host "  Logs UI:       http://localhost:5080  (user $logsEmail)"
if ($generated) {
  Write-Host "  Logs password: $password  (generated; also in .env)"
}
Write-Host "Put a tunnel or a TLS proxy in front for a public domain, then enter that https:// address as the instance URL in the wizard."
Write-Host "To upgrade later: git pull, then run this script again."
