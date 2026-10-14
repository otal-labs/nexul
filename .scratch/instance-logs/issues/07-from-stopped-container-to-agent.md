# 07: How a stopped container reaches an agent without writing an automation

Type: grilling
Status: open
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
