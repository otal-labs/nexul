# 05 — Run on chosen computers, in order

**Status:** ready-for-agent
**Blocked by:** 02

Spec: `../spec.md`, decision 11. ADRs 0156, 0165.

`auto_plays.computer_id` becomes an ordered list, every computer checked against the picker's rights as today; the
`ComputerPicker` becomes the same ordered picker the Projects tab uses, and `play_create`/`play_update` take
`computer_ids`.

Done when: a stack play with two chosen computers runs on the second while the first is offline.
