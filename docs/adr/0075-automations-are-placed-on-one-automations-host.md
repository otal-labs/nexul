# Each automation is placed on one automations host, with a host-scoped token

Amends ADR 0046: automations still dial in, but a host now learns which automations to run from the instance.

Every automation runs on exactly one automations host. New automations go to the bundled host, named `instance`;
the automation page and `automation_update` (`host_id`) move one to another host. A host asks the instance for its
assignments with its own credential (ADR 0074) and gets back the enabled automations placed on it, each with the
token its worker dials in with. That token is host-scoped: the instance accepts it for that one automation only
while the automation is placed on that host and the host's credential is live. Moving an automation away or
removing the host makes the instance refuse the old host's token straight away, and the host stops the worker on
its next poll.

Why: the bundled host used to read a tokens file the server wrote next to its database, which only worked while
both shared a volume. A second host on another machine had no way to learn what to run, and nothing stopped two
hosts from running the same automation.

Rejected: letting any host run any automation it holds a token for. Two hosts would both answer the same events,
and a token copied off one machine would keep working after the host was removed. Also rejected: pushing
assignments to hosts over the dial-in connection. Each worker dials in per automation, so there is no host-level
connection to push on, and a poll with the host credential is enough at the rate placements change.

The trade-off: placement is a manual choice. There is no scheduling across hosts and no failover; an automation on
a host that is down does not run until the host is back or the automation is moved.

Decided: 2026-09-27
