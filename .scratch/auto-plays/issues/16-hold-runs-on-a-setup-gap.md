# 16: Hold a run on a setup gap, and let its person pick a computer

Type: task
Status: open
Blocked by: None — can start immediately

## Question

The walkthrough found that a run whose person's computer setup isn't done
(nothing paired, the session expired, no T3 project picked, or several
computers and none the default) fails at once as "didn't run". The owner
decided (2026-10-11) that such a run waits instead, like an offline
computer: it stays queued with the plain reason, the ticket's rail says what
to fix, and it starts once the gap is fixed.

When the gap is "several computers are paired and none is the default",
the waiting run asks its person which computer to run on: a "Pick a
computer" action on the queued row in the rail (and on the person's
notification if one exists), listing their paired computers; picking one
starts the run there. It doesn't change their default.

"Nobody to run on" and "may not run the play" still don't run (ADR 0132).
Update ADR 0132 or add an amendment line if the reasons a run waits change,
and the guide's Auto plays section. Go and web tests; verified at 768, 1024,
and 1440px.
