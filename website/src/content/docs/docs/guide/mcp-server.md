---
title: MCP server
description: Connect Claude Code, Codex, opencode, or any MCP client to your instance so an agent can work in Nexul with your permissions.
sidebar:
  order: 11
---

Every Nexul instance has an MCP server at `/mcp`. Connect your agent tool to it and the agent can read docs, file tickets, post in chat, deploy stacks, and everything else in the table below, acting as you, with your permissions and nothing more.

A [paired computer](/docs/guide/paired-computers/) needs none of this: its [setup](/docs/guide/computer-setup/) connects every provider for you, with the computer's own token. Follow these steps for any other tool or machine.

## Connect

1. Open **Settings → Security → Tokens** and create a token. Copy it now; the raw token (`dep_…`) is shown once.
2. Add the server to your tool, with your instance's URL and the token.

**Claude Code**, in the top-level `mcpServers` object of `~/.claude.json` (your user scope, every project):

```json
{
  "mcpServers": {
    "nexul": {
      "type": "http",
      "url": "https://nexul.example.com/mcp",
      "headers": { "Authorization": "Bearer dep_your_token_here" }
    }
  }
}
```

For one project, use `.mcp.json` in its root. That file is usually committed, so keep the token out of it: Claude Code expands `${VAR}` in `headers`, so set `NEXUL_TOKEN` in your shell and write the header as `"Authorization": "Bearer ${NEXUL_TOKEN}"`.

**A client that starts the server through `mcp-remote`** takes the token from its environment, so it never shows in the process list:

```json
{
  "mcpServers": {
    "nexul": {
      "command": "npx",
      "args": ["mcp-remote", "https://nexul.example.com/mcp", "--header", "Authorization:${NEXUL_AUTH}"],
      "env": { "NEXUL_AUTH": "Bearer dep_your_token_here" }
    }
  }
}
```

Never put a token in a command's arguments, such as `claude mcp add --header` or a literal `--header` for `mcp-remote`: any local user can read it with `ps`, and your shell history keeps it.

**opencode**, in `opencode.json`:

```json
{
  "mcp": {
    "nexul": {
      "type": "remote",
      "url": "https://nexul.example.com/mcp",
      "headers": { "Authorization": "Bearer dep_your_token_here" },
      "enabled": true
    }
  }
}
```

**Codex**, in `~/.codex/config.toml`:

```toml
[mcp_servers.nexul]
url = "https://nexul.example.com/mcp"
http_headers = { Authorization = "Bearer dep_your_token_here" }
```

A missing or wrong token gets `401`. Revoke the token in the same place to cut the agent off.

## What an agent can do

112 tools, each named `<object>_<verb>` and shaped around a task rather than a button.

