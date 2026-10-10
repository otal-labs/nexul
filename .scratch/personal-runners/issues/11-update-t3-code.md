# 11 — Update T3 Code from the computer row

**Status:** wontfix

**Blocked by:** None

Read first: the spec (decision 21).

## What was proposed

An "Update T3 Code" action on a computer whose T3 Code is a command-line install: a typed frame that runs
`t3 update --yes`, then reports the new version. Never automatic; never on the desktop app, which updates
itself. A turn running on the computer when the update lands ends with ADR 0113's "T3 Code was updated
during this turn" message.

## Acceptance criteria

- None. Not built.

## Comments

wontfix: the owner decided the runner never updates T3 Code, neither automatically nor from a button (2026-10-10).
