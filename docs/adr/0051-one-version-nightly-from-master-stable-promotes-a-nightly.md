# One product version; nightly cuts from master, stable only promotes a nightly

Superseded in part by 0053: the hourly nightly became a beta cut on every push to master. The one-version and promote-only-what-shipped rules stand.

Nexul ships one version for the whole product — server, web,
automations and runner images, plus the runner binaries and server
tarballs released alongside them — cut from a single commit and stamped with
a single tag. The `VERSION` file at the repo root holds the version the
*next* stable release will carry. Nightly is a scheduled build: whenever
master has moved since the last nightly, it publishes a prerelease tagged
`v<VERSION>-nightly.<date>.<run>`. Stable is never built from master
directly — a manual dispatch resolves the newest published nightly and
builds that exact commit as `v<VERSION>`, then bumps `VERSION`'s patch
component for the next cycle. A stable release therefore always ships a
commit a nightly has already carried and exercised.

Rejected: per-component versions (server, web, runner, automations each
tagged independently) — the product is deployed as a unit, and a per-service
version matrix multiplies the compatibility questions ("does web v3 work
with server v7?") without buying anything, since nothing here is consumed as
a library. Also rejected: releasing stable straight from master on every
dispatch — that would mean the first time a commit is built for real
platforms and pushed as multi-arch images is also the moment it ships as
stable, with no soak time. Also rejected: keeping the old run-number scheme
(`v0.1.<run>`) for the whole product — it carries no signal about what
changed and doesn't distinguish a deliberate release from an automatic one.

The trade-off: a stable release always lags master by at least one nightly
cycle, so there is no path to ship a commit as stable within the hour it
lands. A hotfix means cutting a nightly first (or dispatching nightly by
hand) and only then promoting it — slower than a direct release, but the
same soak-before-ship discipline applies to every stable release without a
"this one's an emergency, skip the check" escape hatch to maintain.
