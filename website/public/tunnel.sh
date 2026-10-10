#!/bin/sh
# Connects this computer to Nexul through its Cloudflare tunnel, with the command the pairing dialog shows:
#   curl -fsSL https://nexul.io/tunnel.sh | sh -s -- <token> [--port <port>] [--no-t3]
# It installs cloudflared when it is missing and runs it as a service with this computer's tunnel token, then makes
# sure T3 Code answers on the port the tunnel routes to, installing its command line as a service when there is none.
set -eu

usage="usage: curl -fsSL https://nexul.io/tunnel.sh | sh -s -- <token> [--port <port>] [--no-t3]"

main() {
  token=""
  port=3773
  t3=yes
  while [ $# -gt 0 ]; do
    case "$1" in
      --no-t3) t3=no ;;
      --port)
        [ $# -gt 1 ] || fail "$usage"
        port=$2
        shift
        ;;
      -*) fail "$usage" ;;
      *)
        [ -z "$token" ] || fail "$usage"
        token=$1
        ;;
    esac
    shift
  done
  [ -n "$token" ] || fail "$usage"
  case "$port" in '' | *[!0-9]*) fail "--port takes the T3 Code port the tunnel was made with, like 3773" ;; esac
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

  if [ "$t3" = no ]; then
    printf 'Done. Skipped T3 Code (--no-t3): the pairing dialog turns green once Cloudflare sees the tunnel and T3 Code answers on port %s.\n' "$port"
    return
  fi
  ensure_t3
  printf 'Done. The pairing dialog turns green once Cloudflare sees the tunnel.\n'
  printf 'Next, for Pair T3 Code: run %s pair on this computer, then paste the token in Nexul.\n' "$(tilde "$t3_command")"
}

# ensure_t3 reuses a T3 Code it finds answering or installed, and installs one only when there is none.
ensure_t3() {
  desktop=${T3CODE_HOME:-$HOME/.t3}/bin/t3
  cli=${T3CODE_INSTALL_BIN_DIR:-$HOME/.local/bin}/t3
  t3_command=$(command -v t3 2>/dev/null || true)
  if [ -z "$t3_command" ] && [ -x "$desktop" ]; then t3_command=$desktop; fi
  if [ -z "$t3_command" ] && [ -x "$cli" ]; then t3_command=$cli; fi

  if t3_answers; then
    printf 'T3 Code is running on port %s\n' "$port"
    t3_command=${t3_command:-t3}
    return
  fi
  if [ -x "$desktop" ]; then
    printf 'T3 Code'"'"'s desktop app is installed. Open T3 Code, then continue in Nexul.\n'
    return
  fi
  if [ -n "$t3_command" ]; then
    printf 'T3 Code is installed (%s) but nothing answers on port %s. Start it with %s service install, or %s serve --port %s, then continue in Nexul.\n' \
      "$(tilde "$t3_command")" "$port" "$(tilde "$t3_command")" "$(tilde "$t3_command")" "$port"
    return
  fi
  install_t3
}

# install_t3 uses T3 Code's own installer (no root, ~/.local/bin) and its own background service, never a copy of either.
install_t3() {
  if [ "$os" = darwin ] && [ "$port" != 3773 ]; then
    fail "T3 Code's macOS service always starts on port 3773; install T3 Code yourself and run t3 serve --port $port, or make the tunnel with port 3773"
  fi
  printf 'Installing T3 Code, which Nexul pairs with: its command line from https://t3.codes/install.sh into %s, run as a background service on port %s. Pass --no-t3 to skip this.\n' \
    "$(tilde "${cli%/*}")" "$port"
  if [ "$os" = linux ]; then install_libatomic; fi
  download https://t3.codes/install.sh "T3 Code's installer"
  sh "$file" || {
    rm -f "$file"
    fail "T3 Code's installer failed; install T3 Code yourself from https://t3.codes, then continue in Nexul"
  }
  rm -f "$file"
  if [ "$os" = linux ]; then
    # Lingering keeps T3 Code's user service running at boot and after logout; turning it on needs root.
    if command -v loginctl >/dev/null 2>&1; then $sudo loginctl enable-linger "$(id -un)"; fi
    if [ "$port" != 3773 ]; then
      dropin=${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/t3code.service.d
      mkdir -p "$dropin"
      printf '[Service]\nEnvironment=T3CODE_PORT=%s\n' "$port" >"$dropin/nexul-port.conf"
    fi
  fi
  "$cli" service install ||
    fail "T3 Code is installed, but its background service did not start; run $(tilde "$cli") serve --port $port in a terminal you keep open, then continue in Nexul"
  printf 'Waiting for T3 Code to answer on port %s\n' "$port"
  tries=0
  until t3_answers; do
    tries=$((tries + 1))
    [ "$tries" -lt 30 ] || fail "T3 Code's service is installed but nothing answers on port $port; run $(tilde "$cli") service status to see why"
    sleep 1
  done
  t3_command=$cli
}

# install_libatomic adds the C library T3 Code's Linux binary links, which minimal server images leave out.
install_libatomic() {
  case "$(ldconfig -p 2>/dev/null)" in *libatomic.so.1*) return ;; esac
  if command -v apt-get >/dev/null 2>&1; then
    $sudo apt-get update -qq
    $sudo apt-get install -y -qq libatomic1
    return
  fi
  for pm in dnf yum; do
    if command -v "$pm" >/dev/null 2>&1; then
      $sudo "$pm" install -y -q libatomic
      return
    fi
  done
}

t3_answers() {
  curl -fsS -m 2 -o /dev/null "http://127.0.0.1:$port/.well-known/t3/environment" 2>/dev/null
}

# tilde shortens a path under the home directory, which is how a person types it.
tilde() {
  case "$1" in
    "$HOME"/*) printf '~%s' "${1#"$HOME"}" ;;
    *) printf '%s' "$1" ;;
  esac
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
