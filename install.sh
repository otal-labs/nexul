#!/usr/bin/env bash
# install.sh — first-time setup of a Nexul instance on this machine, and
# also the upgrade path: `git pull && ./install.sh` pulls the newer prebuilt
# images and restarts the stack on them. It installs Docker if missing, sets
# the logs password (the one value the setup wizard cannot collect, because
# OpenObserve reads it at its own first boot), pulls images, starts the
# stack, and prints where to open the setup wizard.
# Re-runnable: an existing logs password in .env is kept.
# NEXUL_VERSION=0.2.0 ./install.sh pins to that release (no leading v), or rolls back
# to one already pulled; NEXUL_VERSION=beta tracks the beta channel (a new beta per merge to master).
# Unset, it pulls `latest`: the newest stable release, or the newest beta until one exists.
set -euo pipefail
cd "$(dirname "$0")"

LOGS_EMAIL=nexul@nexul.local
DEV_PASSWORD=DevLogs-rotate-me-1

random() { head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 24; }

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is not installed; installing it with get.docker.com."
  curl -fsSL https://get.docker.com | sh
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "The docker compose plugin is missing. Install it, then run this script again." >&2
  exit 1
fi

touch .env
if grep -Eq "^NEXUL_LOGS_PASSWORD=.+" .env && ! grep -q "^NEXUL_LOGS_PASSWORD=$DEV_PASSWORD$" .env; then
  echo "Keeping the logs password already in .env."
  GENERATED=""
else
  read -r -s -p "Password for the logs UI (user $LOGS_EMAIL), blank to generate one: " LOGS_PASSWORD || true
  echo
  GENERATED=""
  if [[ -z "$LOGS_PASSWORD" ]]; then
    LOGS_PASSWORD=$(random)
    GENERATED=1
  fi
  sed -i '/^NEXUL_LOGS_/d' .env
  {
    echo "NEXUL_LOGS_EMAIL=$LOGS_EMAIL"
    echo "NEXUL_LOGS_PASSWORD=$LOGS_PASSWORD"
    echo "NEXUL_LOGS_TOKEN=$(random)"
  } >>.env
  chmod 600 .env
fi

docker compose pull
docker compose up -d

HOST=$(hostname -I 2>/dev/null | awk '{print $1}')
HOST=${HOST:-localhost}
echo
echo "Nexul is starting."
echo "  Setup wizard:  http://$HOST/"
echo "  Logs UI:       http://$HOST:5080  (user $LOGS_EMAIL)"
if [[ -n "$GENERATED" ]]; then
  echo "  Logs password: $LOGS_PASSWORD  (generated; also in .env)"
fi
echo "Put a tunnel or a TLS proxy in front for a public domain, then enter that https:// address as the instance URL in the wizard."
echo "To upgrade later: git pull, then run this script again."
