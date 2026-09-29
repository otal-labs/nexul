# Betas count up the patch number

Supersedes the tag format in ADR 0071; the daily cadence stays.

Betas were tagged `v<next>-beta.<n>`, with the count in the pre-release part. GitHub lists releases by day and then
by tag text, so on a day with several betas `beta.9` sat above `beta.11` on the Releases and tags pages and in the
API's list order, and anything reading that list first picked the wrong one.

Decision: every release on a version line takes the next patch number. Betas are `v<line>.<n>-beta` (`v0.3.0-beta`,
`v0.3.1-beta`, ...), `<n>` one past the highest patch on the line, beta or stable. A stable `patch` promotion
releases the newest beta's commit under the same number (`v0.3.12-beta` becomes `v0.3.12`), so the next beta is
`v0.3.13-beta`; a `minor` or `major` promotion starts a new line (`v0.4.0`) and betas continue at `v0.4.1-beta`. The
line starts at 0.3, after the `v0.2.0-beta.<n>` series.

Why: the number that orders releases now sits in the version core, which every tool compares as a number: semver,
`sort -V`, and GitHub's version handling. A stable release still outranks its own beta, and one counter replaces two.

Consequences: a day that crosses from 9 to 10 can still show GitHub's text order once, since `v0.3.10-beta` sorts
below `v0.3.9-beta` as text; Nexul itself never reads that order (the release client, installers and changelog sort
by publish time). The last `v0.2.0-beta.<n>` cannot be promoted; the first stable release promotes a `v0.3.<n>-beta`.

Decided: 2026-09-29
