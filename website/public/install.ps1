& {
  $ErrorActionPreference = "Stop"

  if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    throw "Git is required. Install Git, then run this script again."
  }

  if (Test-Path -LiteralPath nexul) {
    throw "The nexul directory already exists. To upgrade, run git pull and .\install.ps1 inside it."
  }

  git clone https://github.com/otal-labs/nexul.git nexul
  if ($LASTEXITCODE -ne 0) {
    throw "Could not download Nexul. Git exited with code $LASTEXITCODE."
  }

  & .\nexul\install.ps1
  if ($LASTEXITCODE -ne 0) {
    throw "The Nexul installer failed with exit code $LASTEXITCODE."
  }
}
