# Voice channels are conversations on a bring-your-own LiveKit, with two occupancy feeds

A voice channel is a conversation kind, not a new entity, so it carries its
own text messages with no extra modelling (Discord's text-in-voice comes free)
and inherits conversation lifecycle, listing, and permissions.

Nexul never hosts, deploys, or bundles the media server. The owner
supplies a LiveKit URL, API key, and secret through the connectors surface,
verified with a live call at save; LiveKit Cloud and self-hosted are the same
configuration. Media relay has an operational profile — UDP port ranges, TURN,
bandwidth — that has nothing to do with the rest of the product, and owning it
would make every install carry that cost for a feature most never enable. A
voice channel without a configured connector renders an inline "owner setup
needed" state, never a dead button. One LiveKit room per channel, named by
conversation id, created implicitly on first join and reaped by LiveKit with
its last participant, so there is no room bookkeeping to keep in sync.

**Occupancy has two feeds on purpose.** LiveKit webhooks update it in real
time, and a ListRooms reconciliation poll — running only while some client has
a chat surface open — corrects drift. Neither alone is sufficient: webhook
delivery is best-effort, and a private or localhost instance is unreachable
from LiveKit at all, where the poll is the only feed. Occupancy is ephemeral
state, published over the existing browser live-events socket rather than a
second transport, and never persisted.

Decided 2026-08-27.
