# 14: Sweep the phone app

Type: task
Status: open
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/native.md`, `practices/react-guide.md`, `practices/testing.md`, the spec.

## What to build

Move the phone's writes onto its action components (research 01's phone table): ticket status, assign to me, and
the rest.
- A read inbox row gains Mark unread through `POST /api/notifications/{id}/unread` (a swipe or an accessibility action, beside the one that marks it read).

## Acceptance criteria

- [ ] The phone tests show hidden and locked states.
- [ ] Checked on the emulator in the T3 Device panel.
