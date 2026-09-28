#!/bin/sh
# Connects this computer to Nexul through its Cloudflare tunnel, with the command the pairing dialog shows:
#   curl -fsSL https://nexul.io/tunnel.sh | sh -s -- <token>
# It installs cloudflared when it is missing and runs it as a service with this computer's tunnel token.
set -eu

main() {
  [ $# -eq 1 ] && [ -n "$1" ] || fail "usage: curl -fsSL https://nexul.io/tunnel.sh | sh -s -- <token>"
  token=$1
  sudo=""
  if [ "$(id -u)" -ne 0 ]; then
    command -v sudo >/dev/null 2>&1 || fail "run this as root, or install sudo"
    sudo=sudo
  fi

  case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "this script supports Linux and macOS; on Windows, run tunnel.ps1 from the pairing dialog" ;;
  esac

  if command -v cloudflared >/dev/null 2>&1; then
    printf 'cloudflared is installed (%s)\n' "$(cloudflared --version 2>/dev/null | head -n1)"
  else
    printf 'Installing cloudflared\n'
    "install_$os"
  fi

  printf 'Starting the tunnel as a service\n'
  if [ "$os" = darwin ]; then
    # Not as root: a login item, which is when T3 Code runs too.
    cloudflared service uninstall >/dev/null 2>&1 || true
    cloudflared service install "$token"
  else
    $sudo cloudflared service uninstall >/dev/null 2>&1 || true
    $sudo cloudflared service install "$token"
  fi
  printf 'Done. The pairing dialog turns green once Cloudflare sees the tunnel.\n'
}

# install_linux uses Cloudflare's signed package repository where there is one, so cloudflared updates with the
# system, and its release binary everywhere else.
install_linux() {
  if command -v apt-get >/dev/null 2>&1; then
    download https://pkg.cloudflare.com/cloudflare-main.gpg "Cloudflare's package key"
    $sudo mkdir -p -m 0755 /usr/share/keyrings
    $sudo install -m 0644 "$file" /usr/share/keyrings/cloudflare-main.gpg
    rm -f "$file"
    echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" |
      $sudo tee /etc/apt/sources.list.d/cloudflared.list >/dev/null
    $sudo apt-get update -qq
    $sudo apt-get install -y -qq cloudflared
    return
  fi
  for pm in dnf yum; do
    if command -v "$pm" >/dev/null 2>&1; then
      download https://pkg.cloudflare.com/cloudflared.repo "Cloudflare's package repository"
      $sudo install -m 0644 "$file" /etc/yum.repos.d/cloudflared.repo
      rm -f "$file"
      $sudo "$pm" install -y -q cloudflared
      return
    fi
  done
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    armv7l | armv6l) arch=arm ;;
    *) fail "no cloudflared build for $(uname -m)" ;;
  esac
  download "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-$arch" cloudflared
  $sudo install -m 0755 "$file" /usr/local/bin/cloudflared
  rm -f "$file"
}

install_darwin() {
  if command -v brew >/dev/null 2>&1; then
    brew install cloudflared
    return
  fi
  case "$(uname -m)" in
    arm64) arch=arm64 ;;
    *) arch=amd64 ;;
  esac
  tmp=$(mktemp -d)
  curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-$arch.tgz" -o "$tmp/cloudflared.tgz" ||
    fail "could not download cloudflared"
  tar -xzf "$tmp/cloudflared.tgz" -C "$tmp"
  $sudo mkdir -p /usr/local/bin
  $sudo install -m 0755 "$tmp/cloudflared" /usr/local/bin/cloudflared
  rm -rf "$tmp"
}

# download saves url to a new temporary file named in $file; a failed download stops here rather than being lost
# inside a pipe.
download() {
  file=$(mktemp)
  curl -fsSL "$1" -o "$file" || {
    rm -f "$file"
    fail "could not download $2 from $1; check this computer can reach it"
  }
}

fail() {
  echo "error: $*" >&2
  exit 1
}

main "$@"
