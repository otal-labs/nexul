# 05 — Automation secrets by reference on the automations host

**Status:** open
**Type:** grilling
**Blocked by:** 01, 02, 07

## Question

How does an automations host get its 1Password token (bundled host and a placed host, ADR 0074), and
when does it resolve a pool entry that is a reference: when the automation starts, on each
`ctx.secrets.NAME` read, or cached for a time? What does an automation see when a reference cannot be
resolved, and how does the pool's write-only behaviour (ADR 0047, ADR 0095) read once an entry is a
reference rather than a value?
