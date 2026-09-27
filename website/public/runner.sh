#!/bin/sh
# Installs a Nexul runner as a service, with the command the instance shows when you add one:
#   curl -fsSL https://nexul.io/runner.sh | sh -s -- --server <url> --name <name> --code <code>
# It hands over to install.sh, which fetches and verifies the nexul command and runs `nexul install runner`.
set -eu
script=$(curl -fsSL "${NEXUL_INSTALL_URL:-https://nexul.io/install.sh}") || {
  echo "error: could not download install.sh" >&2
  exit 1
}
printf '%s\n' "$script" | sh -s -- runner "$@"
