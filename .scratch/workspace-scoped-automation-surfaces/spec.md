# Runners, automations, and integrations are still instance-wide

**Status:** needs-triage

Carried over from the retired workspace requirements document, which said these
three move "from implicit instance-level to explicit workspace-level scoping",
and gave the reason for integrations plainly: so their events and permissions
cannot mix unrelated workspaces. The workspace split shipped; this half of it
did not, and it was never recorded as a deliberate no. It needs a decision
before it is sliced into tickets — either build it, or write down why one
instance-wide pool is the right answer after all.

## Problem Statement

An instance hosts many workspaces. Docs, tickets, projects, categories, board
configuration, chat, and membership all stop at the workspace boundary.
Runners, automations, and integrations do not — none of `runners`,
`automations`, or `integration_installs` carries a `workspace_id`, and every
one of them was built when a single implicit workspace was the whole product.

The consequences are not hypothetical:

- An automation subscribes to a topic, not to a workspace, so it receives
  `ticket.finished` for every workspace on the instance and acts on all of
  them. Its scoped token is minted against the instance, so there is no
  boundary underneath to stop it either.
- An integration installed by one workspace's owner receives signed webhooks
  for events raised in workspaces they may not be a member of, and its scoped
  token reaches those workspaces' API.
- A runner is visible and targetable from any workspace, so one workspace's
  deploy can occupy another workspace's build capacity, and a machine
  belonging to one team appears in another team's topology.

None of this is visible in the UI, because the switcher makes the instance
look partitioned. That is what makes it worth deciding rather than leaving:
somebody will invite a second team into a second workspace expecting the
boundary to hold.

## Solution

Give each of the three an owning workspace, matching the layer Docs and
Automations were said to sit at: one level above Project, not project-scoped.

- `runners`, `automations`, and `integration_installs` gain a `workspace_id`.
- Event delivery filters on it: an automation and an integration receive only
  events raised inside their own workspace, which means the event envelope has
  to carry the originating workspace and every publisher has to set it.
- Listing, creating, and editing each of the three is scoped to the selected
  workspace, as projects already are.
- Scoped tokens (`int_`, `dat_`) are minted against a workspace, so a token
  cannot reach outside the one that issued it.

The alternative worth pricing before building any of it: keep one instance-wide
pool and state that a Nexul instance is a trust boundary and a workspace
is only an organizational one. That is a legitimate answer for a single-tenant,
self-hosted product, and it is much cheaper — but it has to be written down,
because the switcher currently implies the opposite.

## User Stories

1. As a workspace owner, I want an automation I install to react only to my workspace's events, so that it never acts on work belonging to another team on the same instance.
2. As a workspace owner, I want an integration I install to receive webhooks only for my workspace, so that installing Discord in one workspace does not stream another workspace's deploys into my channel.
3. As a workspace owner, I want an integration's scoped token to be unable to read another workspace, so that the consent I gave is bounded by what I can see.
4. As a workspace owner, I want to see only my own workspace's runners and machines, so that the topology reflects my infrastructure rather than the instance's.
5. As a workspace owner, I want my deploys to queue against my own runners, so that another workspace cannot exhaust my build capacity.
6. As an instance admin, I want to know which workspace every automation, integration, and runner belongs to, so that offboarding a team is a bounded operation.
7. As an instance admin, I want a default automation to still ship with every workspace, so that scoping does not mean setting the same thing up repeatedly.
8. As a developer reading the event envelope, I want the originating workspace on every event, so that a consumer can filter without reaching into each payload's shape.

## Implementation Decisions

- The three scope to **workspace**, not project: they were never project-scoped
  and nothing has asked for that. This mirrors where Docs landed.
- The originating workspace belongs on the **event envelope**, not in each
  payload. Payloads are a published, additive contract (ADR 0044); adding a
  workspace field to every topic separately would be a wider change and would
  let publishers disagree about where it lives.
- Filtering happens at **delivery**, not inside each subscriber. The EventBus
  seam (ADR 0013) is the one place that knows a subscriber's workspace, and a
  subscriber that has to remember to filter is a subscriber that will forget.
- A scoped token is minted against a workspace, so the existing permission
  check (`<domain>:<action>`, ADR 0010) gains a workspace dimension it does not
  have today rather than a parallel mechanism.
- No migration or backfill: there is no production data, so this is a clean
  schema change, the same way the workspace split itself was.
- Default automations ship per workspace rather than per instance, so creating
  a workspace seeds them the way it already seeds status columns.
- This is deliberately independent of the deferred permission sweep (ADR 0023).
  Scoping is about which rows an actor can reach at all; the sweep is about
  which actions they may take on the rows they can reach. Doing the sweep first
  would apply a grid over an unbounded set.

## Testing Decisions

- An integration test against real SQLite is the main seam, because the
  question is which rows come back for which workspace, and a fake repo would
  prove nothing. Prior art: the existing storage integration tests.
- Event delivery: an event raised in workspace A reaches a subscriber in A and
  does not reach an identically subscribed one in B. This is the test that
  actually encodes the decision.
- Token boundary: a scoped token minted in A is rejected against B's API, for
  every action in the grid rather than one sampled endpoint.
- Listing: each of runners, automations, and integrations returns only the
  selected workspace's rows, and returns empty rather than erroring for a
  workspace with none.
- A regression test that the event envelope carries a workspace on every
  published topic, so a new topic cannot ship without one.

## Out of Scope

- Cross-workspace automations or integrations — one deliberately reaching
  several workspaces is a later question, and a harder one.
- Per-project scoping of any of the three.
- The deferred app-wide permission sweep (ADR 0023).
- Sharing a runner between workspaces on purpose, which is the obvious first
  request once this lands and should be designed on its own.

## Further Notes

If the decision goes the other way — one instance-wide pool stays — the outcome
is still a write: an ADR saying a workspace is an organizational boundary and
not a trust boundary, and a note in the UI where the switcher implies
otherwise. The thing that must not survive triage is the current state, where
the boundary looks like it holds and does not.

Sequencing, if it is confirmed: the event envelope's workspace field first
(everything else depends on it and it is additive), then automations, then
integrations, then runners — runners last because queueing and dispatch are the
only part with live behaviour attached.
