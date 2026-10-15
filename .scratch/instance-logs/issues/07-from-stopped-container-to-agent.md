# 07: How a stopped container reaches an agent without writing an automation

Type: grilling
Status: resolved
Blocked by: 05

## Question

`service.container_stopped` reaches automations, so an automation can already open a ticket
and call `runPlay`. But auto plays fire only on ticket and doc moments (`momentPlayType` in
`internal/plays/autoplay.go`), so nothing reacts out of the box. What does a team get without
writing code, and how is a person told?

- Does a stack get a switch, off by default, that opens a ticket in its project when a service
  stops? The ticket would name the outage and say to read it with `stack_get` logs. One open
  ticket per service: a repeat adds a comment, and recovery adds a "came back" comment without
  closing it. Its `ticket.created` auto plays (a "Fix with AI" play) then start an agent on it.
- If so: which ticket type and stage, who it is assigned to (an auto play runs on the developer
  by default, so a ticket with nobody gets "didn't run"), and who the switch's setting picks.
- Or instead a stack moment for auto plays: a third play type besides ticket and doc. That is a
  larger change to ADR 0132.
- How a person learns of it when no agent runs: a notification to the stack's project, the
  stack page's outage line only, or both.

## Answer

- **A stack switch opens the ticket, and stack auto plays start the agent.** Both, since they do
  different jobs: the ticket is the record a team works from, the play is what wakes an agent.
- **The switch: "Open a ticket when a service crashes", off by default,** on the stack's settings
  (`stacks:write`). It opens one ticket per crashed service in the stack's project, titled
  "api crashed: exit 137, out of memory", whose body names the stack, service, outage id and times
  and says to read the output with `stack_get` and `logs` for that service. While that ticket is
  open, a repeat crash adds a comment instead of a ticket, and a "came back" adds a comment and
  leaves it open, because a crash that stopped is not a crash that was fixed. The ticket takes
  the project's default ticket type, its first stage, no developer and no tester, and is created
  by the instance (`ticket.created` with no causer).
- **Stack plays are a third play type beside ticket and doc** (a successor to ADR 0132). Its
  moments are `stack.service_crashed` and `stack.service_recovered`, and a third,
  `stack.error_rate_crossed`, comes with error-rate alerts. Conditions read the stack, its
  project and the service name. The prompt names the stack and the outage by id, as ticket plays
  name a ticket (ADR 0111).
- **Run on: a chosen computer.** A stack has no developer or tester, so a stack play names the
  computer its runs land on, for example a box that is on all day. The choice is offered on every
  play type, not only stack plays. The run acts as the person who picked the computer, who must
  own it or hold Run agents (`computers:run_agents`) on its share; the picker lists only those
  computers. If that right is later taken away, the run says "didn't run" with the reason, as a
  play with nobody to run on does today.
- **A shipped play, "Investigate a crash".** It is in every workspace beside "Implement with AI",
  with an auto play on `stack.service_crashed`, off until someone sets its computer and turns it
  on. Its prompt reads the outage and the service's logs, finds the cause against the stack's
  repository, and writes it on the crash ticket if the switch made one (otherwise on the stack's
  project as a new ticket). It never deploys, restarts or edits the stack. It opens a fix pull
  request only if the play's own instructions are edited to say so. If it believes the fault is
  Nexul's own rather than the app's, it says so on the ticket and points at the triage skill.
- **Everyone who can read the stack is watching it by default.** A per-person Watching toggle on
  the stack page turns it off; this is the reverse of docs, where watching is opt-in. A stored
  `stack_unwatchers` row marks someone who opted out, so a new member starts watching. A watcher
  gets one notification when a service crashes (a restart loop is one) and one when it comes
  back. Agents read the same through `notification_list`. The stack page's outage line shows
  either way.
