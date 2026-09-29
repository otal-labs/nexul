# 34 — Fixes from the first device pass

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** tickets 11, 12

## What to build

The first end-to-end pass on an emulator (all of tickets 24–33 merged) found
these problems. Fix them in four batches that touch separate files, each with
before and after screenshots on a device, and tests only where they guard a
real regression.

**A. Navigation (layouts under `native/src/app/`)**
- Changing the theme in Your settings → Appearance crashes with the native
  error "ScreenStackFragment added into a non-stack container" and a blank
  screen until relaunch.
- Opening a ticket from Inbox, or "Open thread" from a ticket, makes that
  screen the root of the Board or Chat tab: no back arrow, and the list is
  unreachable until restart. The target tab must keep its list underneath.
- Docs, a doc, and Runners show a doubled header (an outer lowercase bar with
  a second back arrow).
- Pickers and confirms open as untitled full-height sheets; each needs a
  title naming what it acts on and a fitted height.
- Screen titles are generic ("Ticket", "Stack", "Ticket thread").

**B. Content**
- No image loads in chat or docs: Android sends the image request without
  the `Authorization` header and the server answers 401
  (`components/chat/MessageImage.tsx`). The file route stays private.
- The keyboard covers the chat composer.
- Doc links: mention chips render as untappable code text, and
  `/docs/<projectToken>/<id>` links fail; doc and ticket mentions and both
  doc URL shapes must open in the app.

**C. Visual polish**
- Icons coloured through `className` are invisible in dark mode (Board
  chevron, picker check marks); pass `color` from the tokens.
- The `small` text variant clips descenders.
- Relative times freeze at first render; drive them from one app-wide
  minute clock kept in state.
- Board rows show unlabelled dots; show type and label names.
- Touch targets under 44dp (Mine, Mark all read, the device sign-out X).
- "Assign to me" shows when already assigned; long workspace names overflow
  Your settings; destructive buttons use a muddy red in dark instead of the
  token; the profile sign-in row is blank for an unknown provider.

**D. Workspace scoping and dead ends**
- Switching workspace leaves the old workspace's ticket, thread, and log on
  screen; every tab returns to its root on switch.
- Deploys lists stacks from every workspace; scope to the current one (an
  additive server filter if the API has none).
- The "server too old" screen offers only Retry; add Sign out or "Use a
  different server".
- Redeploy is offered while a deploy is running, and a 409 shows raw.
- Server: creating a ticket without `type_id` returns 409, and a ticket
  created through the API gets status `open`, which matches no board column
  and is invisible on web and phone. Fix at the use-case so HTTP and MCP
  agree.

## Acceptance criteria

- [ ] Each problem reproduced before and gone after, on a device
- [ ] `bun run lint`, `bun run typecheck`, `bunx jest --ci` in `native/`; Go gates for the server part
- [ ] A second device pass finds none of the above

## Read first

`practices/native.md`, `practices/react-guide.md`, `practices/design-language.md`,
`practices/testing.md`, and `practices/go.md` for the server part.

**Size:** L
