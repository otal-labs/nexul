# A stack play runs on a computer someone chose, as the person who chose it

Succeeds ADR 0132 for the third play type and for whose computer a run lands on. ADR 0091 is why a crashed service's
output is read live by the agent and never carried in an event.

A crashed service has no ticket, no developer and no tester, and nobody caused it, so none of the three people an auto
play could run on (ADR 0132) exists. Teams still want an agent to read the logs the moment a service goes down, on a box
that is on all day.

Decision: a play has a third type, **stack**, and every auto play may run on **a chosen computer**.

- **A stack play runs on a stack.** Its run target is the stack, so one stack has one active run at a time, the daily cap
  counts per stack, and a run's trail and thread belong to the stack: the thread is the stack's own conversation, one per
  stack, read through its project. Its moments are `stack.service_crashed` and `stack.service_recovered`, taken from
  `service.container_stopped` and `service.container_recovered` (ADR 0091); a later `stack.error_rate_crossed` joins them.
  A stack in no project, an instance stack, matches nothing. Conditions read the stack, its project and the service by
  name; the service travels with the queued run, since the stack alone cannot say which one went down.
- **The prompt names, it does not carry** (ADR 0111). It names the stack by name and id, and for an auto run the service
  and the outage, and tells the agent to read the rest with `stack_get`, passing `logs` with the service's name.
- **Run on: a chosen computer** is a fourth choice beside developer, tester and whoever caused it, offered on every play
  type. A stack play offers only this one; a doc play offers it beside whoever caused it. The auto play stores the
  computer and the person who picked it, and the run takes that person's identity: the trail is theirs, the permissions
  are theirs, and it takes their slot in the queue. Whoever saves the auto play with a new computer becomes that person;
  saving it with the same computer keeps who picked it.
- **The right is checked three times**: when the computer is picked, when the moment matches, and again at the front of
  the queue. The person must own the computer or hold `computers:run_agents` on its share (ADR 0148). If they no longer
  do, the run does not start and says "didn't run" with the reason, as a play with nobody to run on does; if someone else
  has picked the computer since, the queued run is skipped. A computer the picker cannot see reads as not shared at all.
- **Off until a computer is set.** An auto play that runs on a computer cannot be switched on without one, so the seeded
  "Investigate a crash" waits for a person to pick it.
- **"Investigate a crash"** is a shipped stack play with an auto play on `stack.service_crashed`, in every workspace like
  the other shipped plays. It reads the service's logs and the stack's recent deploys, diagnoses against the project's
  repository, writes the findings on the crash's ticket or a new one, and never deploys, restarts or edits the stack. It
  points a fault in Nexul itself at the triage skill.

The alternatives were running a stack play as the instance, rejected because every run would need an identity no person
holds and permissions nobody picked; running it as the stack's project owner, rejected because a project has no single
owner to stand in; and a per-service thread, rejected because a restart loop would open one per outage when the agent
reads the whole stack either way. The cost is that a play now spends someone's agent time with nobody pressing anything
and under their name; the queue, the "didn't run" line and the switch that stays off until a computer is set are how that
stays visible. A second service crashing while a run on the same stack waits or runs is covered by that run, since the
agent reads every service that is down.

Decided 2026-10-16, amending ADR 0132.
