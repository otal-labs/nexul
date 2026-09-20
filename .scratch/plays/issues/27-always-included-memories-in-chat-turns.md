# 27 — Always-included memories in chat turns

**What to build:** An `@Agent` mention in any conversation inlines the project's always-included memories in full, the same way a play run does, so chat and plays never disagree about the standing rules. The memories index still lists the rest by title and when-to-use. The same per-run ceiling applies, and the fixed Agent instructions explain the two tiers.

**Blocked by:** 16

**Status:** done

- [ ] A mention in a ticket thread sends the "Working in this project" body in full; a mention in a channel with no project sends none
- [ ] The prompt text explains always-included memories versus the index
- [ ] Over-ceiling always-included memories are trimmed with a note rather than failing the turn, and the trim is logged
