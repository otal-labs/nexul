# Security policy

## Supported versions

Security fixes land on `master` and in the latest stable release only.

## Reporting a vulnerability

Report it privately through
[GitHub's vulnerability reporting](https://github.com/otal-labs/nexul/security/advisories/new).
Include what's wrong, how to reproduce it, and the version or commit you
tested. Only maintainers see the report. Please don't open a public issue.

We acknowledge reports within three working days and track the fix in the
advisory. Once the fix ships, we credit you in the release notes unless you'd
rather we didn't.

## Scope

In scope: the Nexul server, runner, web app, desktop app, SDK, MCP server, and
the install scripts in this repository.

Out of scope: third-party services Nexul connects to (GitHub, Cloudflare, OAuth
providers) and the OpenObserve container shipped in the compose stack. Report
those upstream.
