---
title: API and tokens
description: Call the HTTP API with a token, see what each token can do, and sign in a phone or desktop app.
sidebar:
  order: 13
---

The web app runs on the same HTTP/JSON API under `/api` that you can call yourself. The [MCP server](/docs/guide/mcp-server/) sits on the same rules, so a person, a script and an agent can each do exactly what their permissions allow.

## Calling the API

1. Open **Settings → Security → Tokens**, name a token and click **Create token**.
2. Copy it. It starts with `dep_` and is shown once.
3. Send it as a bearer token:

   ```sh
   curl -H "Authorization: Bearer dep_…" https://nexul.example.com/api/permissions/catalog
   ```

The list shows each token as **Active** with when it was last used, or **Revoked**. A personal access token carries exactly your permissions and lasts until you revoke it. Revoking cuts access at once. Use one for scripts, personal integrations and an agent's MCP connection.

Each paired computer also gets a token of its own, "Nexul MCP on <computer>", listed here and marked as the computer's. Un-confirming the computer's setup or removing the computer revokes it.

Browse the API at `/swagger`, or fetch the OpenAPI document at `/openapi.json`. Both are built from the routes the server actually mounts.

## Permissions

A permission is `<domain>:<action>`, where the action is `read`, `write` or `delete`, or a verb such as `plays:run` or `stacks:logs`. Roles, tokens and agents are all checked against the same list:

```
GET /api/permissions/catalog
```

Each domain in the catalog has an area: `project`, `workspace` or `instance`.

- Edit roles in **Configuration → Roles**. Every workspace has an **Owner** role that holds everything and can't be removed; every other role is yours to define.
- A role's **Every project** levels apply to all projects, new ones included, for members whose **Every project** is **From role**.
- A member set to **Chosen projects** sees only the projects you give them on **Team**, each at the level you set there. Their role still covers the workspace areas but opens no instance area.
- Instance areas, such as runners, DNS and connectors, are checked against every workspace you belong to (except where you are on chosen projects only). Holding one in any of them is enough.

`GET /api/workspaces/{workspaceID}/me` tells a client whether the caller is on chosen projects and, if so, which projects with which actions.

To copy a custom role into another workspace, choose **Clone to workspace…** in its **…** menu in **Configuration → Roles**. That takes `roles:clone` in this workspace and `roles:write` in the other. A clashing name gets a suffix, so `Editors` arrives as `Editors (copy)`. Over the API it is `POST /api/workspaces/{workspaceID}/roles/{roleID}/clone` with `{"workspace_id": "<target>"}`; over MCP, `role_update` with `clone_from_id`.

## Integrations and outgoing webhooks

An integration is a service of yours that receives events and calls the API back. Register one with `POST /api/integrations`. It gets a scoped token (`int_`) with only the permissions you grant, from the same values as the catalog:

```
GET /api/integrations/scopes
```

Events arrive as signed HTTP POSTs:

- `X-Nexul-Signature` holds `sha256=<hex>`, an HMAC of the body keyed with the integration's webhook secret. Check it before trusting the payload.
- `X-Nexul-Delivery-Id` is unique per delivery. A failed delivery is retried, so use it to drop duplicates.
- Nothing from a direct message or a private channel is ever sent.

Every event has a versioned JSON Schema in `GET /api/events/catalog`. Every action taken with a token is recorded in `GET /api/audit`.

## Signing in a phone

1. In **Settings → Security → Devices**, click **Generate code** on **Connect a phone**.
2. Open the Nexul app on your phone and scan the QR code.

The code works once, for two minutes. Only a signed-in device can make one, never a token, so no agent can sign a phone in. **Signed-in devices** lists every browser, desktop app and phone on your account. **Sign out** one, or **Sign out everywhere else**.

## Connection tokens

A connection token points a client at your instance without you typing its address. It holds the instance URL and a few settings, nothing about you, so it isn't secret. Copy one with **Copy connection token** under **Settings → Security → Devices**, and paste it into the [desktop app](/docs/guide/desktop-app/).