| Area | Tools |
|---|---|
| Workspaces and projects | `workspace_list`, `workspace_update`, `project_list`, `project_get`, `project_create`, `project_update`, `project_delete` |
| Tickets | `ticket_list`, `ticket_get`, `ticket_create`, `ticket_update`, `ticket_delete`, `ticket_test_report` |
| Docs | `doc_list`, `doc_get`, `doc_create`, `doc_update`, `doc_delete` |
| Memories | `memory_list`, `memory_get`, `memory_create`, `memory_update`, `memory_delete` |
| Attachments | `attachment_create`, `attachment_get` |
| Templates | `template_get`, `template_update` |
| Chat and notifications | `conversation_list`, `conversation_update`, `conversation_delete`, `message_list`, `message_post`, `mention_search`, `notification_list`, `notification_update` |
| Bots | `botwebhook_list`, `botwebhook_create`, `botwebhook_update` |
| Calendar | `meeting_list`, `meeting_create`, `meeting_update` |
| Stacks and deploys | `stack_list`, `stack_get`, `stack_create`, `stack_update`, `stack_delete`, `stack_deploy`, `deploy_list`, `deploy_cancel` |
| Machines and the instance | `machine_list`, `machine_discover`, `machine_import`, `host_create`, `host_delete`, `instance_get`, `instance_upgrade` |
| DNS and routing | `dns_zone_list`, `dns_record_list`, `dns_record_create`, `dns_record_update`, `dns_record_delete`, `dns_tunnel_list`, `dns_tunnel_create`, `dns_tunnel_update`, `dns_tunnel_delete`, `gateway_list`, `gateway_create`, `gateway_delete`, `exposure_list`, `exposure_create`, `exposure_delete` |
| Topology | `topology_get`, `topology_update` |
| Repositories and pull requests | `repository_list`, `repository_scan`, `pull_request_list`, `pull_request_get` |
| Plays | `play_list`, `play_create`, `play_update`, `play_delete`, `play_run`, `trail_list`, `trail_update` |
| Automations | `automation_list`, `automation_create`, `automation_update`, `automation_delete`, `automation_token_create` |
| Paired computers and skills | `computer_list`, `computer_create`, `computer_pair`, `computer_delete`, `computer_project_create`, `computer_setup_run`, `computer_setup_update`, `computer_mcp_token_create`, `computer_mcp_token_delete`, `skill_get` |
| Accounts and access | `account_get`, `account_list`, `account_update`, `account_delete`, `invitation_create`, `invitation_list`, `invitation_delete`, `permission_overwrite_list`, `permission_overwrite_update`, `role_update`, `role_delete` |
| Failed events | `dead_letter_list`, `dead_letter_replay` |

A few things that save an agent a wasted call:

