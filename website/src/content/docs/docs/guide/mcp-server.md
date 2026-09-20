---
title: MCP Server
description: Connect an agent tool to your instance over MCP — what it can do and how to authenticate it.
sidebar:
  order: 11
---

Nexul ships a first-class MCP server so an LLM agent can drive the same workflows the web UI does — docs, tickets, deploys, topology, runners — through one adapter over the same use-case layer. An agent gets nothing the UI doesn't get, and every action it takes is attributed to the user whose token it authenticates with.

## What it exposes

**Tools** — one per use-case, grouped by domain:

| Domain | Tools |
|---|---|
| Docs | `doc_create`, `doc_get`, `doc_update`, `doc_archive`, `doc_restore`, `doc_search`, `search_docs` |
| Tickets | `ticket_create`, `ticket_get`, `ticket_update`, `ticket_update_status`, `ticket_set_type`, `ticket_add_label`, `ticket_remove_label`, `ticket_list_labels`, `ticket_list_all_labels`, `ticket_set_label_color`, `ticket_label_colors`, `ticket_link_branch`, `ticket_link_pr`, `ticket_get_links`, `ticket_search`, `search_tickets` |
| Projects & board | `project_create`, `project_get`, `project_list`, `project_rename`, `project_delete`, `project_reorder`, `project_delete_impact`, `project_add_repo`, `project_remove_repo`, `project_list_repos`, `project_move_ticket`, `category_create`, `category_get`, `category_list`, `category_rename`, `category_delete`, `category_reorder`, `category_move_ticket`, `category_clear_ticket`, `ticket_type_create`, `ticket_type_list`, `ticket_type_rename`, `ticket_type_delete`, `status_create`, `status_list`, `status_rename`, `status_reorder`, `status_delete` |
| Deploys & stacks | `stack_create`, `stack_get`, `stack_list`, `stack_update`, `stack_delete`, `stack_deploy`, `stack_rollback`, `service_list`, `deploy_get`, `deploy_list`, `deploy_list_by_service`, `deploy_list_by_status`, `deploy_cancel` |
| Repositories & machines | `repository_list`, `repository_scan`, `machine_list`, `machine_discover`, `machine_import`, `runner_list`, `runner_queue`, `instance_upgrade_status`, `instance_upgrade` |
| DNS & gateways | `dns_list_zones`, `dns_list_records`, `dns_create_record`, `dns_update_record`, `dns_delete_record`, `dns_check_propagation`, `dns_gateway_list`, `dns_gateway_create`, `dns_gateway_delete`, `dns_exposure_list`, `exposure_create`, `exposure_delete`, `dns_list_service_hostnames`, `dns_get_service_hostname`, `dns_set_service_hostname`, `dns_remove_service_hostname`, `dns_tunnel_list`, `dns_tunnel_get`, `dns_tunnel_create`, `dns_tunnel_delete`, `dns_tunnel_rotate`, `dns_tunnel_route`, `dns_tunnel_provision_agent`, `dns_provision_reverse_proxy`, `dns_verify_credentials` |
| Topology | `topology_get`, `topology_add_node`, `topology_remove_node`, `topology_add_edge`, `topology_remove_edge` |
| Git & code review | `git_list_prs`, `git_get_pr`, `review_get`, `review_list_by_ticket` |
| Chat & mentions | `chat_list_conversations`, `chat_list_messages`, `chat_post_message`, `mention_resolve`, `mention_search` |
| Automations | `automation_create`, `automation_get`, `automation_list`, `automation_update_config`, `automation_set_enabled`, `automation_delete`, `automation_mint_token`, `automation_revoke_token` |
| Notifications | `notification_list`, `notification_mark_read`, `notification_mark_all_read` |
| Access | `access_list_grants`, `access_set_grants` |
| Operations | `list_dead_letters`, `replay_dead_letter` |

Deploys and rollbacks started through MCP are attributed to the token's user, the same as the UI, with the source suffixed `:mcp` so it's clear where the trigger came from — an agent can trigger a real deploy or rollback, not just read state.

**Resources** expose readable entities by URI: `docs://{id}`, `tickets://{id}`, and `topology://current`.

**Prompts** template common workflows: `create_ticket_from_doc`, `deploy_stack` (deploy and watch until healthy or failed), `investigate_failure`, and `ship_repository` (list repositories, scan one, create a stack, deploy it, and expose it — the wizard's own steps, callable end to end).

## Connecting

The MCP endpoint is your instance URL at `/mcp`, over Streamable HTTP:

```
https://your-instance.example.com/mcp
```

It authenticates with a personal access token (a session token also works from the browser) — see [API and Tokens](/docs/guide/api-and-tokens/) for how permissions work. Mint one:

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

Settings also shows a ready-to-copy version of the Claude Code and opencode snippets once you've set an instance URL — paste your minted token in and copy the block for your tool.

## Protocol revisions

The server speaks two MCP protocol revisions side by side: **2026-07-28** (the stateless revision) and **2025-03-26**. Which one a request gets is decided per-request by the `_meta["io.modelcontextprotocol/protocolVersion"]` field; omitting `_meta` gets you the legacy 2025-03-26 behavior. The server has been stateless from the start — no session store, no held-open streams — so nothing had to be removed to support the newer revision. `initialize`, `notifications/initialized`, and `ping` are kept around for as long as clients still rely on them.
