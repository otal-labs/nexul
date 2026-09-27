# Installs a Nexul automations host as a Windows service, with the command the instance shows when you add one:
#   & ([scriptblock]::Create((irm https://nexul.io/automations.ps1))) --server <url> --name <name> --code <code>
# It hands over to install.ps1, which fetches and verifies the nexul command and runs `nexul install automations`.
& {
  $ErrorActionPreference = 'Stop'
  $url = if ($env:NEXUL_INSTALL_URL) { $env:NEXUL_INSTALL_URL } else { 'https://nexul.io/install.ps1' }
  & ([scriptblock]::Create((Invoke-RestMethod $url))) automations @args
} @args
