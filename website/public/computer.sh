#!/bin/sh
# Adds this computer to Nexul, with the command Add a computer shows; run it as yourself, never with sudo:
#   curl -fsSL https://nexul.io/computer.sh | sh -s -- <token>
# The token is signed by your instance and carries its address, a one-time code and the computer. This script reads
# only the address, to say where it connects; it cannot check the signature, and needs not: only the instance that
# signed the token accepts it, once. It hands over to install.sh, which fetches and verifies the nexul command and
# runs `nexul install computer`.
set -eu

fail() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

case "$(uname -s)" in
  Linux) ;;
  Darwin) fail "adding a Mac is coming soon; for now, add a Linux computer" ;;
  *) fail "adding this kind of computer is coming soon; for now, add a Linux computer" ;;
esac

token=${1:-}
[ $# -eq 1 ] && [ -n "$token" ] || fail "run the whole command Add a computer shows: it ends in one token"
payload=$(printf '%s' "$token" | cut -d. -f2 | tr -- '-_' '+/')
case $((${#payload} % 4)) in
  2) payload="$payload==" ;;
  3) payload="$payload=" ;;
esac
server=$(printf '%s' "$payload" | base64 -d 2>/dev/null | sed -n 's/.*"server":"\([^"]*\)".*/\1/p') || true
[ -n "$server" ] || fail "that is not a computer token; copy the whole command from Add a computer"
printf 'Adding this computer to %s\n' "$server"

script=$(curl -fsSL "${NEXUL_INSTALL_URL:-https://nexul.io/install.sh}") || fail "could not download install.sh"
printf '%s\n' "$script" | sh -s -- computer --token "$token"
