# A DM shows in every workspace its people share

A DM lived in the workspace it was started in (ADR 0087): two people who share several workspaces saw their DM in one
and not the others, and it vanished from view whenever they switched.

Decision: a DM still starts in a workspace, between members of it, and that stays its home row. It is listed, with
its unread count, in its home workspace and in every other workspace all its participants belong to. Reading and
posting in it take being a participant alone, not membership of its home workspace, so a participant who leaves
that workspace keeps it wherever the people still meet.

- **Where all its people belong**, not every workspace the viewer is in. People and names resolve per workspace
  (ADR 0086), so a DM shown where the other person is not a member would name nobody.
- **The home workspace keeps it** even after someone leaves, so an old DM never drops out of every list.
- **No new storage.** The `workspace_id` column keeps the home, events still name it, and lists read the person's
  participant rows through a new index. Clients refresh every workspace's list for a new DM, and every cached list
  that holds a DM for its messages.

The trade-off: a DM between people who no longer share any workspace is reachable only by its link or id, still read
by its participants.

Rejected: every DM in every workspace of the viewer's, which shows DMs whose people cannot be resolved there; and
moving DMs out of workspaces entirely, a table rebuild of `conversations` and a change to every event's
`workspace_id` for the same visible result.

Amends ADR 0087, whose "a DM's participants must all be members of its workspace" now holds when the DM is started,
and whose membership check no longer applies to reading one. Decided 2026-10-10.
