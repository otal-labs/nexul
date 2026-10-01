# 14 — End to end on the test box

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 09, 11, 12, 13
**Decided in:** the whole effort; spec

## What to build

Prove the spec on a native install on the Incus test box (`nexul-box`,
from its `registered` snapshot, its own copy if anything else runs in
parallel), with a release built from master after 08 to 13 merged. Fix what
breaks in follow-up PRs.

Set up: a workspace with three projects (A, B, C), a private channel, a
public channel, an Owner, and a team member on From role. Invite a client
with Every project None and Project access on A only (tickets Write, docs
Read, memories Read). Redeem in a second browser profile.

Check, as the client, in the browser and over MCP with a token they mint:

- [ ] Only A in the project list, switcher, search, @ picker, board, and
      `project_list`; B and C by id or link read as not found, in the
      browser and through `project_get`, `ticket_get`, `doc_get`
- [ ] No public channel, not even `#general`; DMs both ways work; the
      private channel appears only once they are added; its lock shows
- [ ] Ticket and doc threads of A only; a pasted link to a B ticket shows
      the generic chip and opens to not found
- [ ] No instance area in the sidebar or over MCP (runners, machines,
      topology, DNS, automations), even with a role that grants them
- [ ] Memories of A only; no workspace memory anywhere; a plain chat turn
      carries none
- [ ] Notices from B (created while the client was From role) are gone,
      unread count included, and come back when B is granted
- [ ] A play runs on an A ticket; refused on an A ticket when the play
      excludes A
- [ ] Not offered as developer or tester on B's tickets; setting it over
      MCP is refused

Check, as the Owner and the team member:

- [ ] Granting B live: the client's open browser shows B without a refresh
- [ ] Taking A live: the client's open A page shows the revoked state; their
      socket stops A's frames; their MCP token loses A at once
- [ ] Switching the client to From role and back keeps their levels
- [ ] People with access lists the client on A; deleting C names nobody,
      deleting a project the client holds names them
- [ ] Making a channel private with the team member kept removes it from
      everyone else live; making it public restores it
- [ ] The Owner sees every private channel
- [ ] `account_list`, `account_update`, `invitation_create`, and
      `conversation_update` round-trip the new fields
- [ ] Upgrade path: an instance on the previous release with workspace
      memories, members, and invitations upgrades cleanly; members read as
      From role; workspace memories are gone

## Acceptance criteria

- [ ] Every box above checked, with a note of what failed and the PR that
      fixed it
- [ ] Screenshots at 768, 1024, and 1440px of the Team dialog, People with
      access, the revoked page, and the private channel settings, kept out
      of the repository

## Read first

`practices/testing.md`; this effort's `spec.md`; the test-box recipe the
team keeps for native-install end-to-end runs.

## Files likely touched

None by design; follow-up fixes land as their own PRs.
