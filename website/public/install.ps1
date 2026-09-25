# Installs the nexul command on Windows, then runs `nexul install`, which checks for Docker Desktop and starts Nexul.
#   irm https://nexul.io/install.ps1 | iex
# Set $env:NEXUL_VERSION = "v0.2.1" first to pin a release; unset, it takes the newest stable release, or the
# newest beta before one exists. Everything runs in a script block so nothing leaks into the calling session.
& {
  $ErrorActionPreference = 'Stop'
  $ProgressPreference = 'SilentlyContinue'
  # Windows PowerShell 5.1 can default to TLS versions GitHub no longer accepts.
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

  $repo = 'otal-labs/nexul'
  $releases = if ($env:NEXUL_RELEASE_URL) { $env:NEXUL_RELEASE_URL } else { "https://github.com/$repo/releases/download" }
  $api = if ($env:NEXUL_API_URL) { $env:NEXUL_API_URL } else { 'https://api.github.com' }
  $binDir = if ($env:NEXUL_BIN_DIR) { $env:NEXUL_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\Nexul' }

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

  # Put nexul on the user's PATH for new terminals, and on this session's for the install below.
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (-not (($userPath -split ';') -contains $binDir)) {
    $newPath = if ($userPath) { "$userPath;$binDir" } else { $binDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
  }
  if (-not (($env:Path -split [IO.Path]::PathSeparator) -contains $binDir)) {
    $env:Path = "$env:Path$([IO.Path]::PathSeparator)$binDir"
  }

  & $exe install
  if ($LASTEXITCODE -ne 0) { throw "nexul install stopped (exit code $LASTEXITCODE); fix what it said, then run: nexul install" }
}
