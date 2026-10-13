# 17: Delete a tunnel, refused while hostnames use it

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- The server refuses while any hostname or gateway uses the tunnel, naming them. Once it is free, it also removes the
  CNAME.
- The web adds Delete and shows the refusal.
- Update `topology-and-dns.md`.

## Acceptance criteria

- [ ] Use-case tests for the refusal and the CNAME cleanup.
- [ ] Web test for the refusal message.
