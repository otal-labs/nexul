# 08 — Takes plays

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decision 9 and Rules. ADR 0211.

`pairing_computers.takes_plays`, on by default, set by the computer's owner through `PATCH /api/pairing/computers/{id}`
and `computer_pair`, returned by `GET /api/pairing/computers` and `computer_list`. `place` passes a computer with it
off as `no_plays` before any other check, for pressed, queued and @Agent runs alike; commands, logs and setup are
untouched. Switching it on publishes `computer.placement_changed` so a waiting run tries again. A computer shared with
Run agents and Takes plays off takes no one's runs.

Done when: placement tests cover a computer switched off being passed over and the next one taking the run, and a run
waiting on the only computer starting once it is switched back on; the switch is on the computer's detail on the Agent
page at 768, 1024 and 1440px; `paired-computers.md` documents it and the reasons table names `no_plays`.
