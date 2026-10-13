# 06: How an admin sees each person's progress on Team

Type: prototype
Status: resolved
Blocked by: 02

## Question

Team shows each person's Welcome wizard progress, so an admin can see why
something isn't working for them. Where does it sit (a column, a chip, the
person's detail), how does it read ("2 of 3, computer not added"), and who may
see it? Build it in auto design mode behind a picker. Decide too whether
accepted invitations ("accepted by Bob, 2 days ago") belong in the same view.

## Answer

Decided in auto design mode, 2026-10-13. The prototype is on the pushed branch
`proto/team-progress`, at `/proto/team-progress?v=1|2|3`, and is never merged. The locked
reference is in `~/Code/Images/`, saved as
`2026-10-13-nexul-team-progress-grouped-task-counts.jpg`.

**Gap line (picked).** It scored 5. Progress column and Step strip each scored 4.

- **The row**: a third line under the person's role, shown only while a step is left or skipped.
  - It reads: a dot, then a mono "2 of 3 set up", then "· computer not added", then
    "· skipped phone app" in muted text.
  - The dot is `warning` while a step is left, and muted when only skips remain.
  - With several steps left, the first is named and the rest counted ("· GitHub not connected
    +2"). When the wizard was never opened, it says "hasn't opened the wizard yet".
  - Nothing shows when every step is done, or for disabled and removed accounts.
  - The wording for each step left:
    - GitHub not connected
    - computer not added
    - not in a project yet
    - deploy server not added
    - phone app not set up
- **The person dialog**: a "Welcome wizard" microheader with "N of M set up".
  - One bordered row per step that is left or skipped: the name, a muted line saying why from
    the data, and either "Not done" with a `warning` dot or "Skipped · 5 days ago" muted.
  - Done steps fold into one muted line with a check.
  - The header gains "Joined 2 days ago, invited by alice".
- **"Not in a project yet"** shows on Team whenever the person is in no project, even though it
  doesn't light their own dot. The admin is the one who fixes it.
- **Who sees it**: whoever can see Team's member list today. They see the whole wizard, since
  there is one per person.
- **Accepted invitations are included.** The Invitations tab gains "Accepted in the last 30
  days", newest first. Each row shows the name, the role it granted, who invited them, a mono
  "accepted 2 days ago", the same gap line, and opens the person on click.
- **Server data**:
  - `TeamPerson.welcome.steps: [{key, state: done|skipped|left, at?}]`, derived under ticket
    02's rules.
  - `TeamPerson.joined: {invited_by, accepted_at}`, read from the invitation's existing
    `redeemed_by` and `redeemed_at`.
  - The invitations list returns the links accepted in the last 30 days.
- **Runner-up, Progress column**: a 96px column of step segments over "2/3 set up". It lost
  because what's missing only shows on hover, and it puts the accent on most rows. To swap,
  render `ProgressColumn` in place of `GapLine`.
- **Not verified**: real data (it uses a mock route), motion, light mode beyond 1024, and a
  screen reader.
