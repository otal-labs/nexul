# 28 — The External secrets guide page, its links and the README row

**Status:** ready-for-agent

**Blocked by:** 18, 20, 22, 25, 26, 27

Read first: the spec (Hit every surface, Docs), `website/src/content/docs/docs/guide/`.

## What to build

- A new guide page, "External secrets": a service account per runner, host and server, writing a
  reference, the `nexul … 1password` commands, the secret check and its interval, and what each error
  means. Example vaults and items use neutral names.
- One line linking to it from the stack, automation, connector and runner guide pages.
- The README row turns ✅, and the release notes say an existing `op://` literal becomes a reference.

## Acceptance criteria

- [ ] The website builds, and the new page renders at 320, 375, 414 and 768px.
- [ ] Every command and setting in the page matches the built CLI's `--help` output.

Verify: the website's build and lint.
