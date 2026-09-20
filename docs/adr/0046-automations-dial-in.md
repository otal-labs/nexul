# Automations dial in to the instance; Nexul never calls out to them

An automation opens one outbound WebSocket to the instance, authenticated by
its own scoped token, announces the subscriptions declared in its code, and
receives events down that connection; run reports ride back up it. It is the
Runner's pattern, for the same reason: first-party code has to work
identically in the bundled automations host, on an owner's server, and on a
laptop behind NAT — none of which can host a reachable inbound URL.

The trade-off against the webhook path the same product already had: no
inbound endpoint, no signature to verify, and a per-automation cursor into the
event log so a reconnect resumes exactly where it left off instead of losing
whatever arrived during downtime. In exchange the platform carries connection
state, and delivery is strictly sequential per automation. Signed webhooks
remain what they were, the delivery path for third-party integrations — two
delivery paths over one event log.

Decided: 2026-08-27.
