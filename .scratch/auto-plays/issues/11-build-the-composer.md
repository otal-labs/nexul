# 11: Build the Auto plays section on a play's settings page

Type: task
Status: open
Blocked by: 05, 09

## Question

Build what 05 locked: the list of a play's auto plays, the composer, and
the play's queued runs. Web tests; verified at 768, 1024, and 1440px.
Update the plays page of the user guide
(`website/src/content/docs/docs/guide/`).

## Notes
- Changing the daily cap publishes no event yet, so another tab's settings
  page won't follow it live; add `workspace.auto_play_limits_updated` (or
  ride an existing workspace topic) with this ticket.
- `PATCH` replaces conditions whole: the composer must send back values it
  can't label (projects the editor can't see), or a save drops them.
