# 24 — Fix with AI on the ticket page

**What to build:** The ticket page shows a Plays section in the right rail under Properties, one button per play whose show-when stage matches, and a sticky bottom bar with the same buttons when the rail is collapsed. A button is hidden without `plays:run`, disabled with the readiness reason when the harness is not ready, disabled with "a run is in progress" while a trail is active, and enabled otherwise. Pressing it opens the run dialog: memories (always-included ticked and locked), instructions, move-to from the project's columns with the never-backwards note, and a confirm naming the destination; memories and move-to are pre-selected from the caller's latest trail for that play in that project; the move-to field is disabled with a reason for a caller without `tickets:write`. While running, the button shows a spinner with a Stop square, the board card shows a spinner beside the id, and the thread shows the running line; all update live. A Trail section under the ticket body lists the ticket's trails and opens each one with its activity lines. Layout as on the `proto/plays-ui` branch.

**Blocked by:** 14, 23

**Status:** done

- [ ] Pressing Fix with AI on an In progress ticket runs it end to end against the fake harness in a browser test, the thread shows Started and the reply, the ticket lands in In review
- [ ] All four button states render from real readiness and trail data; no state is hardcoded
- [ ] The dialog's pre-selection changes after a run with different choices
- [ ] Board card and ticket page update without a refresh when a trail starts and ends
- [ ] Stop works from the page; the Trail section shows the interrupted row and opens its record
- [ ] Verified at desktop width; the bottom bar appears when the rail collapses
