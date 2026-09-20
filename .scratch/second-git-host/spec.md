# A second git host

**Status:** needs-triage

> Carried over from the connectors and git provider domain specifications when
> they were dissolved. An old wish, not a queued build — re-confirm it is
> still wanted before slicing it.

## Problem Statement

Nexul is open source and self-hosted, but its only git host is
github.com. Someone running their code on a self-hosted GitLab, on Gitea, or
on GitHub Enterprise Server cannot use the product's core loop at all: no
repository scan, no pull request mirror, no branch links, no repo-driven
build. Which hosts are usable is currently gated on the maintainers shipping
code, which is the opposite of how a self-hosted product should work.

## Solution

An owner picks a git host type from the list Nexul ships adapters for,
registers their own app on that host, pastes its credentials and — for a
self-hosted instance — its base URL, and connects. From then on repositories
on that host behave exactly like repositories on github.com: scanning,
linking, pull requests, webhooks, and builds. Having github.com and a
self-hosted host connected at the same time is normal, not an edge case.

## User Stories

1. As an owner, I want to connect a git host other than github.com, so that I
   can deploy code that does not live there.
2. As an owner, I want to connect a self-hosted instance of a host by giving
   its base URL, so that GitHub Enterprise Server and self-hosted GitLab work
   the same way as the public services.
3. As an owner, I want to register my own app on that host rather than trust a
   Nexul-operated one, so that no credential of mine is held by anyone
   else.
4. As an owner, I want to have several git hosts connected at once, so that
   one project's repository can live on one host and another's somewhere else.
5. As an owner, I want to choose which connected host a repository lives on
   when I link it to a project, so that the right credentials are used for it.
6. As an owner, I want the picker to preselect the only connected host when
   there is only one, so that the common case stays one click.
7. As a developer, I want a repository whose host is disconnected to fail
   loudly for that repository only, so that work on other repositories is
   unaffected.
8. As a developer, I want pull requests, reviews, branch links, and pushes
   from the new host to drive tickets, deploys, and previews exactly as
   GitHub's do, so that nothing about the product changes with the host.
9. As an owner, I want repository scanning to work on the new host, so that
   the project wizard can propose a compose file or Dockerfile from it.
10. As an owner, I want a runner to be able to clone a private repository on
    the new host, so that repo-driven builds work there.
11. As an owner, I want the host's webhook deliveries verified and normalized
    into the same internal events, so that automations written against those
    events keep working unchanged.
12. As an owner, I want everything already linked to keep working when this
    ships, so that adding host support is not a re-linking exercise.

## Implementation Decisions

- The plumbing is already in place and is not revisited: the provider
  interface is the seam, a repository record already names the connector
  hosting it, a connector's app config already carries an optional base URL,
  and the composition root already resolves a provider per repository and
  builds the client fresh per call. What is missing is a second adapter and
  the owner-facing picker.
- The host type is chosen from a fixed set Nexul ships adapters for.
  This deliberately does not become "paste any OAuth2 configuration":
  different git hosts need genuinely different auth and API handling, not just
  different credentials.
- A new host is one package satisfying the provider interface plus one
  registry entry flagged as a git provider. Provider-specific quirks — auth
  shape, pagination, vocabulary, what an "installation" means where the
  concept does not exist — stay inside that package; nothing upstream imports
  provider internals.
- Webhook deliveries from the new host normalize into the same
  provider-agnostic envelope, so every existing consumer works untouched.
  Signature verification is the provider package's job.
- The credential the runner gets for a private clone comes from the same
  connector the repository is linked to, the same way it does for GitHub.
- Which host to add first is undecided: GitLab (self-hosted is the common
  case) and Gitea (smallest API surface) are both candidates, as is GitHub
  Enterprise Server, which needs only the base URL and no new adapter at all.
  GitHub Enterprise Server is probably the cheapest first proof that the
  resolution path really is host-agnostic.

## Testing Decisions

Test the new adapter against a stub HTTP server the way the GitHub client is
already tested — a real request/response round trip against recorded shapes,
not a mock of the interface. Above that, test the routing behaviour through
the per-repository resolution: two repositories on two connectors reach two
different clients, and disconnecting one leaves the other working. Webhook
normalization is tested at the receiver by asserting the envelope, never the
provider's raw payload; the existing GitHub normalization tests are the prior
art. A good test here never asserts a provider's URL layout beyond what the
stub already pins.

## Out of Scope

- Arbitrary user-supplied provider configuration. The set of host types
  stays a fixed list.
- Migrating an existing repository link from one host to another.
- Any change to the product's behaviour once events are normalized — tickets,
  reviews, deploys, and previews are host-agnostic already.

## Further Notes

One question stays open and should be answered before the first adapter is
written: whether per-repository webhook registration is still needed at all,
given a GitHub App receives events centrally for everything it is installed
on. If it is dropped for GitHub, a new host must either offer an equivalent or
keep the per-repository path alive, and the interface should settle that
before it grows a second implementation.
