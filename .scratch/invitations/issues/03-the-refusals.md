# 03: Turn existing accounts away, and say plainly when a link is gone

Type: task
Status: open
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Invitation), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec (Decisions 4 and 5, Look: refusals,
Architecture).

## What to build

- `CompleteInvitationOAuth` and `PrepareAuthenticatedAcceptance` return an `existing_account` error code and leave
  the invitation unredeemed.
- `Redeem` refuses an existing user as a backstop.
- The existing-account screen and the expired-or-deleted screen.

## Acceptance criteria

- [ ] An existing account, whether signed in or coming through OAuth, is refused, and the same link then admits a new
  person.
- [ ] Expired and deleted links show the one message.
- [ ] Tests cover both server paths and the backstop. Verified at 390, 768, 1024 and 1440.
