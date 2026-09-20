#!/usr/bin/env bash
set -euo pipefail

main() {
  if ! command -v git >/dev/null 2>&1; then
    echo "Git is required. Install Git, then run this script again." >&2
    return 1
  fi

  if [[ -e nexul || -L nexul ]]; then
    echo "The nexul directory already exists. To upgrade, run git pull and ./install.sh inside it." >&2
    return 1
  fi

  git clone https://github.com/otal-labs/nexul.git nexul
  if [[ -t 0 || -t 1 || -t 2 ]]; then
    exec "$BASH" nexul/install.sh </dev/tty
  fi
  exec "$BASH" nexul/install.sh
}

main "$@"
