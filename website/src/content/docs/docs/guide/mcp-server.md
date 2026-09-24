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

**Tools** — one per use-case, grouped by domain:

| Domain | Tools |
|---|---|
| Docs | `doc_create`, `doc_get`, `doc_search`, `doc_update`, `doc_archive`, `doc_restore` |
| Tickets | `ticket_create`, `ticket_get`, `ticket_update`, `ticket_update_status`, `ticket_set_type`, `ticket_set_developer`, `ticket_set_tester`, `ticket_add_label`, `ticket_remove_label`, `ticket_list_labels`, `ticket_list_all_labels`, `ticket_set_label_color`, `ticket_label_colors`, `ticket_search`, `ticket_link_pr`, `ticket_link_branch`, `ticket_get_links`, `ticket_get_ticket_links`, `ticket_set_found_in`, `ticket_remove_found_in`, `ticket_add_blocker`, `ticket_remove_blocker`, `ticket_list_blocked`, `ticket_get_test_target`, `ticket_test_pass`, `ticket_test_fail` |
| Projects and board | `project_create`, `project_get`, `project_list`, `project_rename`, `project_delete`, `project_reorder`, `project_delete_impact`, `project_add_repo`, `project_remove_repo`, `project_list_repos`, `project_set_tests_location`, `project_move_ticket`, `category_create`, `category_get`, `category_list`, `category_rename`, `category_delete`, `category_reorder`, `category_move_ticket`, `category_clear_ticket`, `ticket_type_create`, `ticket_type_list`, `ticket_type_rename`, `ticket_type_set_template`, `ticket_type_delete`, `status_create`, `status_list`, `status_rename`, `status_reorder`, `status_delete` |
| Deploys and stacks | `deploy_get`, `deploy_log`, `deploy_list`, `deploy_list_by_service`, `deploy_list_by_status`, `deploy_cancel`, `service_list`, `machine_import`, `stack_create`, `stack_deploy`, `stack_get`, `stack_list`, `stack_update`, `stack_delete`, `stack_rollback` |
| Repositories, runners, and machines | `repository_list`, `repository_scan`, `runner_list`, `runner_queue`, `machine_list`, `machine_discover`, `instance_upgrade_status`, `instance_upgrade` |
| DNS and gateways | `dns_verify_credentials`, `dns_list_zones`, `dns_list_records`, `dns_create_record`, `dns_update_record`, `dns_delete_record`, `dns_check_propagation`, `dns_list_service_hostnames`, `dns_get_service_hostname`, `dns_set_service_hostname`, `dns_remove_service_hostname`, `dns_tunnel_create`, `dns_tunnel_list`, `dns_tunnel_get`, `dns_tunnel_route`, `dns_tunnel_rotate`, `dns_tunnel_delete`, `dns_tunnel_provision_agent`, `dns_provision_reverse_proxy`, `dns_gateway_create`, `dns_gateway_list`, `dns_gateway_delete`, `exposure_create`, `dns_exposure_list`, `exposure_delete` |
| Topology | `topology_get`, `topology_add_node`, `topology_remove_node`, `topology_add_edge`, `topology_remove_edge` |
| Git and code review | `git_list_prs`, `git_get_pr`, `git_get_change_context`, `review_list_by_ticket`, `review_get` |
| Chat and mentions | `chat_list_conversations`, `chat_list_messages`, `doc_thread_get`, `interview_thread_get`, `chat_post_message`, `mention_search`, `mention_resolve` |
| Automations | `automation_create`, `automation_list`, `automation_get`, `automation_update_config`, `automation_set_enabled`, `automation_delete`, `automation_mint_token`, `automation_revoke_token` |
| Notifications | `notification_list`, `notification_mark_read`, `notification_mark_all_read` |
| Access | `access_list_grants`, `access_set_grants` |
| Invitations | `invitation_create`, `invitation_list`, `invitation_revoke` |
| Accounts | `account_whoami`, `account_list`, `account_disable`, `account_reactivate`, `account_remove`, `account_restore` |
| Computer pairing | `computer_tunnel_create`, `computer_tunnel_status_get`, `computer_tunnel_token_get`, `computer_pair` |
| Computer setup | `computer_setup_start`, `computer_setup_retry_provider`, `computer_setup_get`, `computer_setup_confirm_provider`, `computer_setup_unconfirm_provider`, `computer_setup_confirm`, `computer_setup_unconfirm` |
| Computer MCP tokens | `computer_mcp_token_get`, `computer_mcp_token_mint`, `computer_mcp_token_revoke` |
| Plays | `play_list`, `play_create`, `play_update`, `play_delete`, `play_run`, `play_run_get`, `play_run_stop`, `play_run_answer`, `play_list_runs`, `decisions_check_run` |
| Memories | `memory_list`, `memory_get`, `memory_create`, `memory_update`, `memory_delete`, `memory_list_versions`, `memory_revert`, `memory_clone`, `memory_create_interview`, `interview_template_get`, `interview_template_update` |
| Dead letters | `dead_letter_list`, `dead_letter_replay` |

