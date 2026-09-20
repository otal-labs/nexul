# Board configuration moves from workspace to project scope; the combined all-projects board is retired for a configurable mention-chip template

Status columns (`statuses`), ticket types (`ticket_types`), and label
colors (`label_colors`) were workspace-level (label colors) or entirely
unscoped (statuses, ticket types) — every project in a workspace saw the
same column set. This stopped matching how workspaces were actually being
used: different projects want different workflows (a design project's
statuses aren't a backend project's statuses), and `Project` was already
the real unit tickets, categories, and repos hang off of (`categories`
already made this exact move — `Category.ProjectID`). All three gain a
`project_id` (statuses/ticket types) or move their compound key from
`(workspace_id, label)` to `(project_id, label)` (label colors), following
the `categories` precedent exactly. No migration/backfill logic — no
production data exists yet, so this is a clean schema change. New
projects seed their statuses/types from the previous global defaults
(`Open`/`In progress`/`Done`/`Closed`, `task`/`bug`/`feature`) on
creation, so nothing starts empty.

This removes the only view that showed tickets across every project at
once: the unfiltered `/board` route. Keeping it would have meant either
unioning mismatched column sets from different projects into one
confusing board, or picking one project's columns arbitrarily to render
others' tickets under — both worse than just requiring a project. `/board`
with no project now redirects to the last-viewed project (persisted the
same way `selectedWorkspaceId` already is), falling back to the first
project by position.

To offset the lost cross-project visibility, the `@`-mention chip
(`MentionChip.tsx`) — previously a hardcoded icon+title+status — becomes a
workspace-configurable free-text template (`{ticket.Project}
{ticket.Ticket}` → `ERF-1 Fix login redirect loop`), so a mention dropped
into a doc from any project can now surface which project it came from
without needing a shared board to browse. Default template renders
identically to the old fixed shape, so existing workspaces see zero visual
change until an owner opts in. Editing the template is gated by a new
`manage_mention_layout` workspace permission, following the
`manage_roles`/`manage_workspace_members` pattern exactly. `@`-mention
search also gained direct `PREFIX-NUMBER` key lookup (`@ERF-1`) alongside
the existing title search, using the display-only prefix/number pair from
ADR 0004.

**Known gap, not fixed by this change:** `BoardSettingsSection` (the UI
for editing a project's statuses, ticket types, and label colors) still
has no permission gate of its own — anyone who can reach project settings
can edit board configuration, same as before this migration. This was
flagged during design as a real gap but ruled explicitly out of scope;
it's tracked as a future effort, not silently fixed here.
