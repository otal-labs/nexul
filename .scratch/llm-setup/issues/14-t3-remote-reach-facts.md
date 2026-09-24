# 14 — How a remote Nexul can reach a user's T3 Code

**Type:** research
**Status:** claimed
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
