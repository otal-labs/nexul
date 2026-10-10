# 11: Build the Auto plays section on a play's settings page

Type: task
Status: resolved
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

## Answer

Built in PR #529: line tabs Play | Auto plays in the play dialog
(ticket and doc plays, `autoplays:read`), sentence rows with a switch that
saves at once, Duplicate and Delete, drill-in composer with a
"Discard changes?" check on back, Cancel, Esc, close and the Play tab,
values the editor can't name kept as "Unknown …", read-only without
`autoplays:write`. The daily cap is an inline 1–50 select under the list
("…auto plays a day, across every play"), not a Configuration link, since
Configuration has no cap setting; it publishes `auto_play.limits_updated`.
The guide's plays page has an Auto plays section. Left out: the "Right
now: N queued" line (needs 10's queue, now in 12).
