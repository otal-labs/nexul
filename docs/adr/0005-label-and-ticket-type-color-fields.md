# Label and ticket-type color: additive fields, not a Label entity

The Board design pass (`.scratch/board-design-pass/`) needs both ticket
labels and ticket types to carry an owner-configurable color, replacing
`ticketTypeColor.tsx`'s frontend-only hash for type and giving labels a
color for the first time. `Status` and `TicketType` are already real
workspace-owned tables (`internal/workspace`); `ticket_labels` is not — it's
just `(ticket_id, label)` text pairs, with no `labels` table at all.

Two shapes were considered for labels: promote `Label` to a full entity
(`labels(id, workspace_id, name, color)`, `ticket_labels` migrated to
reference `label_id`) or add a small additive side table keyed by the
label's text. Full promotion would introduce rename-everywhere/
delete-everywhere semantics that nothing has asked for and that don't
exist today — labels are still just text for every purpose except color.
We chose the side table: `label_colors(workspace_id, label, color)`,
purely additive, zero changes to existing label CRUD.

`TicketType`, unlike `Label`, already has a real table — its color is a
plain additive `color` column on `ticket_types`, the same shape as
`Status.icon`.

Both use a fixed suggested palette (not freeform hex), validated the same
way `StatusIcon` validates against `validStatusIcons`. An unset color on
either falls back to the existing hash-based color
(`ticketTypeColor.tsx`'s pattern) so nothing looks broken before an owner
configures anything.
