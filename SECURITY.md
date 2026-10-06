# Security policy

## Supported versions

Only the latest stable release and the current `master` branch receive
security fixes.

## Reporting a vulnerability

Report it privately through
[GitHub's vulnerability reporting](https://github.com/otal-labs/nexul/security/advisories/new)
with a description of the issue, the steps to reproduce it, and the version or
commit you tested against. Only maintainers can see the report. Do not open a
public issue for a vulnerability.

You will get an acknowledgement within three working days, and the advisory
tracks the fix. Once it is released, we credit reporters in the release notes
unless they ask us not to.

## Scope

In scope: the Nexul server, runner, web app, desktop app, SDK, MCP server, and
the install scripts in this repository.

Out of scope: third-party services Nexul connects to (GitHub, Cloudflare, OAuth
providers) and the OpenObserve container shipped in the compose stack. Report
those upstream.
