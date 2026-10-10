# Computers are paired through a personal runner, not a Cloudflare tunnel

Status: proposed, with the personal runners effort (`.scratch/personal-runners/`).

Supersedes ADR 0062 and ADR 0142. Amends ADR 0031, ADR 0073, ADR 0074 and ADR 0102.

Sharing a computer is built as computer rules (ADR 0148, `.scratch/permission-overrides/`).

Nexul runs on a server; T3 Code runs on each person's own computer, usually behind a router. ADR 0062 reached it
through a tunnel in the instance's own Cloudflare account, one per computer, closed by an Access rule. That made
Cloudflare with Zero Trust a precondition for the main feature, made pairing three steps (a tunnel command, a
pasted pairing link, setup), and left a paste every 30 days because T3 Code's session cannot be refreshed. Pair by
URL, the other way, asked people to open T3 Code's port on every interface.

Decision: a person adds a computer by installing a **personal runner** on it with one command,
`curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`. The runner is the existing `nexul-runner` in a personal
mode: enrolled with a one-time code bound to that person and that computer, with no Docker. The script installs for
`$SUDO_USER`, the person who typed `sudo`, never for root, and refuses a root login with no `SUDO_USER`. Because
sudo is always there, the runner is a system service whose user is that person (a systemd system unit with
`User=`, a LaunchDaemon with `UserName`, on Windows a service under their account or the closest equivalent), so
it starts at boot and survives logout with no lingering, and the process itself never runs as root. It holds the
runner's usual outbound WebSocket, and the server reaches T3 Code through it:

- **Each relayed connection is its own WebSocket, opened by the runner.** The server asks over the control
  connection (`harness_dial {id}`); the runner dials T3 Code on loopback and opens `/api/runners/streams/{id}`
  with its host credential; both ends wrap the socket with coder/websocket's `NetConn`. The harness client's
  `http.Client` dials a computer's address through this, so both T3 clients and the `harness.Client` interface
  are unchanged.
- **One process, isolated lanes.** T3 Code's bytes never ride the control connection that carries jobs, logs
  and heartbeats; each lane has its own caps; a panic in one stream or job ends only that unit, and the service
  manager restarts the process for anything worse. A runner update drops relayed streams briefly, and the
  server redials and resumes turns as it already does for a dropped harness connection.
- **The runner decides what it dials:** loopback, on the T3 Code port it found. No frame names a host or port.
- **The runner mints the pairing token locally** (`t3 auth pairing create`) and hands it over its own
  authenticated connection; the server exchanges it through the relay exactly as before, and re-pairs on its
  own before the 30-day session ends.
- **It is the only way to pair.** The tunnel flow, the pasted link and Pair by URL are retired: existing
  computers keep working until their owner installs a runner, which adopts the same computer record, and then
  the tunnel pairing code is deleted. They keep working for at least 30 days after runners ship, and the
  deletion waits until none remain or every remaining owner has had the in-app notice for 30 days; their rows,
  links and setup are kept. Cloudflare stays for the instance's domain, gateways and exposures.
- **The install makes T3 Code answer, as the person.** `nexul install computer` reuses a T3 Code it finds,
  starts the background service of a `t3` command line that is not running, and otherwise runs T3 Code's own
  installer and `t3 service install` as the person, never as root (`--no-t3` skips all of it). T3 Code's Linux
  service is a systemd user service, so the install turns on lingering for that person; the runner needs none.
- **The runner never updates T3 Code.** It reports the version it finds and restarts a stopped background service;
  updating is the person's.
- **A personal runner is private to its owner.** It never takes deploy jobs, never shows in runner or machine
  lists, publishes members-only events that reach only its owner's sockets, and is checked by ownership and the
  owner's own rules, never by role, so no role or Owner bypass reaches it (ADR 0148). Its facts are the owner's
  alone. The owner may later share a computer with named people through computer rules (ADR 0148), at two
  levels, running commands and running agents; a run an agent makes for
  a grantee carries a short-lived Nexul token for the person who started it, never the owner's. Disabling or
  removing an account revokes its runners.
- **One deliberate exception: commands on a computer are on the record.** The audit log records which computer
  and the command for actions on it, and a new instance-wide permission, Read computer activity, lets its holder
  read that. The Owner role has it by default. It reads the record only: it lists no computers, shows no facts and
  grants no use. Each computer's page tells its owner so. The command line, who ran it, when and the exit code
  are kept forever; command output is deleted after 30 days.

Why one WebSocket per relayed connection: each relayed connection keeps TCP's own flow control end to end, a
large transcript never delays another stream or the control connection's heartbeats, and a control reconnect
does not cut running turns. Multiplexing streams over the control socket would need a new dependency or
flow-control code of our own. The cost is one WebSocket handshake per new connection, which HTTP keep-alive and
T3 Code's long-lived session make rare.

Why minting in the runner is not what ADR 0142 rejected: that ADR refused to mint a credential and write to the
instance from a one-line command a person pasted. Here the long-running runner mints on request over its own
enrolled connection; the pasted command only installs.

## Considered Options

- **Keep the tunnel and add the runner beside it as a choice.** Rejected by the owner: two ways to pair is two
  flows to keep working, and the tunnel's Cloudflare precondition is the thing to remove.
- **Multiplex streams over the control WebSocket**, with a stream multiplexer library or our own framing.
  Rejected for the reasons above.
- **Issue a long-lived bearer on the computer** (`t3 auth session issue --ttl …`) instead of exchanging a
  one-time token, to avoid re-pairing. Rejected: it moves a long-lived credential across the wire and forks the
  pairing path; automatic re-pairing already makes the 30 days invisible.
- **A second binary for computers.** Rejected: enrollment, credentials, self-update and self-removal are the
  runner's already.
- **A systemd user unit with lingering, installed without sudo.** Rejected: lingering needs root on some
  distributions, so the runner would stop at logout on exactly those. With sudo always present, a system service
  that runs as the person starts at boot with nothing to configure.

## Consequences

- Every computer needs the runner running. A sleeping laptop is offline, as it was with `cloudflared`, and a run
  aimed at an offline computer fails saying so rather than moving to another of the person's computers.
- T3 Code's macOS service is a LaunchAgent, which launchd runs only while its person is logged in at the screen. A
  Mac's runner is connected from boot, but its T3 Code, and so its runs, wait for that login.
- `internal/install` gains an install on three OSes that puts a system service under the person's own user: a
  systemd unit, a LaunchDaemon, and on Windows a service or, if a service under a named account is not workable,
  a per-user logon task. A revoked runner removes itself through a small root-owned helper that only undoes the
  install, because a process running as the person cannot delete a root-owned unit.
- The retired tunnel routes stay mounted and answer 410, because ADR 0082 never removes a route.
- ADR 0074's enrollment codes gain a personal kind that any signed-in person may mint for their own computer,
  with no `runners:write`. ADR 0073's "runners run as root" holds for deploy runners only. ADR 0031's connection
  carries harness dial requests, while the bytes ride their own sockets. ADR 0102's "always one of their own"
  computer gains shared computers once sharing ships.
- The settings page called T3 Code Setup is renamed Computers; "personal runner" stays in code and `CONTEXT.md`.

Decided 2026-10-10.
