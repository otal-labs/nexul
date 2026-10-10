# 14 — Remove tunnel and URL pairing

**Status:** ready-for-agent

**Blocked by:** 12, 13

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md` (section 11),
`practices/react-guide.md`, ADRs 0062, 0082, 0142, 0146, the deprecation-and-migration approach, the spec
(Moving existing computers, removal list).

## What to build

Ships only after the owner's 30 days (spec, decision 20): at least 30 days after the release carrying ticket
12's notice, and either no tunnel or URL computer remains or every remaining owner has had the notice for 30
days (their inbox receipt). The old computers' rows, links and setup are kept.

- Delete: `CreateComputerTunnel`, `ComputerTunnelToken`, the tunnel watch, URL pairing in `pair`/`Repair`,
  `AccessTransport` in the harness client, `computerTunnelAccess`, `tunnel.sh` and `tunnel.ps1`, the Tunnel
  step, `PairT3CodeStep`/`PairingLinkField`/`DesktopPairSteps`/`PairCommands`, Pair by URL,
  `TunnelPrerequisiteAlert`, `computer_tunnel_token_get`, and `computer_create`'s `port` and
  `computer_pair`'s `token`/`server_url`/`name` (every reference updated in the same change, mcp.md
  section 11).
- A one-time, idempotent cleanup at boot: delete the Cloudflare resources of any computer still on a
  tunnel, then the instance's Access service token, then clear the tunnel columns.
- Keep the retired routes mounted, answering 410 with "Pairing through a tunnel was retired; add the
  computer with the Nexul app" (ADR 0082).
- Old computers' rows stay with their links and setup, showing "Add this computer with the Nexul app".
- `CONTEXT.md`: delete Computer tunnel, update Paired computer; ADRs 0062 and 0142 marked superseded by 0146.

## Acceptance criteria

- [ ] At least 30 days have passed since the release with ticket 12's notice, and production shows no tunnel
      or URL computer, or every remaining owner's inbox receipt is at least 30 days old.
- [ ] `rg -i 'tunnel'` over `internal/pairing`, `web/src/components/pairing` and the computer guide pages
      finds nothing that pairs.
- [ ] Upgrade test from the previous schema with a tunnel computer in it: the cleanup runs once and the
      row survives.
