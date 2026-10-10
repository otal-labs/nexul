# Connects this Windows computer to Nexul through its Cloudflare tunnel, with the command the pairing dialog shows,
# in PowerShell opened as administrator:
#   & ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) <token> [-Port <port>] [-NoT3]
# It installs cloudflared when it is missing and runs it as a service with this computer's tunnel token, then installs
# T3 Code's desktop app when no T3 Code is found: on Windows T3 Code has no background service to install instead.
& {
  param([string]$Token, [int]$Port = 3773, [switch]$NoT3)
  $ErrorActionPreference = 'Stop'
  if (-not $Token) { throw 'usage: & ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) <token> [-Port <port>] [-NoT3]' }
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

  if ($NoT3) {
    Write-Host "Done. Skipped T3 Code (-NoT3): the pairing dialog turns green once Cloudflare sees the tunnel and T3 Code answers on port $Port."
    return
  }
  $t3Home = if ($env:T3CODE_HOME) { $env:T3CODE_HOME } else { Join-Path $HOME '.t3' }
  $launcher = Join-Path $t3Home 'bin\t3.cmd'
  $pair = "& `"$launcher`" pair"
  $answering = $true
  try { Invoke-WebRequest "http://127.0.0.1:$Port/.well-known/t3/environment" -UseBasicParsing -TimeoutSec 2 | Out-Null } catch { $answering = $false }
  if ($answering) {
    Write-Host "T3 Code is running on port $Port"
    if (Get-Command t3 -ErrorAction SilentlyContinue) { $pair = 't3 pair' }
  } elseif (Test-Path $launcher) {
    Write-Host "T3 Code's desktop app is installed. Open T3 Code, then continue in Nexul."
  } else {
    Write-Host "Installing T3 Code's desktop app, which Nexul pairs with (winget install T3Tools.T3Code). Pass -NoT3 to skip this."
    winget install --id T3Tools.T3Code --exact --silent --accept-source-agreements --accept-package-agreements
    Write-Host 'Open T3 Code once, so it starts its server.'
  }
  Write-Host 'Done. The pairing dialog turns green once Cloudflare sees the tunnel.'
  Write-Host "Next, for Pair T3 Code: run $pair in PowerShell on this computer, then paste the token in Nexul."
} @args
