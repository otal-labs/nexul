# 0057. A domain may declare a permission verb beside read, write, and delete

ADR 0010 fixed the permission vocabulary to `<domain>:<read|write|delete>`.
Plays broke it three ways at once: pressing a play button is neither
reading a definition nor editing one, cloning a memory into another
workspace is more than writing it, and seeing a doc's working thread is a
different act from reading the doc, one a client must be able to hold
without the other.

Decision: `read`, `write`, and `delete` stay universal, and a domain may
declare a further verb when the act is neither reading nor editing. Three
arrive with plays: `plays:run`, `memories:clone`, `docs:thread`. The Go
domain table already lists actions per domain, so the catalog, the roles
grid, and token scopes pick a new verb up without special cases.

The alternative, folding each act into `write`, over-grants: anyone who may
press a button could edit its definition, anyone who may edit a memory
could copy it into a workspace it was never meant for, and any doc reader
would see the team's work behind the doc. The cost is a vocabulary that is
no longer three words, so a verb must be declared, named, and documented
per domain rather than assumed.

Decided 2026-09-16, amending ADR 0010.
