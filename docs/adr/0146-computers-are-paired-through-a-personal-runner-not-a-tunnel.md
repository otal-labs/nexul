# Computers are paired through a personal runner, not a Cloudflare tunnel

Status: proposed, with the personal runners effort (`.scratch/personal-runners/`).

Supersedes ADR 0062 and ADR 0142. Amends ADR 0031, ADR 0073, ADR 0074 and ADR 0102.

Nexul runs on a server; T3 Code runs on each person's own computer, usually behind a router. ADR 0062 reached it
through a tunnel in the instance's own Cloudflare account, one per computer, closed by an Access rule. That made
Cloudflare with Zero Trust a precondition for the main feature, made pairing three steps (a tunnel command, a
pasted pairing link, setup), and left a paste every 30 days because T3 Code's session cannot be refreshed. Pair by
URL, the other way, asked people to open T3 Code's port on every interface.

Decision: a person adds a computer by installing a **personal runner** on it with one command. The runner is the
existing `nexul-runner` in a personal mode: enrolled with a one-time code bound to that person and that computer,
running as the person's own OS user (a systemd user unit, a LaunchAgent, a per-user logon task on Windows), with
no Docker. It holds the runner's usual outbound WebSocket, and the server reaches T3 Code through it:

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
  the tunnel pairing code is deleted. Cloudflare stays for the instance's domain, gateways and exposures.
- **A personal runner is private to its owner.** It never takes deploy jobs, never shows in runner or machine
  lists, publishes members-only events that reach only its owner's sockets, and is checked by ownership rather
  than permission, so no role or Owner bypass reaches it. The owner may later share a computer with named
  people; disabling or removing an account revokes its runners.

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

## Consequences

- Every computer needs the runner running. A sleeping laptop is offline, as it was with `cloudflared`, and a run
  aimed at an offline computer fails saying so rather than moving to another of the person's computers.
- `internal/install` gains a user-level install on three OSes, including a Windows path that is not a service.
- The retired tunnel routes stay mounted and answer 410, because ADR 0082 never removes a route.
- ADR 0074's enrollment codes gain a personal kind that any signed-in person may mint for their own computer,
  with no `runners:write`. ADR 0073's "runners run as root" holds for deploy runners only. ADR 0031's connection
  carries harness dial requests, while the bytes ride their own sockets. ADR 0102's "always one of their own"
  computer gains shared computers once sharing ships.

Decided 2026-10-10.
