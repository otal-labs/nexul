# 15 — How a remote Nexul connects to a user's harness

**Type:** grilling
**Status:** resolved
**Blocked by:** 14

## Question

Nexul runs remotely; T3 Code runs on each user's own machine, usually behind
NAT with no public address. Which connection model does pairing use?

- Nexul dials the harness (today's model), with the user exposing T3 Code
  through a tunnel or private network.
- The user's machine dials out to Nexul — the runner pattern — through a
  small relay (a runner-style binary or the desktop app) that forwards to
  the local T3 Code, so no port is ever exposed.
- Something T3 Code already provides, if ticket 14 finds one.

Also: what "pair a computer" looks like for a new user under the chosen
model, how presence (Connected / Not connected) is derived, and how this
changes the setup wizard's first step. Likely an ADR — it reverses how
pairing reaches a harness.

## Answer

Grilled with the owner 2026-09-24. Options walked and set aside: a relay
through the browser tab (dies with the tab, useless from a phone); T3
Connect itself (T3's own apps only); a Nexul-built relay over one outbound
WebSocket with yamux (sound, but new transport code); Tailcat with or
without a self-hosted DERP relay (solves both-sides-behind-NAT, which Nexul
does not have, at the cost of a young library with no wire-format promise, a
heavy dependency tree, and a UDP port on the server).

**Decision: copy what T3 Connect does, on the instance's own Cloudflare.**

- Each paired computer runs `cloudflared` as a background service with a
  tunnel Nexul creates in the instance's own Cloudflare account, through the
  Cloudflare connection Nexul already has. The owner accepts both costs: the
  instance must have Cloudflare connected, and each computer gets a public
  hostname.
- **Hostname**: the computer's name slugged, plus eight random characters,
  on the instance's domain — "Onik Laptop" becomes
  `onik-laptop-<8 random>.<instance domain>`, as T3 Connect does. The random
  part stops anyone guessing another user's hostname.
- **Locked to Nexul**: a Cloudflare Access rule on each computer's hostname
  admits only requests carrying a service token that only the Nexul server
  holds; T3 Code's own pairing auth still sits behind it.
- **Pairing today's way over the tunnel**: Nexul dials the hostname exactly
  as it dials a T3 server URL now, so the T3 client and harness code barely
  change, and HTTP and WebSockets ride the tunnel natively.
- **Harness-neutral**: one tunnel per computer routes one hostname per
  local harness — T3 Code now, opencode's local server later.
- **Pairing flow, first step is the tunnel** (the owner, mirroring
  Cloudflare's own dashboard): the dialog shows the install-and-run command
  for the user's operating system (`cloudflared` installed as a service with
  the computer's tunnel token), waits until Cloudflare reports the
  connector online and Nexul reaches the hostname, and only then moves on —
  to pairing T3 Code over the verified hostname, then to the setup wizard
  (MCP connection, skills, per-provider confirmation).
- **URL pairing stays as an Advanced option** for machines the server can
  already reach (a VPS, a LAN machine, Tailscale).
- **Credentials**: the tunnel token lives only in the computer's
  `cloudflared` service; the Access service token only on the server; the
  per-computer MCP token only in the harnesses' configs. Removing the
  computer deletes its tunnel, hostname, and Access rule and revokes the MCP
  token.
- Set aside for later: a Nexul-built relay, if an instance without
  Cloudflare ever needs laptop pairing; Tailcat with a self-hosted DERP
  relay, if either end ever sits behind NAT with no public address.

Facts still to verify are in ticket 16; if any fails, this ticket reopens.
An ADR records the decision when the map is sliced.
