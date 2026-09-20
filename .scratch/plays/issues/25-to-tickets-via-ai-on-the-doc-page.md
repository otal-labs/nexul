# 25 — To tickets via AI on the doc page

**What to build:** The doc header gains a Plays menu beside the Thread button; each doc play is a row with its label and description, and the disabled reason shows once at the top when the harness is not ready. Choosing one opens the same run dialog without move-to. The run lands in the doc thread, and a Trail section on the doc lists the doc's trails; both are visible only with `docs:thread` on that doc. The seeded "To tickets via AI" run creates backlog tickets linked to the doc.

**Blocked by:** 19, 23

**Status:** done

- [ ] A doc's Plays menu lists doc plays only, honouring exclusions and readiness
- [ ] Running To tickets via AI against the fake harness posts Started and the reply in the doc thread; against a real harness, tickets appear in the project's backlog with the doc link
- [ ] A user without `docs:thread` sees the doc, no Plays menu, no Trail section
- [ ] The Trail rows open the same trail view as on tickets
