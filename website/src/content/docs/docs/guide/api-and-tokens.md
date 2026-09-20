---
title: API and Tokens
description: The HTTP/JSON API, its permission model, and the tokens that authenticate against it.
sidebar:
  order: 13
---

Every capability the web UI uses is also available as a plain HTTP/JSON API under `/api` — the same use-case layer the MCP server calls, just with a REST-shaped adapter instead. Nothing is MCP-exclusive and nothing is UI-exclusive.

## OpenAPI

The API is documented as an OpenAPI 3.x spec, generated directly from the routes the gateway actually mounts — never hand-written, so it can't drift out of sync.

- `/openapi.json` — the raw spec.
- `/swagger` — a Swagger UI browsing it.

## Permissions

One vocabulary covers every actor: a permission is `<domain>:<action>`, where the action is `read`, `write`, or `delete`, and the domain is one of the API's own domains — `docs`, `tickets`, `stacks`, `topology`, `projects`, `workspaces`, `members`, `roles`, `dns`, `automations`, `integrations`, and more. The full catalog (26 domains) is served at:

```
GET /api/permissions/catalog
```

A role, a personal access token, and an automation's scoped token are all checked against these same values — "what can this actor do" has one answer regardless of who's asking.

**Roles** live per workspace membership. Every workspace has a singleton, unremovable **Owner** role that implicitly holds every permission. Beyond that, roles are fully custom: anyone holding `roles:write` can create a role with whichever permissions it needs.

## Personal access tokens

Mint one from **Settings → Personal access tokens**. A PAT is long-lived, revocable, and carries exactly your own permissions — it's the credential an agent or a personal integration uses to act as you, including for the [MCP server](/docs/guide/mcp-server/). The raw token (prefixed `dep_`) is shown once at creation and can never be retrieved again; revoking it cuts access immediately.

## Connection tokens

A **connection token** is different: it carries no identity or credentials at all, just server information (the instance URL, derived MCP endpoint, and basic settings) as a signed JWT. It exists so a standalone client — the [desktop app](/docs/guide/desktop-app/) today — can be pointed at your instance without you typing a URL by hand. Generate one from **Settings → Connection token**; after importing it, the client still signs you in through the normal GitHub OAuth flow.

## Outgoing webhooks

Third-party integrations receive events as signed, durable HTTP POST deliveries rather than polling:

- Each delivery carries `X-Nexul-Signature` (an HMAC over the payload, `sha256=<hex>`, keyed to the integration's own webhook secret) and `X-Nexul-Delivery-Id`, so a receiver can verify authenticity and dedupe retries.
- Delivery is backed by the transactional outbox with retry and a dead-letter queue — durable, at-least-once.
- Every event topic has a published, versioned JSON Schema. The full catalog is served at:

```
GET /api/events/catalog
```

Installing an integration (today an API call, `POST /api/integrations`; a store with a consent screen is planned) mints it a scoped token (never a raw connector credential) with the least-privilege scopes the install requested, recorded alongside the integration's trust tier (`verified` or `community`). Scopes come from:

```
GET /api/integrations/scopes
```

which returns the identical values `GET /api/permissions/catalog` does, restricted to what that domain's routes actually expose — a scoped token can be granted exactly what a role can. Every action taken through a token, and by which token, is recorded in the audit log:

```
GET /api/audit
```
