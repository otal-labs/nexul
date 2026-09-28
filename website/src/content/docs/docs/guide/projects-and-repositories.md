---
title: Projects and Repositories
description: How projects organize work, how repositories attach to them, and how members and roles control access.
sidebar:
  order: 9
---

A **project** is the organizational grouping inside a workspace. Everything you build — tickets, docs, stacks — belongs to exactly one project. A workspace can hold as many projects as you want: split by technical layer (a frontend project and a backend project), or by whole domain (a Docs project, an Integrations project).

Each project gets its own board, its own docs list, and its own settings. Nothing here is shared across projects except the workspace they live in.

## Creating a project

A project only ever comes from the project wizard. A new workspace has none: the owner lands in the wizard right after the owner wizard, and until a project exists the sidebar, the board, and the docs list each show a **New project** action instead. Afterwards, open the sidebar and pick **New project**. Either way it starts the project wizard, which walks you straight from an empty project to a deployed service:

1. **Project** — name and prefix (2-5 letters, used to render ticket ids like `BE-42`).
2. **Repository** — say where the project's tests live, in the repository you deploy or in a separate one (see [A tests repository](#a-tests-repository)), then pick one of your installed GitHub repositories to deploy (see [GitHub App](/docs/guide/github-app/)).
3. **Service** — the wizard scans the repository for a Dockerfile or compose file and proposes a candidate; pick the machine to deploy it on (see [Runners](/docs/guide/runners/)).
4. **Env** — only shown if the repository has a `.env.example`; fill in the values it lists.
5. **Reach** — optionally expose the service at a hostname.
6. **Deploy branches** — optionally deploy other branches, like `feature/*` or `staging`, as their own copies (see [Stacks and Deploys](/docs/guide/stacks-and-deploys/#branch-deploys)).
7. **Done** — offers the project's interview while it has none, then lets you choose **View on the canvas** or **View stack**. Skipping the interview asks "are you sure?" first, and a banner stays on the project's board until the interview exists (see [The Interview play](/docs/guide/plays/#the-interview-play)).

The wizard reopens later from a project's **New service** button in **Services**, or from the Topology page's empty state, to attach another repository or add another service to an existing project. **Repositories** has a separate **Add repo** dialog for associating an existing repository directly.

## Four ways to connect a repository or adopt stacks

The repository step isn't the only way to connect a repository, and machine import is a separate path:

- **New project** — the wizard above, starting from nothing.
- **New service** — from an existing project's **Services** section or the Topology empty state, jumping straight to the repository step with the project already chosen.
- **Add repo** — from an existing project's **Repositories** section, associating a repository directly without opening the service wizard.
- **Import from this machine** — from the [Runners](/docs/guide/runners/) page, next to a machine. This reads what's already running there (containers, networks, reverse proxies) and adopts it as unmanaged stacks. An unmanaged stack has no repository attached yet; use **Attach repository** on it to link one and make it a managed, deployable stack.

For the first three doors, a repository ends up attached to exactly one project — the same repository can't be linked into two projects at once. Linking a repository that's already attached elsewhere is rejected outright instead of failing later during a deploy. Import creates unmanaged stacks without a repository; use **Attach repository** on an imported stack to give it a repository and make it managed and deployable.

## A tests repository

A project deploys from one repository. If its end-to-end or other tests live in a repository of their own, attach that one as the project's **tests repository**: the repository step asks where tests live and, for a separate repository, lets you pick it. A tests repository is never deployed. Nexul refuses to build a stack from it or run a deploy of it, so the project keeps one deployable repository.

The answer is stored on the project as its tests location (`same` or `separate`), and the interview starts from it. It shows in the project's **Repositories** list, where the tests repository carries a `tests` marker and can be removed like any other. Over MCP, `project_update` attaches one through `add_repos` with `role: "tests"` (which also records the location as `separate`) and records or withdraws the answer through `tests_location`, and `project_get` returns each repository's role.

## Members and roles

Every workspace auto-creates a singleton **Owner** role at creation. It can't be deleted or renamed, and it implicitly holds every permission — an Owner is never locked out by a permission change.

Beyond Owner, roles are fully custom: anyone holding `roles:write` can create as many roles as needed, each with its own set of permissions. Every permission follows one shared vocabulary, `<domain>:<action>` (`docs:write`, `tickets:read`, `members:delete`, and so on) — the same values gate a role, a personal access token, and an automation, so "what can this actor do" has one consistent answer everywhere.

To invite someone to a workspace:

1. Open **Members** (`/members`).
2. Create an invitation, choose one or more workspaces, and select a role for each. The Owner role isn't offered here — transferring ownership is a separate action.
3. Copy the generated link and send it through any channel you trust. Nexul shows it once.
4. The recipient opens the link, signs in through one of the instance's configured OAuth providers, reviews the access package, and accepts it.

The link grants both first admission to the private instance and the selected workspace memberships. It expires after one or seven days and stops working after one successful acceptance.
