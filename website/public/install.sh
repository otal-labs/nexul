#!/bin/sh
# Installs the nexul command on a Linux server or a Mac, then runs `nexul install`, which sets up Docker and Nexul.
# On Windows, use https://nexul.io/install.ps1 instead.
#   curl -fsSL https://nexul.io/install.sh | sh
#   curl -fsSL https://nexul.io/install.sh | sh -s -- --dir /srv/nexul --port 8080 --yes
#   curl -fsSL https://nexul.io/install.sh | sh -s -- runner --server <url> --name <name> --code <code>
#   curl -fsSL https://nexul.io/install.sh | sh -s -- computer --token <token>   (computer.sh runs this)
# A computer installs with sudo for the person who typed it: the nexul command goes in their own ~/.local/bin.
# NEXUL_VERSION=v0.2.1 pins a release; unset, it takes the newest stable release, or the newest beta before one exists.
set -eu

REPO=otal-labs/nexul
BIN_DIR=${NEXUL_BIN_DIR:-/usr/local/bin}
RELEASES=${NEXUL_RELEASE_URL:-https://github.com/$REPO/releases/download}
API=${NEXUL_API_URL:-https://api.github.com}

fail() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

main() {
  case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "this script supports Linux and macOS; on Windows, run: irm https://nexul.io/install.ps1 | iex" ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) fail "there is no Nexul build for $(uname -m)" ;;
  esac
  command -v curl >/dev/null 2>&1 || fail "curl is required"

  # Copying into the bin directory may need sudo; on a Mac the installer itself then runs as you, since Homebrew
  # refuses to run as root. A computer's command goes in the home of the person who typed sudo, and
  # `nexul install computer` hands it, and every folder it made there, to them.
  sudo=""
  if [ "${1:-}" = computer ]; then
    home=""
    [ "${SUDO_USER:-root}" = root ] || home=$(getent passwd "$SUDO_USER" | cut -d: -f6)
    [ -n "$home" ] || fail "run this from your own account with sudo, not logged in as root: the computer's runner runs as the person who typed sudo"
    BIN_DIR=${NEXUL_BIN_DIR:-$home/.local/bin}
  elif [ "$(id -u)" -ne 0 ]; then
    command -v sudo >/dev/null 2>&1 || fail "run this as root"
    sudo=sudo
  fi
  run_as=$sudo
  [ "$os" = darwin ] && run_as=""

  tag=${NEXUL_VERSION:-}
  if [ -z "$tag" ]; then
    # releases/latest skips prereleases, so before the first stable release the newest beta is the answer.
    tag=$(curl -fsSL "$API/repos/$REPO/releases/latest" 2>/dev/null | tag_name || true)
    [ -n "$tag" ] || tag=$(curl -fsSL "$API/repos/$REPO/releases?per_page=30" | newest_tag || true)
    [ -n "$tag" ] || fail "could not find a Nexul release"
  fi
  case "$tag" in v*) ;; *) tag="v$tag" ;; esac

  asset="nexul-$os-$arch"
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  printf 'Downloading nexul %s (%s/%s)\n' "$tag" "$os" "$arch"
  curl -fsSL "$RELEASES/$tag/$asset" -o "$tmp/nexul" || fail "could not download $asset for $tag"
  curl -fsSL "$RELEASES/$tag/checksums.txt" -o "$tmp/checksums.txt" || fail "could not download checksums.txt for $tag"
  want=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
  [ -n "$want" ] || fail "checksums.txt lists no $asset"
  got=$(sha256 "$tmp/nexul")
  [ "$want" = "$got" ] || fail "checksum mismatch for $asset; refusing to install it"
  $sudo mkdir -p "$BIN_DIR"
  $sudo install -m 0755 "$tmp/nexul" "$BIN_DIR/nexul"

  # Piped from curl, stdin is this script; the installer's questions go to the terminal instead, when there is one.
  if (: </dev/tty) 2>/dev/null; then
    $run_as "$BIN_DIR/nexul" install "$@" </dev/tty
    return
  fi
  $run_as "$BIN_DIR/nexul" install "$@"
}

# sha256 prints a file's digest with whichever tool the system has: sha256sum on Linux, shasum on macOS.
sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print $1 }'
    return
  fi
  shasum -a 256 "$1" | awk '{ print $1 }'
}

tag_name() {
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

# newest_tag picks the most recently published release. GitHub orders the list by tag text, which puts beta.9
# above beta.10, and a draft has no publish time, so it is never picked; a list without publish times keeps its order.
# The phone app's phone-v* and android-v* releases share the repository and carry no server binaries, so only v* tags count.
newest_tag() {
  list=$(cat)
  newest=$(printf '%s' "$list" | tr ',' '\n' | awk '
    /"tag_name"[[:space:]]*:/ { t = $0; sub(/.*"tag_name"[[:space:]]*:[[:space:]]*"/, "", t); sub(/".*/, "", t); if (t !~ /^v/) t = "" }
    /"published_at"[[:space:]]*:[[:space:]]*"/ {
      p = $0; sub(/.*"published_at"[[:space:]]*:[[:space:]]*"/, "", p); sub(/".*/, "", p)
      if (t != "") print p, t
      t = ""
    }' | sort | tail -n1 | cut -d' ' -f2)
  [ -n "$newest" ] || newest=$(printf '%s' "$list" | tag_name)
  printf '%s\n' "$newest"
}

main "$@"
