# Installs the nexul command into %ProgramFiles%\Nexul, then runs `nexul install` with this script's arguments.
#   irm https://nexul.io/install.ps1 | iex
#   & ([scriptblock]::Create((irm https://nexul.io/install.ps1))) runner --server <url> --name <name> --code <code>
# Set $env:NEXUL_VERSION = "v0.2.1" first to pin a release; unset, it takes the newest stable release, or the
# newest beta before one exists. Without administrator rights it re-runs itself elevated, carrying the NEXUL_*
# variables and the arguments. Everything runs in a script block so nothing leaks into the calling session.
& {
  $install = {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    # Windows PowerShell 5.1 can default to TLS versions GitHub no longer accepts.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $repo = 'otal-labs/nexul'
    $releases = if ($env:NEXUL_RELEASE_URL) { $env:NEXUL_RELEASE_URL } else { "https://github.com/$repo/releases/download" }
    $api = if ($env:NEXUL_API_URL) { $env:NEXUL_API_URL } else { 'https://api.github.com' }
    $binDir = if ($env:NEXUL_BIN_DIR) { $env:NEXUL_BIN_DIR } else { Join-Path $env:ProgramFiles 'Nexul' }

    $tag = $env:NEXUL_VERSION
    if (-not $tag) {
      # releases/latest skips prereleases, so before the first stable release the newest beta is the answer.
      try { $tag = (Invoke-RestMethod "$api/repos/$repo/releases/latest").tag_name } catch { $tag = $null }
    }
    if (-not $tag) {
      $tag = @(Invoke-RestMethod "$api/repos/$repo/releases?per_page=1")[0].tag_name
    }
    if (-not $tag) { throw 'could not find a Nexul release' }
    if (-not $tag.StartsWith('v')) { $tag = "v$tag" }

    # Windows on Arm runs the amd64 build under emulation; there is no native arm64 build.
    $asset = 'nexul-windows-amd64.exe'
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("nexul-" + [guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
      Write-Host "Downloading nexul $tag (windows/amd64)"
      Invoke-WebRequest "$releases/$tag/$asset" -OutFile (Join-Path $tmp 'nexul.exe') -UseBasicParsing
      Invoke-WebRequest "$releases/$tag/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt') -UseBasicParsing
      $line = Get-Content (Join-Path $tmp 'checksums.txt') | Where-Object { ($_ -split '\s+')[1] -eq $asset } | Select-Object -First 1
      if (-not $line) { throw "checksums.txt lists no $asset" }
      $want = ($line -split '\s+')[0]
      $got = (Get-FileHash (Join-Path $tmp 'nexul.exe') -Algorithm SHA256).Hash
      if ($got -ne $want) { throw "checksum mismatch for $asset; refusing to install it" }
      New-Item -ItemType Directory -Force -Path $binDir | Out-Null
      $exe = Join-Path $binDir 'nexul.exe'
      Copy-Item -Force (Join-Path $tmp 'nexul.exe') $exe
    }
    finally {
      Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }

    # Put nexul on the machine PATH for new terminals, and on this session's for the install below.
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    if (-not (($machinePath -split ';') -contains $binDir)) {
      $newPath = if ($machinePath) { "$machinePath;$binDir" } else { $binDir }
      [Environment]::SetEnvironmentVariable('Path', $newPath, 'Machine')
    }
    if (-not (($env:Path -split [IO.Path]::PathSeparator) -contains $binDir)) {
      $env:Path = "$env:Path$([IO.Path]::PathSeparator)$binDir"
    }

    & $exe install @args
    if ($LASTEXITCODE -ne 0) { throw "nexul install stopped (exit code $LASTEXITCODE); fix what it said, then run: nexul install $args" }
  }

  $ErrorActionPreference = 'Stop'
  $principal = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
  if ($principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    & $install @args
    return
  }

  # The elevated shell starts with a fresh environment, so the NEXUL_* variables and arguments travel in the command.
  $quote = { "'" + ($args[0] -replace "'", "''") + "'" }
  $variables = @(Get-ChildItem env: | Where-Object Name -like 'NEXUL_*' | ForEach-Object { "`$env:$($_.Name) = $(& $quote $_.Value)" })
  $arguments = @($args | ForEach-Object { & $quote $_ }) -join ' '
  # A failure pauses the elevated window, which would otherwise close before its error could be read.
  $command = ($variables + "try { & {$install} $arguments } catch { Write-Host `$_ -ForegroundColor Red; Read-Host 'Press Enter to close'; exit 1 }") -join "`n"
  $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($command))
  Write-Host 'Nexul installs as a Windows service; approve the administrator prompt to continue.'
  $elevated = Start-Process powershell.exe -Verb RunAs -Wait -PassThru -ArgumentList '-NoProfile', '-ExecutionPolicy', 'Bypass', '-EncodedCommand', $encoded
  if ($elevated.ExitCode -ne 0) { throw "the elevated install stopped (exit code $($elevated.ExitCode))" }
} @args
