# The Agent page holds where runs go, and a shared computer joins your order

Amends ADR 0197 (its "no shared computers for auto plays") and ADR 0210 (its "shared computers stay out of a person's
order").

Where a run lands was read across three places: Plays in the sidebar for plays and their auto plays, Settings →
Computers for your order and the computers shared with you, and its Projects tab for which checkout a project uses. Each
answered a part, and none said where a given play would go. ADR 0197 makes one order the only answer; this decides where
that answer is shown, and closes the gap 0197 left for a computer someone shared with you.

Decision:

- **One Agent page.** The sidebar's Plays becomes **Agent**: one list in groups (Plays, Your computers, Shared with you,
  Projects) beside a detail pane. A play's detail shows where it lands now; a computer's shows what lands on it, its
  commands and its setup; a project's shows its computers and checkouts in order. Settings → Computers moves there and
  leaves a link. The name is `CONTEXT.md`'s Agent: the page holds what the Agent does by itself and where it runs.
- **Takes plays.** A computer its owner switches off for plays leaves their order for plays, auto plays and @Agent turns,
  and keeps commands, logs and setup.
- **A shared computer joins your order.** A computer shared with you with Run agents can be added to your order at any
  position and takes your runs as a guest run (ADR 0210). It leaves your order by itself when its owner takes Run
  agents away or switches Takes plays off. Commands-only sharing never puts a computer in anyone's order.
- **Moving over keeps shared auto plays running.** An auto play whose list named shared computers becomes Run as its
  picker, with those computers added to the picker's order after their own, instead of being switched off.

The trade-off: a person's order can now hold a computer they don't own, so a run may land on a client's machine without
being picked for it. That is what the owner's Run agents grant says they allow, the owner sees every such run under Runs
by others, and taking the grant away takes the computer out of the order at once.

Rejected: tabs for Plays, Auto plays, Computers and Runs on one page, which keeps today's split under one header; a
play-by-computer matrix like the roles grid, which reads well at four computers and not at ten, and has nowhere for
commands or setup; keeping Settings → Computers beside the new page, which is two places to keep in step.

Decided 2026-11-25. Amended 2026-11-27: the Your computers, Shared with you and Projects groups moved out of the Agent
page's list into a Computers dialog that opens from any page; the page's list holds plays.
