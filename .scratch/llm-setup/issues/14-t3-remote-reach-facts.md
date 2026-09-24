# 14 — How a remote Nexul can reach a user's T3 Code

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Pairing today stores a **T3 server URL** and Nexul's server dials it
(`harness.Client` connects out to T3 Code's WebSocket). That works when
Nexul and T3 Code share a machine, as in the owner's local testing. A real
install runs Nexul on a remote server, which cannot reach
`localhost` on a user's laptop. Gather the facts a connection decision needs:

- **T3 Code's own remote story**: does `t3 pair` / T3 Code's server support
  remote access natively — binding beyond localhost, a relay or tunnel
  service, auth for remote clients, a documented "connect from another
  machine" path? Read pingdotgg/t3code (GitHub MCP, never clone).
- **Nexul's runner pattern**: runners are a small host binary that dials
  out to the Nexul server over a WebSocket so the server holds no keys.
  How is that connection authenticated and multiplexed, and could the same
  channel relay traffic to a local T3 Code?
- **Nexul's desktop app**: an Electron shell on the user's machine that
  imports a connection token. What does it hold and run today — could it
  host an outbound relay to the local T3 Code?
- **Tunnels**: what exposing T3 Code through a tunnel (Cloudflare Tunnel,
  Tailscale) involves, and the security cost of putting an
  agent-execution endpoint on a reachable address.

Findings to `research/t3-remote-reach.md`.

## Answer

Full findings: [research/t3-remote-reach.md](../research/t3-remote-reach.md)

- **T3 Code's own remote options**: pairing on a reachable address
  (`t3 serve --host`, or "Network access" in its desktop app); Tailscale
  HTTPS (`t3 pair --tailscale`); SSH-launched servers (wrong direction for
  Nexul); and **T3 Connect**, a hosted relay giving each machine a Cloudflare
  Tunnel — but it admits only T3's own apps as clients, with no path for an
  outside server. **T3 Connect is a blocker for Nexul.** Whether a plain
  `t3 pair` token works against a T3 Connect hostname is untested.
- `t3 pair` yields `<base>/pair#token=…`, single-use, 5-minute default TTL.
- **Tailscale and a reachable address work with today's code** (pairing
  accepts any http(s) URL), but every user must share a tailnet with the
  server, and container routing into a tailnet is unverified.
- **A user-run Cloudflare Tunnel works unchanged** but exposes an endpoint
  that runs shell commands, guarded only by T3's own login, unless Nexul
  learns to send Cloudflare Access headers.
- **The runner channel can't be reused as-is**: one instance-wide secret,
  small JSON job frames capped at 64 KiB, one job at a time. The
  automations connection (a token per automation) is the closer pattern.
- **A relay must carry several two-way streams per computer at once** —
  Nexul opens a fresh WebSocket per list or turn call plus a held presence
  socket. `t3client` already accepts a custom HTTP client, so relayed
  traffic plugs in without touching the T3 protocol code; multiplexing needs
  new code or a new dependency.
- **The Nexul desktop app** is a thin Electron window whose token holds only
  server info; it quits on window close on Windows and Linux and has no tray,
  so a relay there drops whenever the window closes.
- Nexul's Cloudflare tunnel API calls are reusable; its `cloudflared`
  launcher assumes a Docker runner on the target machine.
