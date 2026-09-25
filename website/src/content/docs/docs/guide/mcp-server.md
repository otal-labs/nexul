---
title: MCP Server
description: Connect an agent tool to your instance over MCP — what it can do and how to authenticate it.
sidebar:
  order: 11
---

Nexul ships an MCP server over the same use-case layer as the web UI. The
registry exposes the tools listed below, and every call is attributed to the
user whose token authenticates the connection.

## What it exposes

**Tools** — 97 of them, each covering one task an agent does rather than one
button, named `<object>_<verb>`. Updates are patches: send only the fields
you mean to change, and omitted ones keep their values. Lists take `limit`
and `offset` and return `items`, `total`, `has_more`, and `next_offset`.

| Area | Tools |
|---|---|
| Workspaces and projects | `workspace_list`, `project_list`, `project_get`, `project_create`, `project_update`, `project_delete` |
| Tickets | `ticket_list`, `ticket_get`, `ticket_create`, `ticket_update`, `ticket_delete`, `ticket_test_report` |
| Docs | `doc_list`, `doc_get`, `doc_create`, `doc_update` |
| Memories | `memory_list`, `memory_get`, `memory_create`, `memory_update`, `memory_delete`, `interview_template_get`, `interview_template_update` |
| Chat and notifications | `conversation_list`, `message_list`, `message_post`, `mention_search`, `notification_list`, `notification_update` |
| Stacks and deploys | `stack_list`, `stack_get`, `stack_create`, `stack_update`, `stack_delete`, `stack_deploy`, `deploy_list`, `deploy_get`, `deploy_cancel` |
| Machines and the instance | `machine_list`, `machine_discover`, `machine_import`, `instance_get`, `instance_upgrade` |
| DNS and routing | `dns_zone_list`, `dns_record_list`, `dns_record_create`, `dns_record_update`, `dns_record_delete`, `dns_tunnel_list`, `dns_tunnel_create`, `dns_tunnel_update`, `dns_tunnel_delete`, `gateway_list`, `gateway_create`, `gateway_delete`, `exposure_list`, `exposure_create`, `exposure_delete` |
| Topology | `topology_get`, `topology_update` |
| Repositories and pull requests | `repository_list`, `repository_scan`, `pull_request_list`, `pull_request_get` |
| Plays | `play_list`, `play_create`, `play_update`, `play_delete`, `play_run`, `trail_list`, `trail_update`, `decisions_check_run` |
| Automations | `automation_list`, `automation_create`, `automation_update`, `automation_delete`, `automation_token_create` |
| Paired computers | `computer_list`, `computer_create`, `computer_pair`, `computer_delete`, `computer_tunnel_token_get`, `computer_setup_run`, `computer_setup_update`, `computer_mcp_token_create`, `computer_mcp_token_delete` |
| Accounts and access | `account_get`, `account_list`, `account_update`, `account_delete`, `invitation_create`, `invitation_list`, `invitation_delete`, `permission_overwrite_list`, `permission_overwrite_update` |
| Failed events | `dead_letter_list`, `dead_letter_replay` |

Every tool carries hints a client uses to decide what to confirm: the reads
are marked read-only, and deletes and anything that starts work are marked
destructive. A tool's failure comes back as an error result the agent can
read and act on, with the fields it needs to fix the call. Deploys,
rollbacks, and other work started through MCP are attributed to the token's
user, with the source suffixed `:mcp`. Instance upgrades and failed events
are for instance admins only, as they are in the web app.

`computer_pair` takes the one-time token `t3 pair` prints. With `id` it pairs
a computer tunnel over its hostname, once `computer_list` with that id reports
both tunnel checks passing. With `name` and `server_url` instead, it pairs a
machine the server can already reach by URL.

`computer_setup_run` runs the same setup turns as the **Set up** step of the
pairing dialog and returns at once. It takes an optional `models` object
mapping a provider's driver kind to a model slug; with `provider` it runs
that one provider's turn again, on an optional `model`. A provider without a
model runs on its own default. Progress arrives as
`computer.setup_turn_changed` and `computer.setup_finished` events. The
confirmations themselves are made only by the agent in each turn, through
`computer_setup_update`.

