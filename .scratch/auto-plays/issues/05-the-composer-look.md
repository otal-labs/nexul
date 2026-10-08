# 05: How the auto play composer looks on a play's settings page

Type: prototype
Status: open
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
