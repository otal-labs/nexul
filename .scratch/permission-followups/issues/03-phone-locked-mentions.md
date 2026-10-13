# 03: Locked doc and ticket mentions on the phone

**Status:** needs-triage

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