`pull_request_get` answers "why does this code exist". Give it the repository
and a pull request number, or a commit SHA from `git blame`, and it returns
the pull request, the tickets linked to it, each ticket's doc and the bugs
found in it after it was done, and the decisions-log entries citing those
tickets. The same walk is `GET /api/repos/{owner}/{repo}/change-context` with
`?pr=` or `?commit=`.

`decisions_check_run` reruns the decisions check on a done ticket whose check
didn't run, on the caller's own paired computer.

`ticket_create` files a bug (a ticket of the type named `bug`) only with
`origin_id`, the ticket it was found in, or `origin_unknown` set to true, the
same rule the web app applies.

`ticket_get` includes the ticket's `test_target`: the preview of a branch
linked to it, else a shared test environment that may include other changes.
It never points at production, meaning the default branch's deployment or any
branch deployment on production's network with nothing overridden; an empty
`url` means no safe environment exists yet. `ticket_test_report` with
`outcome` `pass` moves the ticket to the first done-stage column, makes the
caller its tester only when none is assigned, and posts "Passed by Nexul ·
for <login>", with the test URL, to the ticket's thread. With `outcome`
`fail` it posts the steps, expected result, actual result, and screenshot
attachment ids to the ticket's thread under "Test failed by Nexul · for
<login>" and moves it back to the first progress-stage column; it refuses a
done ticket, which takes a new bug found in it instead. Both record the move
as made by `user:mcp`. The Pass and Fail buttons sign with the person's own
login instead.

**Resources** expose readable entities by URI, for attaching one to a
conversation: `docs://{id}`, `tickets://{id}`, and `topology://current`.

**Prompts** template common workflows: `create_ticket_from_doc`,
`deploy_and_watch_stack` (deploy and watch until healthy or failed),
`investigate_failure`, and `ship_repository` (list repositories, scan one,
create a stack, deploy it, and expose it — the wizard's own steps, callable
end to end).

## Connecting

The MCP endpoint is your instance URL at `/mcp`, over Streamable HTTP:

```
https://your-instance.example.com/mcp
```

It authenticates with a personal access token. A browser session token also
works for an interactive session. See [API and tokens](/docs/guide/api-and-tokens/)
for the permission vocabulary. Mint one:

1. Open **Settings → Personal access tokens**.
2. Give it a name and create it. The raw token (`dep_…`) is shown once — copy it now, it can't be listed again later.

A paired computer needs none of the steps below: its
[setup wizard](/docs/guide/computer-setup/) connects each provider with the
computer's own token, "Nexul MCP on <computer>", and installs the skills. The
token is listed as **MCP token** on the computer's row in **Settings → T3
pairing**. Un-confirming the computer's setup or removing the computer revokes
it.
Nexul replaces any `dep_` token with `[redacted token]` before it saves a play's
trail or an agent's reply.

### Claude Code

```sh
claude mcp add --transport http nexul https://your-instance.example.com/mcp \
  --header "Authorization: Bearer dep_your_token_here"
```

Or drop it straight into `.mcp.json`:

```json
{
  "mcpServers": {
    "nexul": {
      "type": "http",
      "url": "https://your-instance.example.com/mcp",
      "headers": { "Authorization": "Bearer dep_your_token_here" }
    }
  }
}
```

### opencode

Add the same shape to `opencode.json`:

```json
{
  "mcp": {
    "nexul": {
      "type": "remote",
      "url": "https://your-instance.example.com/mcp",
      "headers": { "Authorization": "Bearer dep_your_token_here" },
      "enabled": true
    }
  }
}
```

### Codex

Add an MCP server entry to `~/.codex/config.toml`:

```toml
[mcp_servers.nexul]
url = "https://your-instance.example.com/mcp"
http_headers = { Authorization = "Bearer dep_your_token_here" }
```

## Protocol revisions

The server runs on the official MCP Go SDK and speaks the **2026-07-28**
revision statelessly, per request, alongside the earlier revisions that open
with an `initialize` handshake (2025-11-25 back to 2024-11-05), so clients on
either side of the change connect. It answers `POST /mcp` with a single JSON
response and keeps no sessions or open streams. A browser request is accepted
only from your instance's own URL, and a missing or invalid token gets `401`
with a `WWW-Authenticate: Bearer` challenge.
