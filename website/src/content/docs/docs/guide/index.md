---
title: Using Nexul
description: Go from a fresh server to a deployed stack, then find your way around everything else.
sidebar:
  label: Overview
  order: 0
---

Nexul runs on your own server and keeps your projects, docs, tickets, chat and deploys in one place. You use it in the browser, agents use it through its MCP server, and scripts use its HTTP API.

The first time, read these in order:

1. [Install](/docs/guide/install/): one command on a Linux server, a Mac or a Windows PC.
2. [Setup wizard](/docs/guide/setup-wizard/): give the instance its domain and sign in as the owner.
3. [GitHub App](/docs/guide/github-app/): create the App Nexul signs people in and reads repositories with.
4. [Runners](/docs/guide/runners/): add the services that build and deploy.
5. [Stacks and deploys](/docs/guide/stacks-and-deploys/): ship a repository and its branches.
6. [Topology and DNS](/docs/guide/topology-and-dns/): put your services on the internet.
7. [Logs](/docs/guide/logs/): read the instance's own logs.

Keep [Upgrade](/docs/guide/upgrade/) for later. The rest of the guide covers projects, docs and tickets, chat, agents and automations. [API and tokens](/docs/guide/api-and-tokens/) and the [desktop app](/docs/guide/desktop-app/) are for when you connect something else to the instance.

## Getting around

Press ⌘K on a Mac, or Ctrl+K anywhere else, to open the command palette; the Search row at the top of the sidebar opens it too. Type to jump to any page, board, settings section or tab (Tokens, Team or Roles, say), search the workspace for tickets, docs, pull requests, stacks, automations, channels and memories you can open, start a new ticket, doc, channel or direct message, switch workspace, or change the theme. Results come as one list: up to three pages or actions whose name starts with what you typed, then what the search found, kind by kind and best match first (a name equal to what you typed, then one holding each word, then the most recently updated), then the other pages and actions that match. Each result says what it is under its title: a ticket's key and status, a pull request's number, or Doc, Stack, Memory and so on. Tickets and docs that mention your words only in their text sit under **In text**, and a pull request opens the ticket it is linked to. When nothing matches, Enter starts a new ticket with what you typed as its title. Empty, it lists what changed lately in your current project, then every page, the boards and the create actions. Arrow keys move, Enter opens, Escape closes.

**Settings → Appearance** shows each mode and theme as a small picture of the app in its colours; picking one applies it at once, in this browser.
