---
title: API and Tokens
description: The HTTP/JSON API, its permission model, and the tokens that authenticate against it.
sidebar:
  order: 13
---

The web UI uses the plain HTTP/JSON API under `/api`. MCP is a separate
adapter over the same use-case layer.

## OpenAPI

The server builds an OpenAPI 3.x document from the gateway's tracked mounted
routes. Maintained summaries and tags are applied as annotations to those
routes. The document is not a manually aligned route manifest or a separate
YAML file.

- `/openapi.json` — the raw spec.
- `/swagger` — a Swagger UI browsing it.

## Permissions

One vocabulary covers every actor: a permission is `<domain>:<action>`, where
the action is usually `read`, `write`, or `delete`. Some domains also declare
a verb: `docs:thread`, `plays:run`, `memories:clone`, and `roles:clone`. The
full catalog is served at:

```
GET /api/permissions/catalog
```

A role, a personal access token, and an automation's scoped token are all checked against these same values — "what can this actor do" has one answer regardless of who's asking.

**Roles** live per workspace membership. Every workspace has a singleton, unremovable **Owner** role that implicitly holds every permission. Beyond that, roles are fully custom: anyone holding `roles:write` can create a role with whichever permissions it needs.

A custom role can be cloned into another workspace you belong to, from the Clone button on its row in **Settings → Roles**, or with `POST /api/workspaces/{workspaceID}/roles/{roleID}/clone` and a body of `{"workspace_id": "<target>"}`. The copy keeps the role's name and permissions. Cloning needs `roles:clone` (the Clone toggle on the Roles row of the role editor) in the role's own workspace and `roles:write` in the target; a workspace Owner has both. A name the target already uses gets a suffix instead of failing: `Editors` arrives as `Editors (copy)`, then `Editors (copy 2)`, and so on. The Owner role can't be cloned, since every workspace has its own. Over MCP, `workspace_list` with a workspace's `id` returns its roles and the permission catalog, `role_update` creates, clones (with `clone_from_id`), or edits a role, and `role_delete` removes one.

## Personal access tokens

Mint one from **Your settings → Security → Tokens**. A PAT is long-lived, revocable, and carries exactly your own permissions — it's the credential an agent or a personal integration uses to act as you, including for the [MCP server](/docs/guide/mcp-server/). The raw token (prefixed `dep_`) is shown once at creation and can never be retrieved again; revoking it cuts access immediately.

A paired computer gets its own PAT, "Nexul MCP on <computer>", minted from the computer's row in **Your settings → T3 pairing**. It is listed here marked as the computer's, one is active per computer, and un-confirming the computer's setup or removing the computer revokes it.

## Connection tokens

A **connection token** is different: it carries no identity or credentials at all, just server information (the instance URL, derived MCP endpoint, and basic settings) as a signed JWT. It exists so a standalone client — the [desktop app](/docs/guide/desktop-app/) today — can be pointed at your instance without you typing a URL by hand. Generate one from **Your settings → Security → Tokens**; after importing it, the client still signs you in through the normal GitHub OAuth flow.

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
