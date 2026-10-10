# 11 — Update T3 Code from the computer row

**Status:** needs-triage

**Blocked by:** 10, and the owner's answer to the spec's open question 2

Read first: `practices/go.md`, `practices/react-guide.md`, ADR 0113, the spec (open question 2).

## What to build (as recommended, pending the owner)

An "Update T3 Code" action on a computer whose T3 Code is a command-line install: a typed frame that runs
`t3 update --yes`, then reports the new version. Never automatic; never on the desktop app, which updates
itself. A turn running on the computer when the update lands ends with ADR 0113's "T3 Code was updated
during this turn" message.

## Acceptance criteria

- [ ] Decided by the owner first.
