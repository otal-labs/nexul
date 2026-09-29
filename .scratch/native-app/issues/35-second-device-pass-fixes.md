# 35 — Fixes from the second device pass

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** tickets 11, 12, 34

## What to build

The second pass confirmed every ticket 34 problem fixed and found these.

**E. Inbox and notifications (server and phone)**
- The Inbox ignores the workspace switch: notifications carry no workspace,
  so another workspace's items show and open onto an empty board.
- The server notifies you of your own status change and doc edit (web too).

**F. Errors and states**
- A missing doc or ticket spins about seven seconds (a 404 retried three
  times), then shows the raw server error with the id doubled ("get doc X:
  get doc X: not found"); the error dot touches the screen edge and a
  missing ticket gets a blank title. Never retry 4xx; show a plain "not
  found" state; stop the double wrap in the docs use-case or repository.
- The offline banner covers the last row of a list.

**G. Phone polish**
- Sign-out confirms (one device, everywhere else, sign out) use the stock
  Android alert with teal buttons, and the one-device confirm says "this
  device" about another; use a Mono Console confirm sheet with the right
  wording.
- Chat thread rows all read "Ticket thread"; name the ticket or doc.
- The markdown horizontal rule is a hardcoded light grey in both themes; use
  the border token.
- Chat and doc images keep a 4:3 frame instead of the image's own ratio.
- The Board and Docs project pickers do not mark the current project, and
  the Docs picker omits project prefixes.
- A compose stack reads "needs a running image; build stacks deploy from the
  web"; say what the phone can and cannot do plainly.
- The profile avatar gets an empty image URI; show initials instead.
- The Mine "on" state is barely visible.
- The deploy screen is titled "Deploy"; name the deploy.
- The chat list logs a `recycleItems` warning.
- The keyboard with the chat composer was not verified with a visible
  on-screen keyboard; verify on a device.

## Acceptance criteria

- [ ] Each problem reproduced before and gone after, on a device where visible
- [ ] Native, web and Go gates green
- [ ] A third pass finds none of the above

**Size:** M
