# 26 — Images travel to the harness

**What to build:** When a run inlines a memory, or carries a ticket or doc body, that contains images, the images reach the harness as attachments: the markdown keeps the image's name in place and the stored bytes are handed over through the harness attachments list. Images over the per-attachment limit and non-image attachments are replaced by an "[attachment omitted: name]" note. Chat turns in ticket and doc threads get the same treatment for the target body.

**Blocked by:** 22

**Status:** done

- [ ] A memory with an embedded image, selected for a run, produces one attachment on the fake harness with the right name and MIME and the markdown referencing that name
- [ ] An oversized image and a PDF attachment each produce the omitted note and no attachment
- [ ] Against a real T3 harness, the Agent can describe the image
- [ ] The per-attachment and per-run limits sit next to the existing prompt limit and are covered by tests
