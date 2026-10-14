# 14: Sweep the phone app

Type: task
Status: resolved
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/native.md`, `practices/react-guide.md`, `practices/testing.md`, the spec.

## What to build

Move the phone's writes onto its action components (research 01's phone table): ticket status, assign to me, and
the rest.
- A read inbox row gains Mark unread through `POST /api/notifications/{id}/unread` (a swipe or an accessibility action, beside the one that marks it read).

## Acceptance criteria

- [x] The phone tests show hidden and locked states.
- [x] Checked on the emulator in the T3 Device panel.

## Answer

Swept: ticket status and Assign to me on the ticket screen, the board's card moves (hold-and-drag and "Move to"), Cancel deploy (row and confirm sheet) and the Redeploy confirm sheet. Each goes through `ActionButton` or `useActionState`; the hand-written `editDeploys` check is gone.
A read inbox row gains Mark unread, as a swipe and an accessibility action, through `POST /api/notifications/{id}/unread`.
Checked on the emulator as a read-only role: locked buttons with their popover, the hidden state after the role's switch went off (live, through the push), and Mark unread by swipe. The popover opened at the control's right edge and ran off the screen for a control at the left, so it now starts at the control's left edge and slides in only as far as the window needs. A locked status button drops its chevron and is named "Status, <name>".
Skipped: the status picker sheet itself (reachable only from the gated button; the server refuses the write), the clarification forms (the server's own `can_answer`), and the chat composer's `channels:post` (a per-channel answer the phone doesn't read yet). The phone's workspace `/me` follows role pushes, so the instance-only foreground refresh matters to none of these controls.
