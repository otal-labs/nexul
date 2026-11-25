---
title: MCP server
description: Connect Claude Code, Codex, opencode, or any MCP client to your instance so an agent can work in Nexul with your permissions.
sidebar:
  order: 11
---

Every Nexul instance has an MCP server at `/mcp`. Connect your agent tool to it and the agent can read docs, file tickets, post in chat, deploy stacks, and everything else in the table below, acting as you, with your permissions and nothing more.

A [paired computer](/docs/guide/paired-computers/) needs none of this: its [setup](/docs/guide/computer-setup/) connects every provider for you, with the computer's own token. Follow these steps for any other tool or machine.

## Connect

Paste your instance's MCP URL, `https://nexul.example.com/mcp`, into your tool and sign in. Claude's custom connectors, the MCP Inspector, and any client that supports MCP sign-in open a Nexul page in your browser: sign in the way you usually do, check the app's name and where it sends you back to, and click **Allow**. The tool then connects as you.

Each app you allow is listed under **Settings → Security**, **Personal access tokens**, marked as a signed-in app, with when it was last used. Revoke it there and the app's next call gets `401`. Its access token lasts an hour and the app renews it by itself until you revoke it, or until it goes 30 days without renewing; then sign it in again. If a sign-in code or an old refresh token is ever sent twice, which happens when one was copied, Nexul revokes the app's sign-in and it has to sign in again. So that revoking it always cuts it off, a signed-in app can't create or read credentials: tokens, computers' install commands, automations' tokens, invitation links, or bots' webhook URLs. Use a personal access token for those.

### With a token instead

For a tool without MCP sign-in, or a script:

1. Open **Settings → Security** and create a token under **Personal access tokens**. Copy it now; the raw token (`dep_…`) is shown once.
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

### Behind a proxy or Cloudflare Access

MCP sign-in needs these paths to reach the instance without a login in front of them; they check their own credentials:

- `/mcp`
- `/.well-known/oauth-protected-resource` and `/.well-known/oauth-protected-resource/mcp`
- `/.well-known/oauth-authorization-server`
- `/oauth/register` and `/oauth/token`

With Cloudflare Access, add a Bypass policy for them. A proxy must pass the `Authorization` header and the request body through unchanged. The sign-in page itself, `/oauth/authorize`, is part of the web app and can stay behind Access.

Each address may register 60 apps an hour, and an app that registers but is never approved is dropped after a day. Behind a proxy or tunnel every request arrives from the proxy's address, so set `NEXUL_TRUSTED_PROXIES` in the server's environment to the proxy's IPs or CIDR ranges, comma-separated (for example `127.0.0.1,172.17.0.0/16` for a tunnel running on the same machine or in Docker). Nexul then reads the client's address from the `X-Forwarded-For` header those peers send, and ignores it from anyone else.

## What an agent can do

114 tools, each named `<object>_<verb>` and shaped around a task rather than a button.

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
| Imports | `import_run` |
| Calendar | `meeting_list`, `meeting_create`, `meeting_update` |
| Stacks and deploys | `stack_list`, `stack_get`, `stack_create`, `stack_update`, `stack_delete`, `stack_deploy`, `deploy_list`, `deploy_cancel` |
| Machines and the instance | `machine_list`, `machine_discover`, `machine_import`, `host_create`, `host_delete`, `instance_get`, `instance_update`, `instance_upgrade` |
| DNS and routing | `dns_zone_list`, `dns_record_list`, `dns_record_create`, `dns_record_update`, `dns_record_delete`, `dns_tunnel_list`, `dns_tunnel_create`, `dns_tunnel_update`, `dns_tunnel_delete`, `gateway_list`, `gateway_create`, `gateway_delete`, `exposure_list`, `exposure_create`, `exposure_delete` |
| Topology | `topology_get`, `topology_update` |
| Repositories and pull requests | `repository_list`, `repository_scan`, `pull_request_list`, `pull_request_get` |
| Plays | `play_list`, `play_create`, `play_update`, `play_delete`, `play_run`, `trail_list`, `trail_update` |
| Automations | `automation_list`, `automation_create`, `automation_update`, `automation_delete`, `automation_token_create` |
| Paired computers and skills | `computer_list`, `command_run`, `computer_create`, `computer_pair`, `computer_delete`, `computer_project_create`, `computer_setup_run`, `computer_setup_update`, `computer_mcp_token_create`, `computer_mcp_token_delete`, `skill_get` |
| Accounts and access | `account_get`, `account_list`, `account_update`, `account_delete`, `invitation_create`, `invitation_list`, `invitation_delete`, `permission_overwrite_list`, `permission_overwrite_update`, `role_update`, `role_delete` |
| Failed events | `dead_letter_list`, `dead_letter_replay` |

