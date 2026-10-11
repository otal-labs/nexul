# Adds this Windows computer to Nexul, with the command Add a computer shows, in PowerShell from your own account:
#   & ([scriptblock]::Create((irm https://nexul.io/computer.ps1))) <token> [--no-t3]
# It needs no administrator: the runner it installs is yours, in your %LOCALAPPDATA%\Nexul, and runs as you while you
# are logged on. The token is signed by your instance and carries its address, a one-time code and the computer. This
# script reads only the address, to say where it connects; it cannot check the signature, and needs not: only the
# instance that signed the token accepts it, once. It hands over to install.ps1, which fetches and verifies the nexul
# command and runs `nexul install computer`, which also installs T3 Code's desktop app when no T3 Code is found;
# --no-t3 skips that.
& {
  $ErrorActionPreference = 'Stop'
  $token = $args[0]
  if (-not $token) { throw "run the whole command Add a computer shows: it ends in one token" }
  $server = $null
  try {
    $payload = ($token -split '\.')[1].Replace('-', '+').Replace('_', '/')
    $payload += '=' * ((4 - $payload.Length % 4) % 4)
    $server = ([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($payload)) | ConvertFrom-Json).server
  } catch { }
  if (-not $server) { throw "that is not a computer token; copy the whole command from Add a computer" }
  Write-Host "Adding this computer to $server"
  $url = if ($env:NEXUL_INSTALL_URL) { $env:NEXUL_INSTALL_URL } else { 'https://nexul.io/install.ps1' }
  & ([scriptblock]::Create((Invoke-RestMethod $url))) computer --token @args
} @args
