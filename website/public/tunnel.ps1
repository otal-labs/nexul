# Connects this Windows computer to Nexul through its Cloudflare tunnel, with the command the pairing dialog shows,
# in PowerShell opened as administrator:
#   & ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) <token>
# It installs cloudflared when it is missing and runs it as a service with this computer's tunnel token.
& {
  param([string]$Token)
  $ErrorActionPreference = 'Stop'
  if (-not $Token) { throw 'usage: & ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) <token>' }
  $admin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)
  if (-not $admin) { throw 'Open PowerShell as administrator and run this again: installing a service needs it.' }

  if (-not (Get-Command cloudflared -ErrorAction SilentlyContinue)) {
    Write-Host 'Installing cloudflared'
    winget install --id Cloudflare.cloudflared --exact --silent --accept-source-agreements --accept-package-agreements
    # winget adds cloudflared to the machine PATH, which this session read before the install.
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
  }

  Write-Host 'Starting the tunnel as a service'
  if (Get-Service cloudflared -ErrorAction SilentlyContinue) { cloudflared service uninstall | Out-Null }
  cloudflared service install $Token
  Write-Host 'Done. The pairing dialog turns green once Cloudflare sees the tunnel.'
} @args