The workflow prompts are not tools. They template common sequences and are
listed separately below. Deploys and rollbacks started through MCP are
attributed to the token's user, with the source suffixed `:mcp`.

`computer_pair` takes the one-time token `t3 pair` prints. With `computer_id`
it pairs a computer tunnel over its hostname, once `computer_tunnel_status_get`
reports both checks passing. With `name` and `server_url` instead, it pairs a
machine the server can already reach by URL.

`computer_setup_start` runs the same setup turns as the **Set up** step of the
pairing dialog and returns at once; `computer_setup_retry_provider` runs one
provider's turn again. Progress arrives as `computer.setup_turn_changed` and
`computer.setup_finished` events. The confirmations themselves are still made
only by the agent in each turn, through the confirm tools.

`git_get_change_context` answers "why does this code exist". Give it the
repository and a pull request number, or a commit SHA from `git blame`, and it
returns the pull request, the tickets linked to it, each ticket's doc and the
bugs found in it after it was done, and the decisions-log entries citing those
tickets. The same walk is `GET /api/repos/{owner}/{repo}/change-context` with
`?pr=` or `?commit=`.

`decisions_check_run` reruns the decisions check on a done ticket whose check
didn't run, on the caller's own paired computer.

`ticket_create` files a bug (a ticket of the type named `bug`) only with
`origin_id`, the ticket it was found in, or `origin_unknown` set to true, the
same rule the web app applies.

`ticket_get_test_target` returns where to test a ticket: the preview of a
branch linked to it, else a shared test environment that may include other
changes. It never returns production, meaning the default branch's
deployment or any branch deployment on production's network with nothing
overridden; an empty `url` means no safe environment exists yet.
`ticket_test_pass` moves the ticket to the first done-stage column, makes the
caller its tester only when none is assigned, and posts "Passed by Nexul · for
<login>", with the test URL, to the ticket's thread. `ticket_test_fail` posts
the steps, expected result, actual result, and screenshot attachment ids to
the ticket's thread under "Test failed by Nexul · for <login>" and moves it
back to the first progress-stage column; it refuses a done ticket, which takes
a new bug found in it instead. Both record the move as made by `user:mcp`. The
Pass and Fail buttons sign with the person's own login instead.

**Resources** expose readable entities by URI: `docs://{id}`, `tickets://{id}`, and `topology://current`.

**Prompts** template common workflows: `create_ticket_from_doc`, `deploy_stack` (deploy and watch until healthy or failed), `investigate_failure`, and `ship_repository` (list repositories, scan one, create a stack, deploy it, and expose it — the wizard's own steps, callable end to end).

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

Settings also shows ready-to-copy client snippets once you have set an instance
URL. Paste the minted token into the snippet for your client.

## Protocol revisions

The server speaks two MCP protocol revisions side by side: **2026-07-28** (the stateless revision) and **2025-03-26**. Which one a request gets is decided per-request by the `_meta["io.modelcontextprotocol/protocolVersion"]` field; omitting `_meta` gets you the legacy 2025-03-26 behavior. The server has been stateless from the start — no session store, no held-open streams — so nothing had to be removed to support the newer revision. `initialize`, `notifications/initialized`, and `ping` are kept around for as long as clients still rely on them.
