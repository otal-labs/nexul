---
title: Projects and Repositories
description: How projects organize work, how repositories attach to them, and how members and roles control access.
sidebar:
  order: 9
---

A **project** is the organizational grouping inside a workspace. Everything you build — tickets, docs, stacks — belongs to exactly one project. A workspace can hold as many projects as you want: split by technical layer (a frontend project and a backend project), or by whole domain (a Docs project, an Integrations project).

Each project gets its own board, its own docs list, and its own settings. Nothing here is shared across projects except the workspace they live in.

## Creating a project

Open the sidebar and pick **New project**. This starts the project wizard, which walks you straight from an empty project to a deployed service:

1. **Project** — name and prefix (2-5 letters, used to render ticket ids like `BE-42`).
2. **Repository** — pick one of your installed GitHub repositories (see [GitHub App](/docs/guide/github-app/)).
3. **Service** — the wizard scans the repository for a Dockerfile or compose file and proposes a candidate; pick the machine to deploy it on (see [Runners](/docs/guide/runners/)).
4. **Env** — only shown if the repository has a `.env.example`; fill in the values it lists.
5. **Reach** — optionally expose the service at a hostname.
6. **Done** — lands on the topology canvas showing what was just deployed (see [Stacks and Deploys](/docs/guide/stacks-and-deploys/)).

The same wizard reopens later from a project's **Add service** button, or from the Topology page's empty state, to attach another repository or add another service to an existing project.

## Three ways to attach a repository

The repository step isn't the only door in:

- **New project** — the wizard above, starting from nothing.
- **Add service** — from an existing project's settings (**Repositories** or **Services** section) or the Topology empty state, jumping straight to the repository step with the project already chosen.
- **Import from this machine** — from the [Runners](/docs/guide/runners/) page, next to a machine. This reads what's already running there (containers, networks, reverse proxies) and adopts it as unmanaged stacks. An unmanaged stack has no repository attached yet; use **Attach repository** on it to link one and make it a managed, deployable stack.

Whichever door you use, a repository always ends up attached to exactly one project — the same repository can't be linked into two projects at once. Linking a repository that's already attached elsewhere is rejected outright instead of failing later during a deploy.

## Members and roles

Every workspace auto-creates a singleton **Owner** role at creation. It can't be deleted or renamed, and it implicitly holds every permission — an Owner is never locked out by a permission change.

Beyond Owner, roles are fully custom: anyone holding `roles:write` can create as many roles as needed, each with its own set of permissions. Every permission follows one shared vocabulary, `<domain>:<action>` (`docs:write`, `tickets:read`, `members:delete`, and so on) — the same values gate a role, a personal access token, and an automation, so "what can this actor do" has one consistent answer everywhere.

To invite someone to a workspace:

1. Open **Members** (`/members`).
2. Add their GitHub login and pick a role from the workspace's own role catalog. The Owner role itself isn't offered here — transferring ownership is a separate action.
3. If the login has never signed in before, the invite waits as pending; it resolves into real membership the moment that person first signs in.

A login has to already be allowed to sign in to the instance before it can be invited into a workspace — inviting someone to a workspace never grants them sign-in access on its own; that's a separate, instance-level allowlist under **Settings**.
