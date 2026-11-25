# 07 — The Agent page

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decision 8. ADR 0211.

The sidebar's Plays becomes **Agent** at `/:ws/agent`, one `ListDetailLayout` whose list has four groups, each shown to
whoever may use it: **Plays** (`plays:read`; each with its auto plays count), **Your computers** (in your order, with
the ready dot), **Shared with you** (with what the owner allows), and **Projects** (the projects with a computer link).
The detail pane is today's `PlayDetail`, the computer's details from Settings → Computers (setup, commands, logs,
sharing, Runs by others), or the project's row from the Projects tab, unchanged in content: this ticket moves them, the
later tickets add to them. New play and Pair a computer sit in the list header.

`/:ws/plays/:id`, `/settings/pairing` and its `?setup=` deep link redirect to their place on the Agent page, so every
existing link, notification and guide link keeps working. Settings → Computers is a one-line link to the Agent page.
The page's chunk follows the web shell rule (`web/src/pageChunks.test.tsx`), and a gated group's data is not fetched for
a viewer who can't see it.

Done when: the router permission cases cover each group shown and hidden; the redirects are tested; the page is seen at
768, 1024 and 1440px with design review evidence; `paired-computers.md` and `plays.md` say where things are now, and
`CONTEXT.md` names the page.
