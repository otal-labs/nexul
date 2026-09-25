# Git tags are the version; there is no VERSION file

Supersedes ADR 0053. ADR 0051's one-version and promote-only-what-shipped rules stand.

The repository holds no version file. Every push to `master` is tagged `v<next>-beta.<n>`, where `<next>` is the
newest stable tag with its patch bumped (0.2.0 before any stable release exists) and `<n>` counts that line's
betas from the tags already in the repository. A manual run of the Release workflow promotes the newest beta's
commit to stable with a patch, minor or major bump chosen at that moment, and refuses when that commit is already
released. GoReleaser builds whatever tag the workflow picks, stamps it into both binaries, and publishes the
binaries, `checksums.txt`, the `nexul` image and the GitHub release. Beta tags use a dot before the counter
(`beta.3`), so semver orders them numerically without zero padding.

Why: Go modules and GoReleaser both treat the tag as the version, so a file is a second source of truth to keep
in step with it. Writing the file back on each merge also cannot work here: `master` only accepts pull requests,
and two merges in quick succession would read the same file and claim the same number.

Rejected: release-please's release pull request. It decides the bump from conventional-commit prefixes, which
this repository's plain-sentence pull request titles do not carry, and it releases when its own pull request is
merged rather than on every merge.

The trade-off: nothing in the tree says what version is next; the Releases page and `nexul version` do. Bigger
bumps are a choice made when promoting, not a file edited ahead of time.

Decided: 2026-09-25
