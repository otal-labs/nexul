# 17: Delete a tunnel, refused while hostnames use it

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- The server refuses while any hostname or gateway uses the tunnel, naming them. Once it is free, it also removes the
  CNAME.
- The web adds Delete and shows the refusal.
- Update `topology-and-dns.md`.

## Acceptance criteria

- [x] Use-case tests for the refusal and the CNAME cleanup.
- [x] Web test for the refusal message.

## Answer

- `DeleteTunnel` refuses while a gateway runs through the tunnel, a hostname is exposed on one of its gateways, the
  tunnel serves the instance's own address, or its cloudflared stack still exists. The refusal is `apperrs.InUseError` (thing, users, what to do first): it matches `ErrConflict`, so HTTP answers 409 and
  MCP passes the same text through. The machine delete reuses it.
- A free tunnel is deleted at Cloudflare, then its CNAME, then the local row, so a failed record delete leaves the
  row for a retry. The delete publishes the existing `dns.tunnel_changed` and `dns.record_changed` events, and the
  existing followers refetch the lists; no schema changed.
- Settings → DNS gets a Tunnels card with Delete (hidden without `dns:delete`) and a confirm naming the tunnel and its
  route. The refusal shows as the toast. `topology-and-dns.md` has a Tunnels section.
