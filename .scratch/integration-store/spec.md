# Spec — The integration store

**Status:** needs-triage

> Carried over from the retired `docs/integrationsDomains.md` (the consent
> flow, the SDK's integration surface, the store and trust model, and the
> Discord reference integration). The platform underneath it shipped; the
> store around it never did. Nothing here has been re-confirmed as still
> wanted — the owner decides that before this is sliced into tickets.

## Problem Statement

Everything an integration needs at runtime exists: an install record with a
trust tier, a scoped revocable token, per-topic subscriptions, HMAC-signed
durable deliveries, a published event-schema catalog, and an audit log. None
of it is reachable by a person. Installing an integration means POSTing to the
API by hand, and there is no way to discover one in the first place — no
registry, no consent screen, no install helper, and nothing in the web UI at
all. The documented promise ("browse it, consent to its scopes, run the
container") has no surface behind it, and no integration has ever been written
against the platform, so nothing has proven the model end to end.

## Solution

A store: a place to find an integration, a consent step that shows exactly
what it is asking for before it gets a token, a generated snippet that gets it
running, and one reference integration that proves the whole path works.

## User Stories

1. As a workspace owner, I want to browse the available integrations, so that
   I can find one without reading a repository by hand.
2. As a workspace owner, I want each entry to show its author, version,
   requested scopes, and trust tier, so that I can judge it before installing.
3. As a workspace owner, I want a consent screen listing every requested scope
   in plain words, so that I know what the integration can reach before its
   token exists.
4. As a workspace owner, I want the consent screen to show which events will
   be delivered and to which URL, so that nothing leaves my instance without
   me seeing it.
5. As a workspace owner, I want to cancel at consent, so that it is a real
   decision and not a formality.
6. As a workspace owner, I want a ready-to-run compose snippet generated on
   install with the token and URLs already filled in, so that installing is
   paste-and-run rather than a manual wiring exercise.
7. As a workspace owner, I want installed integrations listed in the UI with
   their scopes, subscriptions, and recent deliveries, so that I can see what
   each one is doing.
8. As a workspace owner, I want to revoke an installed integration in one
   click and see its access stop immediately, so that a misbehaving
   integration is a ten-second problem.
9. As a workspace owner, I want failed deliveries visible with their errors,
   so that "the integration isn't reacting" is diagnosable without reading
   server logs.
10. As a workspace owner, I want integrations to run on their own network with
    only the API gateway reachable, so that a compromised integration cannot
    touch the database, the runner socket, or the MCP server.
11. As an integration author, I want a scaffolder that generates a working
    project, so that I start from something that runs rather than from a blank
    directory.
12. As an integration author, I want the SDK to verify delivery signatures for
    me, so that I do not hand-roll an HMAC comparison and get it subtly wrong.
13. As an integration author, I want typed event payloads generated from the
    published schemas, so that my handler breaks at compile time when I read a
    field that does not exist.
14. As an integration author, I want to list my integration in the registry
    with an honest trust tier, so that users know what review it did and did
    not receive.
15. As an integration author, I want to host my own artifact, so that I keep
    control of my release cadence.
16. As a workspace owner, I want a first-party Discord integration that posts
    deploy results, new tickets, and opened pull requests to a channel, so
    that the value of the store is visible without writing any code.
17. As a maintainer, I want the reference integration to exercise the whole
    path — subscribe, receive a signed event, call back into the API — so that
    a break in the platform shows up in something we run ourselves.

## Implementation Decisions

- The registry is a metadata index — name, author, version, requested scopes,
  trust tier, artifact location — held as static JSON in a git repository
  first; an API only if the static file stops being enough.
- Nexul never hosts integration code. Authors publish an image or a
  repository; the registry points at it.
- Trust stays two tiers, `verified` and `community`, shown prominently at
  consent. Numeric ratings and install counts wait for real volume (ADR 0043).
- Consent mints the scoped token; the raw token and the webhook secret are
  shown exactly once, as the existing install API already does.
- The install helper generates a compose snippet (image, token, instance URL)
  rather than printing raw values for the owner to wire up.
- Integrations sit on their own Docker network. From it only the HTTP gateway
  is reachable — never the database, the runner WebSocket, or the MCP server.
  The generated snippet sets this up by default.
- The SDK gains an integrations entry point beside its automations one: a
  delivery-signature verifier, typed event payloads generated from the
  published catalog, and a scaffolder command in the package's own CLI. A
  personal access token stays the credential for a private, one-off
  integration; store integrations always go through consent.
- An integration may expose itself as an MCP server instead of an HTTP
  service. Both ride the same use-case layer, so that is a second shape of the
  same thing, not a second platform.
- API rate limits land before the store accepts third parties.

## Testing Decisions

- Test at the API seam that already exists for installs, subscriptions, and
  deliveries. The store adds a registry reader and a consent step in front of
  it, not new domain behaviour, so no new seam should be needed.
- The reference integration is the end-to-end test: subscribe, receive a
  signed delivery, verify it with the SDK, call back into the API with the
  minted token, and assert the effect. If that passes, the model holds.
- Cover the negative paths first, as the existing integration tests already
  do: revoked token, wrong signature, replayed delivery, a scope the token was
  never granted, and a delivery to a URL that refuses connections.
- The generated compose snippet is worth one test that it is valid and carries
  the right values; the network isolation claim is worth one test that the
  integration network genuinely cannot reach the database.

## Out of Scope

- The in-code security review of the integration model. It already exists as a
  standing pre-release item and is the gate before the store accepts
  third-party publishers.
- Redacting secret values from event payloads — a separate standing
  pre-release item, due before any third party receives deliveries.
- Field-level subscription filters. Subscriptions stay per-topic.
- Paid integrations, licensing, or anything resembling a marketplace with
  money in it.
- Automations. First-party event-driven code is a different thing on a
  different delivery path (ADR 0046) and is not distributed through this
  store.

## Further Notes

- The platform half is live: scoped tokens, signed and retried deliveries, the
  published schema catalog, and the audit log all ship today. This spec is the
  surface around them.
- The event catalog is a frozen contract (ADR 0044), so an integration written
  against today's payloads keeps working.
- Trust tiers already exist on the install record, so a registry entry has
  somewhere to land.
