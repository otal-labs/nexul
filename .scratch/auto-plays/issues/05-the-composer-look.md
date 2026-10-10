# 05: How the auto play composer looks on a play's settings page

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

A play's settings page gets an Auto plays section: a list of the play's
auto plays (each with an on/off switch and a one-line summary like "When a
ticket becomes unblocked, if type is Bug: High") and the composer for one:

```
When      [Ticket becomes unblocked ▾]
If        [all ▾] of
            Type      is      [Bug ▾]
            Project   is not  [Website ▾]
            [any ▾] of
              Label   has     [urgent ▾]
              Stage   is      [backlog ▾]
Priority  High if Type is Bug, otherwise Normal
Limits    once per ticket per unblock
Run on    [Developer ▾]
```

Also on that page: what is queued for this play right now, with who it is
waiting on and why (computer offline, no free slot), and the workspace's
daily per-ticket cap if it is shown there.

Prototype with `design-mode` against the existing settings kit
(`SettingsCard`, `EmptyRow`) and the Mono Console; the owner picks by
looking.

## Answer

Picked in auto design mode (2026-10-10), runner-up reported to the owner
for a swap. Prototype: branch `proto/auto-plays`, variant B; screenshots
and the reference notes are kept outside the repo.

- The play's edit dialog (36rem, the width for a form wider than its
  fields) gets line tabs, **Play** and **Auto plays**, kept in the dialog
  (`Tabs` with the underline look, not `PageTabs`, since a dialog has no
  path). Interview plays have no Auto plays tab.
- **The list**: one hairline row per auto play reading as a sentence, the
  variable parts in medium foreground and the glue words muted ("When a
  ticket **becomes unblocked**, if **Type is Bug** and **2 more** →
  **High**, else **Normal**, runs on **Developer**"), a Switch and the row
  actions menu (Edit, Duplicate, Delete) on the right; an outline "+ Add
  auto play"; under it two muted lines: "Right now: 2 queued · 1 waiting on
  <person> (computer offline)" and "Each ticket runs at most 5 auto plays a
  day. Set in Configuration" (a prose link). Empty: an `EmptyRow`.
- **Drill in**: a row opens its composer in place of the list, with
  "← Auto plays" at the top of the body; the dialog's footer Save and
  Cancel act on that auto play. A new auto play opens the composer the
  same way.
- **The composer**: section labels small and above, no left label column.
  When: one select (ticket plays: becomes unblocked, enters a stage + a
  stage select, is created, gets a developer, gets a tester, fails a test;
  doc plays: is created, changes, with the muted hint "fires once edits
  stop for 10 minutes, never for an agent's edits"). If: "[All ▾] of these
  3 conditions must match:" with the count in medium weight, one row per
  condition (field, operator, value; yes/no fields put yes and no in the
  operator and have no value box), quiet `+` and `×` icon buttons at the
  right, a nested group one indent in with its own "[Any ▾] of these
  conditions must match:" and no box. Priority: "[High ▾] if [field]
  [operator] [value]" rows and "Otherwise [Normal ▾]". Limits: "At most once
  per ticket every [No limit / 1 hour / 24 hours / 7 days ▾]". Run on:
  Developer, Tester, Whoever caused it.
- Long value lists truncate with an ellipsis and show in full on hover; a
  tall composer scrolls inside `DialogBody` with the footer pinned.
- Rejected: expanding the composer inside the list row (a box inside a
  box), and a 56rem two-pane dialog (the list squeezed to six lines and a
  value picker collapsed at 768px).
