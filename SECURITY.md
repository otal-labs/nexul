# Security policy

## Supported versions

Only the latest stable release and the current `master` branch receive
security fixes.

## Reporting a vulnerability

Open a [security report](https://github.com/otal-labs/nexul/issues/new?template=security_report.yml)
on the issue tracker with a description of the issue, the steps to reproduce
it, and the version or commit you tested against. Issues are public, so keep
working exploits and live instance details out of the report and say you have
them; a maintainer will follow up with a private channel. The project is
young enough that this is the fastest path; a dedicated reporting address
comes later.

You will get an acknowledgement within three working days, and the issue
tracks the fix. Once it is released, we credit reporters in the release notes
unless they ask us not to.

## Scope

In scope: the Nexul server, runner, web app, desktop app, SDK, MCP server, and
the install scripts in this repository.

Out of scope: third-party services Nexul connects to (GitHub, Cloudflare, OAuth
providers) and the OpenObserve container shipped in the compose stack. Report
those upstream.
