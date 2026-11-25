# 09 — A shared computer joins your order

**Status:** ready-for-agent
**Blocked by:** 07, 08

Spec: `../spec.md`, decision 10; Moving over. ADR 0211, amending ADR 0210.

A person adds a computer shared with them with Run agents to their own order, at a position among their own computers,
and removes it again. Stored as a `pairing_order_shares` row (person, computer, position) in the same sequence as their
own computers' positions. Placement's candidates read the order with those rows in place; a row whose grant no longer
holds Run agents, or whose computer has Takes plays off, is skipped and named in a waiting run's reasons, and the grant
change that ends it publishes `computer.placement_changed`. A run placed there is a guest run (ADR 0210), exactly as a
pressed run on that computer is today.

Surfaces: the order endpoints take shared computers; `computer_list` returns them with their position;
`computer_pair`'s reorder accepts them. On the Agent page a shared computer's detail has Add to your order, and Your
computers shows it in its place, marked shared, draggable like the rest. The owner's Runs by others lists these runs.

Done when: placement tests cover a shared computer first in the order taking a run as a guest, the owner taking Run
agents away and the next computer taking it, and a commands-only share that can't be added; the order is seen at 768,
1024 and 1440px; `paired-computers.md`, the sharing section, says how.
