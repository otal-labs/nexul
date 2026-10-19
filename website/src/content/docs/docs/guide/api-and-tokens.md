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

A permission is `<domain>:<action>`, where the action is `read`, `write` or `delete`, or a verb such as `plays:run` (what a token needs to start a play) or `stacks:logs`. Roles, tokens and agents are all checked against the same list:

```
GET /api/permissions/catalog
```

Each permission in the catalog has an area: `project`, `workspace` or `instance`. Most take their domain's, but `projects:create` and `memories:template` are workspace permissions, since creating a project and editing the workspace's Interview template happen outside any project. The catalog's `instance_areas` lists the [instance areas](/docs/guide/people-and-access/#instance-permissions) a rule may sit on, each with its `key`, `label` and the `actions` its rules may set.

- Edit roles in **Access → Roles**. Every workspace has an **Owner** role that holds everything and can't be removed; every other role is yours to define.
- A role's project areas count only inside the projects a member is in, and each project's rules say who is in it, by role or by name. See [People and access](/docs/guide/people-and-access/).

`GET /api/workspaces/{workspaceID}/me` lists the roles the caller holds there in `roles`, and `role_name` joins their names for a client that reads one. `permissions` holds what they hold outside any project, and `projects` lists each project they're in with the project-area actions they hold there. `restricted` is true for everyone but the Owner, so a client from before projects were opt-in answers each project from that list. `GET /api/team` lists each person's `roles` per workspace the same way, with `role_name` joining them. `GET /api/permissions/explain` says what decided each of a person's permissions on a doc, play, or project; see [Why someone can or can't](/docs/guide/people-and-access/#why-someone-can-or-cant).

The member routes under `/api/workspaces/{workspaceID}/members` take `role_ids`, the roles to give or, on `PATCH`, the whole set, which may be empty; `role_id` still works as a set of one. An invitation's grants take `role_ids` and list `roles`. `GET /api/permissions?resource_type=project&resource_id=<id>` lists a project's rules, each naming a person or a role, for someone holding `projects:access` there; `resource_type=instance_area` with an area's key lists that area's rules, for the instance Owner. `instance_permissions` on `GET /api/auth/me` lists each instance permission you hold on at least one instance area. `instance_owner` there says whether you are the instance Owner, who alone sets those rules. The per-person fields `allow`, `deny`, `every_project` and `project_access` still parse and read as empty; a non-empty value is refused with a message naming what replaced it.

To copy a custom role into another workspace, choose **Clone to workspace…** in its **…** menu in **Access → Roles**. That takes `roles:clone` in this workspace and `roles:write` in the other. A clashing name gets a suffix, so `Editors` arrives as `Editors (copy)`. Over the API it is `POST /api/workspaces/{workspaceID}/roles/{roleID}/clone` with `{"workspace_id": "<target>"}`; over MCP, `role_update` with `clone_from_id`.

## Integrations and outgoing webhooks

An integration is a service of yours that receives events and calls the API back. Register one with `POST /api/integrations`. It gets a scoped token (`int_`) with only the permissions you grant, from the same values as the catalog:

```
GET /api/integrations/scopes
```

Events arrive as signed HTTP POSTs:

- `X-Nexul-Signature` holds `sha256=<hex>`, an HMAC of the body keyed with the integration's webhook secret. Check it before trusting the payload.
- `X-Nexul-Delivery-Id` is unique per delivery. A failed delivery is retried, so use it to drop duplicates.
- Nothing from a direct message or a private channel is ever sent.

Every event has a versioned JSON Schema in `GET /api/events/catalog`. A list in a payload is always an array, `[]` when empty, never `null`. Every change made through the API or by an agent over MCP, with a token or a session, is recorded in `GET /api/audit` and kept for 45 days. Reads are not recorded.

## Signing in a phone

1. In **Settings → Security → Devices**, click **Generate code** on **Connect a phone**.
2. Open the Nexul app on your phone and scan the QR code.

The code works once, for two minutes. Only a signed-in device can make one, never a token, so no agent can sign a phone in. **Signed-in devices** lists every browser, desktop app and phone on your account. **Sign out** one, or **Sign out everywhere else**.

## Connection tokens

A connection token points a client at your instance without you typing its address. It holds the instance URL and a few settings, nothing about you, so it isn't secret. Copy one with **Copy connection token** under **Settings → Security → Devices**, and paste it into the [desktop app](/docs/guide/desktop-app/).
