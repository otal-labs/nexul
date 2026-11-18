# 03 — Computer labels

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, Placement, Needs; Moving over; Surfaces.

`pairing_computers.labels`, set by the computer's owner: `PATCH /api/pairing/computers/{id}` and `computer_pair` take
`labels`, `GET /api/pairing/computers` and `computer_list` return `labels` and `derived_labels` (`docker` from Docker
being available, `linux`/`macos`/`windows` from the OS). Validation as the spec says; a bad label is `invalid` with the
rule. A change publishes `computer.placement_changed`. The Computers tab edits them on the computer's row and shows the
derived ones beside them, read only.

Done when: use-case tests cover validation and the derived set, the migration test runs, and the Computers tab edits a
label at 768, 1024 and 1440px. `paired-computers.md` documents labels; `CONTEXT.md` defines label.
