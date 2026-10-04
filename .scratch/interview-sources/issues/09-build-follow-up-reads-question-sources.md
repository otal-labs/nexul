# 09 — The follow-up run reads question sources

**What to build:** The Interview play's instructions (code default and
instance template, plus a forward migration updating the built-in
interview play's instructions where the workspace never edited them, as
in the earlier interview build) learn about sources: read every question
source as well as the checkout, and ask about what they did that the
answers do not settle, each with a recommended answer and a why-line
naming what was found and where, such as "Phase 1 keeps booking state in
the mobile app (`src/state/booking.ts`). Keep it on the server for phase
2?". Never record from a question source what the person did not confirm.
The runner names `project` sources with their checkout path the same way
as 08. Follow sources are read as context for the gaps but drafts are not
written here.

**Blocked by:** 07

**Status:** resolved

- [x] A run on a fake harness asks a round naming a question source's file
- [x] Instructions migration leaves an edited workspace play untouched

## Answer

Built in #439.
