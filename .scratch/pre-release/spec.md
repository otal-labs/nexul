# Pre-release standing items

> Carried over from the retired `reminders.md` (2026-08-15). These are the standing
> items that were still active when the repo moved to the `.scratch/` tracker.
>
> They are **not** a feature — they're standing concerns that were deliberately
> deferred during the build phase and become due before any production or public
> use. They live together here so a cold session can find them in one place.

Each issue keeps its original **Surface when** list — the conditions that mean
"raise this with the owner now, before doing the work". That was the whole point
of the old reminders file: sessions reset, so the trigger list is the memory.

**Never act on one of these silently.** Surface it, let the owner decide.

## Timing

Owner decision (2026-08-11): these are post-main-phase / pre-release concerns.
During ordinary feature work, don't surface them unless a specific trigger on the
issue itself fires. They become due once the main domain phase is complete.

## Issues

| # | Item | Status |
|---|---|---|
| 02 | Update GitHub App homepage + callback URLs | `ready-for-human` |
| 03 | Security-review the integration model before the store launches | `ready-for-human` |
| 05 | Revisit canvas node kinds once topology ships | `needs-triage` |
| 06 | Register the Cloudflare OAuth app | `ready-for-human` |

Closed: 01 (OAuth secret rotated), 04 (deploy env secrets redacted, shipped with
the automations rework), 07 (dev-login verified off: only the debug compose
file sets `NEXUL_DEV_LOGIN`, and the route is never registered without it).
Resolved in the old file and not carried over: *R3 — wipe/rewrite the outdated
README* (done 2026-08-02, PR #4).