A few things that save an agent a wasted call:

- **Updates are patches.** Send only the fields to change; the rest keep their values.
- **Edit a body without resending it.** `doc_update`, `ticket_update`, `play_update`, and `memory_update` take `edits`, a list of `find` and `replace` pairs applied in order to the body (a ticket's description, a play's instructions) as `doc_get`, `ticket_get`, `play_list`, and `memory_get` return it. Each `find` must match exactly once; if any fails, nothing is saved. `edits` does not go with the whole body field or `revert_to_version`, and the result is one saved version.
- **Lists page.** They take `limit` (50 by default, 100 at most) and `offset` and return `items`, `total`, `has_more`, and `next_offset`. `total` counts everything you can see, searches included, so paging with `next_offset` reaches every match. A row added ahead of your place while you page shifts the later pages by one.
- **A list given an id returns that one record.** `deploy_list` with a deploy's `id` returns it with its log; `trail_list`, `automation_list`, `computer_list`, and `dns_tunnel_list` work the same way. A field's description lists the values it takes.
- **Tickets by key.** Ticket tools take an id or a key such as `WEB-12`. A key is unique only within a workspace, so when you hold it in two, pass `workspace` too, or the call is refused with the workspaces to choose from.
- **`me` is you.** A ticket's `developer` and `tester` on `ticket_create` and `ticket_update`, and the `developer` filter on `ticket_list`, take `me` for the person the connection acts as, so an agent need not look its own login up first. An automation token is no member, so it cannot use `me`.
- **Change many tickets in one call.** `ticket_update` takes `ids`, up to 100 ids or keys, and applies the same fields to each, such as a `developer` or a `category_id`. Every ticket is found before any changes, so a mistyped key changes nothing; the tickets then change in order, and the call stops at the first one that fails and names the ones already changed. `title`, `body`, `edits`, `position`, and `revert_to_version` change one ticket at a time. `ticket_create` takes `tickets`, up to 100 titles and bodies that share every other field, such as a doc split into tickets of one type and category; it creates them in order, stops at the first that fails, and returns their keys. `ticket_list` finds the tickets to pass, filtered by `category_id` (an empty string for uncategorized), `status_id`, or `developer` (a login, or an empty string for unassigned).
- **Errors say how to fix the call.** A failed tool returns the fields it needs, not a bare error. A refused one names the role or rule that refused it; `permission_overwrite_list` with a `user_id` explains every answer on a doc, folder, stack, play, project, or instance area.
- **Read-only and destructive hints.** Reads are marked read-only, and deletes and anything that starts work are marked destructive, so your client knows what to confirm.
- **Not found can mean no access.** Someone who sees only some projects gets not found for the rest, and so does their agent.
- **Changes are audited.** Every call to a tool that changes something is recorded in the [audit log](/docs/guide/api-and-tokens/#integrations-and-outgoing-webhooks) under your name and kept for 45 days. Reads are not.

Some tools worth knowing:

- `pull_request_get` answers "why does this code exist". Give it a repository and a pull request number, or a commit SHA from `git blame`, and it returns the pull request, its tickets, each ticket's doc, bugs found after done, and the decisions-log entries citing them.
- `attachment_create` puts a file on a doc, ticket, conversation, or memory, up to 10 MiB, and returns a markdown line to embed it. `attachment_get` reads one back; an image comes back as an image.
- `ticket_get` includes the ticket's test target, and `ticket_test_report` passes or fails it like the **Pass** and **Fail** buttons.
- `instance_get` with a `logs` object also reads the instance's own [logs](/docs/guide/logs/), newest first. It needs `logs:read` besides `instance:read`.
- `instance_get` also returns the instance's [enterprise license](/docs/guide/enterprise-license/), and `instance_update` with `license_key` pastes or removes the key. With the [single sign-on](/docs/guide/single-sign-on/) settings, `instance_get` also returns them and `instance_update` takes an `sso` object. With `branding`, `instance_update` sets the instance's own [look](/docs/guide/branding/).
- `play_run` starts a [play](/docs/guide/plays/) on your paired computer.
- `computer_create` adds one of your computers and returns the command that installs its runner: `commands.unix` for Linux and macOS, `curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token>`, and `commands.windows` for Windows, `& ([scriptblock]::Create((irm https://nexul.io/computer.ps1))) <token>`, where the token is good once, for an hour, and only on your instance. Run it on that computer from your own account: on Linux and macOS `sudo` only places the `nexul-computer` system service (a LaunchDaemon on a Mac), which runs the runner as you, never as root; on Windows it needs no administrator and installs a task of yours that runs the runner while you're logged on. Either installs T3 Code when it's missing ([what it installs](/docs/guide/paired-computers/#what-the-command-installs)). Once the runner connects, `computer_list` shows `runner.connected` on the computer; a computer added without a name takes its hostname. Pass `id` instead for a fresh command when the last one expired, or to [move a computer paired through a tunnel or by URL](/docs/guide/paired-computers/#move-a-computer-to-the-nexul-app) to the Nexul app: once it pairs through its runner it keeps its links, setup, MCP token and trails, and only then are its tunnel, DNS record and Access app deleted; `sudo cloudflared service uninstall` on the computer removes `cloudflared` afterwards. A failed move leaves the computer working as it was. Only you can see or use your computers; anyone else, a workspace Owner included, gets not found.
- Once its runner connects and T3 Code answers there, the computer pairs on its own, and pairs again before its 30-day session ends. `computer_pair` with only the computer's `id` pairs it again now, and moves a computer whose runner `computer_create` with its `id` installed. A failed pairing leaves the computer as it was, and `computer_list` says why in `pair_error`. `computer_list` also returns a computer's `facts` once its runner reports (its system, git identity, whether it can sign in to GitHub, free disk, Docker, T3 Code's state and port, and the providers and projects T3 Code lists) with `facts_at`, when they last changed; only you get them. With one computer's `id`, a computer whose facts show Docker available also lists its `containers`, and `container_logs: {container, tail}` adds that container's last lines (200 by default, at most 1000), read live through the computer's runner. `runner_logs: {lines}` adds the last lines the computer's Nexul app itself logged, the same way. A person the computer is shared with by `computers:read_logs` gets both reads and the container list too, with each container's own environment values hidden.
- `computer_pair` with `position`, `runs_at_once` or `chat_project_id` changes the computer's settings instead of pairing it: its place in [your order](/docs/guide/paired-computers/#your-computers-in-order) (0 for first), how many play runs it takes at a time (1 to 5), or the T3 project chats outside any project run in. `computer_list` returns each computer's `position`, `runs_at_once`, `chat_project_id`, and `running`, the play runs going on it now. Your defaults and a project's own computers are set on the Computers settings page.
- `command_run` runs one command on one of your computers through its Nexul app and waits for it: `computer_id`, `command`, and optionally `shell` (`bash`, `sh`, `pwsh` or `powershell`), `cwd` and `timeout_seconds` (60 by default, at most 300). It returns `status` (`succeeded`, `failed`, `timed_out`, `cancelled`, or `refused` when the computer turned commands off), `exit_code` and the last 64 KB of output, with its `job_id`. `computer_list` with the computer's `id` and `commands: {}` returns its newest 10 without output, and `commands: {job_id}` one with its output. `computer_pair` with `shell_enabled` sets the computer's [switch](/docs/guide/paired-computers/#commands-on-the-computer); false also stops the commands running there.
- `computer_project_create` adds a new T3 project to one of your computers, for when `computer_list` shows none holding the work. Give `repository` (`owner/name`) to clone a GitHub repository, or `name` to make an empty folder (setup makes one called `nexul-chat` by itself on a computer with no T3 project at all). It lands in the folder most of the computer's T3 projects share, or `~/Code` when it has none, and the computer's Nexul app clones with a read-only token for that one repository, so the computer needs no GitHub sign-in (an older app falls back to the computer's own git credentials). It returns the project's id, the `t3_project_id` `play_run` takes, once the clone is done; a failed clone leaves no project behind and says why, such as the computer having no GitHub sign-in. A computer paired before cloning existed refuses with a message to pair it again, which `computer_pair` or **Pair again** on the Computers settings page does.

The server also offers resources (`docs://{id}`, `tickets://{id}`, and `topology://{id}` for a workspace's canvas) and four prompts: `create_ticket_from_doc`, `deploy_and_watch_stack`, `investigate_failure`, and `ship_repository`, which runs the project wizard's steps end to end.

Work an agent starts is recorded as yours, marked as started over MCP. Nexul hides any `dep_` token as `[redacted token]` before it saves a play's trail or an agent's reply.

## Protocol

The server runs on the official MCP Go SDK over Streamable HTTP, statelessly: each `POST /mcp` gets one JSON response, with no sessions or open streams. A browser request is accepted only from your instance's own URL.
