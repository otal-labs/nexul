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
| Tickets | `ticket_create`, `ticket_get`, `ticket_update`, `ticket_update_status`, `ticket_set_type`, `ticket_set_developer`, `ticket_set_tester`, `ticket_add_label`, `ticket_remove_label`, `ticket_list_labels`, `ticket_list_all_labels`, `ticket_set_label_color`, `ticket_label_colors`, `ticket_search`, `ticket_link_pr`, `ticket_link_branch`, `ticket_get_links`, `ticket_get_ticket_links`, `ticket_set_found_in`, `ticket_remove_found_in`, `ticket_add_blocker`, `ticket_remove_blocker`, `ticket_list_blocked` |
| Projects and board | `project_create`, `project_get`, `project_list`, `project_rename`, `project_delete`, `project_reorder`, `project_delete_impact`, `project_add_repo`, `project_remove_repo`, `project_list_repos`, `project_move_ticket`, `category_create`, `category_get`, `category_list`, `category_rename`, `category_delete`, `category_reorder`, `category_move_ticket`, `category_clear_ticket`, `ticket_type_create`, `ticket_type_list`, `ticket_type_rename`, `ticket_type_set_template`, `ticket_type_delete`, `status_create`, `status_list`, `status_rename`, `status_reorder`, `status_delete` |
| Deploys and stacks | `deploy_get`, `deploy_log`, `deploy_list`, `deploy_list_by_service`, `deploy_list_by_status`, `deploy_cancel`, `service_list`, `machine_import`, `stack_create`, `stack_deploy`, `stack_get`, `stack_list`, `stack_update`, `stack_delete`, `stack_rollback` |
| Repositories, runners, and machines | `repository_list`, `repository_scan`, `runner_list`, `runner_queue`, `machine_list`, `machine_discover`, `instance_upgrade_status`, `instance_upgrade` |
| DNS and gateways | `dns_verify_credentials`, `dns_list_zones`, `dns_list_records`, `dns_create_record`, `dns_update_record`, `dns_delete_record`, `dns_check_propagation`, `dns_list_service_hostnames`, `dns_get_service_hostname`, `dns_set_service_hostname`, `dns_remove_service_hostname`, `dns_tunnel_create`, `dns_tunnel_list`, `dns_tunnel_get`, `dns_tunnel_route`, `dns_tunnel_rotate`, `dns_tunnel_delete`, `dns_tunnel_provision_agent`, `dns_provision_reverse_proxy`, `dns_gateway_create`, `dns_gateway_list`, `dns_gateway_delete`, `exposure_create`, `dns_exposure_list`, `exposure_delete` |
| Topology | `topology_get`, `topology_add_node`, `topology_remove_node`, `topology_add_edge`, `topology_remove_edge` |
| Git and code review | `git_list_prs`, `git_get_pr`, `review_list_by_ticket`, `review_get` |
| Chat and mentions | `chat_list_conversations`, `chat_list_messages`, `doc_thread_get`, `chat_post_message`, `mention_search`, `mention_resolve` |
| Automations | `automation_create`, `automation_list`, `automation_get`, `automation_update_config`, `automation_set_enabled`, `automation_delete`, `automation_mint_token`, `automation_revoke_token` |
| Notifications | `notification_list`, `notification_mark_read`, `notification_mark_all_read` |
| Access | `access_list_grants`, `access_set_grants` |
| Invitations | `create_invitation`, `list_invitations`, `revoke_invitation` |
| Accounts | `account_whoami`, `list_accounts`, `disable_account`, `reactivate_account`, `remove_account`, `restore_account` |
| Computer setup | `computer_setup_get`, `computer_setup_confirm_provider`, `computer_setup_unconfirm_provider`, `computer_setup_confirm`, `computer_setup_unconfirm` |
| Plays | `play_list`, `play_create`, `play_update`, `play_delete`, `play_run`, `play_run_get`, `play_run_stop`, `play_run_answer`, `play_list_runs` |
| Memories | `memory_list`, `memory_get`, `memory_create`, `memory_update`, `memory_delete`, `memory_list_versions`, `memory_revert`, `memory_clone` |
| Search and operations | `search_docs`, `search_tickets`, `list_dead_letters`, `replay_dead_letter` |

The workflow prompts are not tools. They template common sequences and are
listed separately below. Deploys and rollbacks started through MCP are
attributed to the token's user, with the source suffixed `:mcp`.

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
