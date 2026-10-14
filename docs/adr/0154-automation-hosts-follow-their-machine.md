# Automation hosts follow their machine, and automations are workspace actions

Status: accepted.

Amends ADR 0151 (automations left as instance-area bits on roles).

An automations host runs automations' code on a machine, but it was checked with `automations:*`, the bits that also
let someone edit their own workspace's automations. Any role that could edit one workspace's automations could list,
add and remove every host on the instance, and place an automation on a host on any machine.

Decision: a host is part of its machine, like a runner.

- **Seeing a host** takes See on its machine; adding one on a machine or removing one takes Manage there. A host on a
  machine Nexul doesn't know yet is the instance Owner's: a host doesn't add machines.
- **Placing an automation on a host** takes Deploy to on the host's machine. The bundled instance host stays open.
- **`automations:*` are workspace actions.** The "automations" instance area and its rules go; roles keep the bits,
  which now reach only their workspace's automations.

Consequences: no new action, and hosts and runners answer the same question the same way. Someone who could manage
hosts through a role before the upgrade needs a rule on the machine instead.
