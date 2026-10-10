# 07 — `nexul install computer` installs T3 Code when it is missing

**Status:** ready-for-agent

**Blocked by:** 02

Read first: `practices/go.md`, `practices/testing.md`, ADR 0142, `website/public/tunnel.sh` (`ensure_t3`,
`install_t3`), `website/public/tunnel.ps1`, the spec (Installing a personal runner).

## What to build

Move `tunnel.sh`'s T3 Code logic into `internal/install` as the `computer` install's "T3 Code" step: reuse
a T3 Code that answers or is installed, else install T3 Code's command line with T3 Code's own installer and
`t3 service install` (Linux, macOS), `libatomic1` where missing, or the desktop app with winget (Windows).
Print what it installs; `--no-t3` skips it. Never prompt. The desktop app found closed prints "Open T3
Code".

The installer now runs as root for the person who typed `sudo` (ticket 20), so T3 Code's installer and its
`t3 service install` run as that user (`runuser -u`/`sudo -u`), never as root, and T3 Code lands under that
user's home. T3 Code's background service is a user service on Linux; confirm that against T3 Code's installer
and, if it is, turn on `loginctl enable-linger <user>` so it survives logout (the runner itself needs no
lingering).

## Acceptance criteria

- [ ] Table tests over the found states (answering, desktop app, command line idle, none) with a fake
      commander, per OS.
- [ ] On nexul-box from the `clean` snapshot: one command ends with T3 Code answering on loopback and the
      computer paired.
- [ ] On nexul-box, after the install, logging out every session leaves T3 Code answering.
- [ ] The guide's "What the command installs" section replaces the tunnel command's.
