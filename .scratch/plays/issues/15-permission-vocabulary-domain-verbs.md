# 15 — Permission vocabulary gains domain verbs

**What to build:** The permission table gains a `plays` domain with read, write, delete, run; a `memories` domain with read, write, delete, clone; and a `thread` verb on docs. The roles grid, personal token scopes, and integration and automation token scopes render and accept them; write grants read at mint as today. Nothing gates on them yet. Implements ADR 0057.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] The catalog endpoint lists the new actions with display names and the web grid shows them without hardcoding
- [ ] A role and a scoped token can hold `plays:run`, `memories:clone`, `docs:thread`, and the permission check recognises them
- [ ] The Owner bypass still covers them; a fresh workspace seeds Owner only, no other grants
- [ ] The table's tests enumerate the new rows
