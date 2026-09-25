#!/bin/sh
# Installs the nexul command on a Linux server or a Mac, then runs `nexul install`, which sets up Docker and Nexul.
# On Windows, use https://nexul.io/install.ps1 instead.
#   curl -fsSL https://nexul.io/install.sh | sh
#   curl -fsSL https://nexul.io/install.sh | sh -s -- --dir /srv/nexul --port 8080 --yes
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
  # refuses to run as root.
  sudo=""
  if [ "$(id -u)" -ne 0 ]; then
    command -v sudo >/dev/null 2>&1 || fail "run this as root"
    sudo=sudo
  fi
  run_as=$sudo
  [ "$os" = darwin ] && run_as=""

  tag=${NEXUL_VERSION:-}
  if [ -z "$tag" ]; then
    # releases/latest skips prereleases, so before the first stable release the newest beta is the answer.
    tag=$(curl -fsSL "$API/repos/$REPO/releases/latest" 2>/dev/null | tag_name || true)
    [ -n "$tag" ] || tag=$(curl -fsSL "$API/repos/$REPO/releases?per_page=1" | tag_name || true)
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

main "$@"
