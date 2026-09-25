# Betas are cut once a day, when master has moved

Supersedes the beta-per-push rule in ADR 0070 and reverses ADR 0053's rejection of a scheduled build; tags stay the
version.

A scheduled run at 03:17 UTC cuts the next `v<next>-beta.<n>` from `master`, and does nothing when `master`'s head
already carries a release tag. A manual run with the beta channel cuts one immediately, for a fix that should not
wait for the next day. A beta's release notes list every pull request merged since the previous beta. Stable
promotion is unchanged: a manual run promotes the newest beta's commit.

Why: Nexul stays in beta until the owner is satisfied, which can be a long time, and a release per merge filled the
Releases page with entries nobody installs individually while rebuilding every binary and image on a docs change.
ADR 0053 rejected a scheduled build as a worse "build when something moves"; the head check answers that objection,
since a quiet day builds nothing.

The trade-off: a merged fix reaches beta installs up to a day later unless someone runs the workflow by hand.

Decided: 2026-09-25