- **Updates are patches.** Send only the fields to change; the rest keep their values.
- **Lists page.** They take `limit` (50 by default, 100 at most) and `offset` and return `items`, `total`, `has_more`, and `next_offset`. `total` counts everything you can see, searches included, so paging with `next_offset` reaches every match. A row added ahead of your place while you page shifts the later pages by one.
- **A list given an id returns that one record.** `deploy_list` with a deploy's `id` returns it with its log; `trail_list`, `automation_list`, `computer_list`, and `dns_tunnel_list` work the same way. A field's description lists the values it takes.
- **Tickets by key.** Ticket tools take an id or a key such as `WEB-12`. A key is unique only within a workspace, so when you hold it in two, pass `workspace` too, or the call is refused with the workspaces to choose from.
- **Errors say how to fix the call.** A failed tool returns the fields it needs, not a bare error. A refused one names the role or rule that refused it; `permission_overwrite_list` with a `user_id` explains every answer on a doc, folder, stack, play, project, or instance area.
- **Read-only and destructive hints.** Reads are marked read-only, and deletes and anything that starts work are marked destructive, so your client knows what to confirm.
- **Not found can mean no access.** Someone who sees only some projects gets not found for the rest, and so does their agent.
- **Changes are audited.** Every call to a tool that changes something is recorded in the [audit log](/docs/guide/api-and-tokens/#integrations-and-outgoing-webhooks) under your name and kept for 45 days. Reads are not.

Some tools worth knowing:

- `pull_request_get` answers "why does this code exist". Give it a repository and a pull request number, or a commit SHA from `git blame`, and it returns the pull request, its tickets, each ticket's doc, bugs found after done, and the decisions-log entries citing them.
- `attachment_create` puts a file on a doc, ticket, conversation, or memory, up to 10 MiB, and returns a markdown line to embed it. `attachment_get` reads one back; an image comes back as an image.
- `ticket_get` includes the ticket's test target, and `ticket_test_report` passes or fails it like the **Pass** and **Fail** buttons.
- `instance_get` with a `logs` object also reads the instance's own [logs](/docs/guide/logs/), newest first. It needs `logs:read` besides `instance:read`.
- `play_run` starts a [play](/docs/guide/plays/) on your paired computer.
- `computer_create` adds one of your computers and returns the command that installs its runner: `commands.unix` for Linux and macOS, `curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token>`, and `commands.windows` for Windows, `& ([scriptblock]::Create((irm https://nexul.io/computer.ps1))) <token>`, where the token is good once, for an hour, and only on your instance. Run it on that computer from your own account: on Linux and macOS `sudo` only places the `nexul-computer` system service (a LaunchDaemon on a Mac), which runs the runner as you, never as root; on Windows it needs no administrator and installs a task of yours that runs the runner while you're logged on. Either installs T3 Code when it's missing ([what it installs](/docs/guide/paired-computers/#what-the-command-installs)). Once the runner connects, `computer_list` shows `runner.connected` on the computer; a computer added without a name takes its hostname. Pass `id` instead for a fresh command when the last one expired, or to [move a computer paired through a tunnel or by URL](/docs/guide/paired-computers/#move-a-computer-to-the-nexul-app) to the Nexul app: once it pairs through its runner it keeps its links, setup, MCP token and trails, and only then are its tunnel, DNS record and Access app deleted; `sudo cloudflared service uninstall` on the computer removes `cloudflared` afterwards. A failed move leaves the computer working as it was. Only you can see or use your computers; anyone else, a workspace Owner included, gets not found.
- Once its runner connects and T3 Code answers there, the computer pairs on its own, and pairs again before its 30-day session ends. `computer_pair` with only the computer's `id` pairs it again now, and moves a computer whose runner `computer_create` with its `id` installed. A failed pairing leaves the computer as it was, and `computer_list` says why in `pair_error`. `computer_list` also returns a computer's `facts` once its runner reports (its system, git identity, whether it can sign in to GitHub, free disk, Docker, T3 Code's state and port, and the providers and projects T3 Code lists) with `facts_at`, when they last changed; only you get them. With one computer's `id`, a computer whose facts show Docker available also lists its `containers`, and `container_logs: {container, tail}` adds that container's last lines (200 by default, at most 1000), read live through the computer's runner. `runner_logs: {lines}` adds the last lines the computer's Nexul app itself logged, the same way. A person the computer is shared with by `computers:read_logs` gets both reads and the container list too, with each container's own environment values hidden.
- `computer_pair` with `position`, `runs_at_once` or `chat_project_id` changes the computer's settings instead of pairing it: its place in [your order](/docs/guide/paired-computers/#your-computers-in-order) (0 for first), how many play runs it takes at a time (1 to 5), or the T3 project chats outside any project run in. `computer_list` returns each computer's `position`, `runs_at_once`, `chat_project_id`, and `running`, the play runs going on it now. Your defaults and a project's own computers are set on the Computers settings page.
- `computer_project_create` adds a new T3 project to one of your computers, for when `computer_list` shows none holding the work. Give `repository` (`owner/name`) to clone a GitHub repository, or `name` to make an empty folder, which is also how a computer with no T3 project at all gets the one setup needs. It lands in the folder most of the computer's T3 projects share, or `~/Code` when it has none, and the computer's Nexul app clones with a read-only token for that one repository, so the computer needs no GitHub sign-in (an older app falls back to the computer's own git credentials). It returns the project's id, the `t3_project_id` `play_run` takes, once the clone is done; a failed clone leaves no project behind and says why, such as the computer having no GitHub sign-in. A computer paired before cloning existed refuses with a message to pair it again, which `computer_pair` or **Pair again** on the Computers settings page does.

The server also offers resources (`docs://{id}`, `tickets://{id}`, and `topology://{id}` for a workspace's canvas) and four prompts: `create_ticket_from_doc`, `deploy_and_watch_stack`, `investigate_failure`, and `ship_repository`, which runs the project wizard's steps end to end.

Work an agent starts is recorded as yours, marked as started over MCP. Nexul hides any `dep_` token as `[redacted token]` before it saves a play's trail or an agent's reply.

## Protocol

The server runs on the official MCP Go SDK over Streamable HTTP, statelessly: each `POST /mcp` gets one JSON response, with no sessions or open streams. A browser request is accepted only from your instance's own URL.
