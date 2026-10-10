# The tunnel command installs T3 Code but never pairs it

Pairing's first step runs `tunnel.sh` (or `tunnel.ps1`) and waits until the tunnel is online and the harness answers
through it (ADR 0062). A computer with no T3 Code could never pass that second check, and the person had to find T3
Code's installer on their own before step 2 made sense.

Decision: after the tunnel, the script makes sure T3 Code answers on the tunnel's port. It reuses whatever it finds
(a server already answering, the desktop app, the `t3` command line) and installs nothing then. With none, it prints
what it is about to install and installs it: on macOS and Linux T3 Code's command line through T3 Code's own installer,
run as T3 Code's own background service; on Windows, where T3 Code has no service, the desktop app through winget. It
ends by printing the exact pairing command for that machine. `--no-t3` (`-NoT3`) skips all of it.

- **T3 Code's own installer and service, never a copy.** The script calls `https://t3.codes/install.sh` and
  `t3 service install` rather than downloading a release or writing a unit itself, so updates, checksums and
  `t3 service status` stay T3 Code's.
- **No prompt.** The script runs piped from curl, so it signals what it installs and offers the flag, instead of asking.
- **No pairing.** The script never runs `t3 pair` or sends anything to Nexul; the token still goes through step 2 by
  hand. Rejected: pairing from the script, which would put a credential-minting call and a write to the instance
  inside a one-line install command the person pasted for a tunnel.

The trade-off: a Nexul script now installs a third-party tool on the person's computer, and breaks when T3 Code's
installer or service command changes. The port is not stored on the computer, so a tunnel made with a port other than
3773 needs `--port` added by hand. Decided 2026-10-10.
