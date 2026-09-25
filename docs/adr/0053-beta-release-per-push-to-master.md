# A beta release on every push to master, numbered from the releases list

Superseded by ADR 0070 (git tags are the version, betas are `v<next>-beta.<n>`, no VERSION file) and ADR 0071 (one beta a day, when master has moved).

Every push to `master` (which, with squash-only merges, means every merged pull request) cuts a prerelease
tagged `v<VERSION>-<NNN>`, where `VERSION` holds the line currently in beta with its suffix (`0.2.0-beta`) and
`NNN` is a zero-padded counter: `v0.2.0-beta-001`, `v0.2.0-beta-002`, and so on. The counter is derived at
build time as the newest published beta on that `VERSION` line plus one; nothing writes it back to the repo.
Stable keeps the promote-only rule from 0051: a manual dispatch builds the commit of the newest beta as
`v<VERSION>` with the suffix dropped, then bumps the patch and keeps the suffix (`0.2.1-beta`). The channel
rule is the same everywhere — a stamped version with a dash after the semver is beta, a bare one is stable — so
`version.Channel()`, the release client's "newest beta" lookup, the workflow's tag filters, and the image prune
all agree without a shared list of suffixes.

Rejected: keeping the hourly nightly. A cron that skips itself when nothing moved is a worse version of "build
when something moves", and its date-and-run-number tags said nothing about how many builds a line had seen.
Also rejected: committing the bumped counter into `VERSION` on every merge. That doubles the commit count on
master with bot commits and, worse, races: two merges queued back to back both check out a `VERSION` that
predates the first run's bump and would claim the same number. Reading the newest beta off the releases list
has neither problem, at the cost of one extra API call per release.

The trade-off: `VERSION` no longer shows the exact current build, only the line (`0.2.0-beta`); the number lives
on the releases page and in the stamped binaries. A docs-only merge also cuts a beta, since the workflow does
not filter paths — cheap, and simpler than deciding which paths "count".
