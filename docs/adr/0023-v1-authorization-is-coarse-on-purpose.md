# Authorization stays coarse inside tickets and chat, but the seams are already there

Ticket permissions are **workspace-level**: there is no per-ticket grant and
no per-ticket permission UI. Crucially, permission is never inferred through
a ticket relationship — a link between two tickets, or between a ticket and
a doc or PR, is a data relationship and never an implicit access grant.
Otherwise anyone could widen their own access by linking, and the effective
permission set would be a graph traversal nobody can reason about.

Chat goes further and **enforces nothing in v1**: any workspace member can
create a conversation and message anyone, and a voice channel is public to
the whole workspace. Conversation participants are still persisted as
explicit rows even though nothing reads them for authorization yet, so the
eventual who-can-talk-to-whom check has exactly one place to land instead of
needing a schema change first.

Projects sit at the same coarseness: every member of a workspace sees every
project in it, and board configuration writes are gated only by the
instance-admin bit. Per-project visibility was left as a later access-domain
option rather than built in alongside the workspace split, because a project
filter that the access layer will eventually own would have had to be written
twice — once as an ad-hoc check in the workspace use-cases and again inside the
grid — and the second one would have to find and delete the first.

This is a deliberate deferral, not an oversight: features are never gated or
shrunk on a missing permission bit, and one sweep applies the shared
`<domain>:<action>` grid (ADR 0010) across every surface once the product
surface stops moving.
