# 03: Locked doc and ticket mentions on the phone

**Status:** done

**Blocked by:** None. Can be triaged independently.

## Problem

`native/src/models/Doc.tsx` turns doc and ticket mentions into ordinary markdown
links. `native/src/components/chat/MessageMarkdown.tsx` opens those links.
The web's locked, inert reference content is absent on the phone.

## Scope

Apply the locked-reference rule in ADR 0148 to phone doc and ticket mentions.
Keep unreadable references inert while showing their full content with a lock,
as the approved mention-pill behavior requires.
Readable references keep working. This concerns references inside text, not
write-action controls in role-aware UI ticket 14; coordinate without duplicating
that sweep.

## Acceptance criteria for triage

- Identify each phone doc and chat rendering path for doc and ticket mentions.
- An unreadable target renders locked content and cannot be opened by tapping it.
- A readable target remains navigable, and access changes update the reference.

## Answer

Done 2026-10-14. The phone draws doc and ticket mentions in one place: `DocBody`,
which renders doc bodies and ticket descriptions. The phone has no ticket comments,
chat messages carry links rather than mention nodes (the web shows those as plain
pills too), and notifications name their subject without rendering mentions.

- `DocBody` resolves the body's mentions through `/api/mentions/resolve`, the call
  the web makes, in one batch. Its `refreshes` names `access.grant.changed`, and a
  change to the viewer's own access refetches every open read.
- A target the viewer can open stays a link into the app. Any other target, or one
  still unresolved or gone, shows the same text the web pill does, followed by a lock
  image whose alt text is "You don't have access". It renders as plain text, so
  tapping it does nothing.
- The ticket text follows the workspace's mention template. The template rule and
  the resolve wire shapes now live in `client-core/mention.ts`, which both apps use.
- The markdown renderer accepts only images inline, so the lock is lucide's lock as
  a PNG for each theme's muted foreground. `web/scripts/generateIcons.ts` generates
  both images from `native/src/global.css`.

Not covered: a stored body that links to a doc or ticket page in plain link text,
not as a mention. The web converts those to pills in its editor; the phone still
shows them as links.
