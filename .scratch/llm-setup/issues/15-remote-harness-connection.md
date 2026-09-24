# 15 — How a remote Nexul connects to a user's harness

**Type:** grilling
**Status:** claimed
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
