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

1. **Project** — name and prefix (2-5 letters or digits, starting with a letter, used to render ticket keys like `BE-42` or `P1-12`). A prefix is unique within its workspace, so two workspaces may both have a `BE` project and a key such as `BE-42` names a ticket only together with its workspace.
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

## Renaming a workspace

A workspace has a name, shown in the sidebar and the switcher, and a **slug**, its name in every link: `/<slug>/board`, `/<slug>/tickets/WEB-12`. Anyone holding `workspaces:write` in the workspace edits both under **Configuration → General**, the first section. Saving a new name changes nothing else. Saving a new URL moves every link into the workspace: links using the old address stop working and are not redirected, so the page says so under the field before you save. A slug is lowercase letters and digits joined by single dashes, at most 48 characters, not a path the app itself owns (such as `settings`), and not taken by another workspace; the form checks the first three as you type, and the server's answer to the last shows under the field. Everyone with the workspace open is moved onto the new address without a reload. Over MCP, `workspace_update` changes the same two fields.

## Team and roles

Every workspace auto-creates a singleton **Owner** role at creation. It can't be deleted or renamed, and it implicitly holds every permission — an Owner is never locked out by a permission change.

Beyond Owner, roles are fully custom: anyone holding `roles:write` can create as many roles as needed, each with its own set of permissions. Every permission follows one shared vocabulary, `<domain>:<action>` (`docs:write`, `tickets:read`, `members:delete`, and so on) — the same values gate a role, a personal access token, and an automation, so "what can this actor do" has one consistent answer everywhere. A role, a role assignment, an override, or an invitation can only hand out permissions its giver holds in that workspace; the Owner holds all of them. Besides read, write, and delete, a few domains carry an extra toggle on their row in the role editor: Docs has Thread, Clone, and Lock (`docs:lock`, locking and unlocking a doc, granted apart from editing it).

Some permissions are about the whole instance rather than one workspace, and holding one in any workspace you belong to is enough, unless that membership is limited to chosen projects, which holds none of them: `instance:read` and `instance:write` for the instance's settings, sign-in providers, and upgrades; `accounts:read`, `accounts:write`, and `accounts:delete` for accounts; `workspaces:create`; `runners:write` and `runners:delete`; `automations:write` and `automations:delete` for automations hosts; `connectors:write` for a connector's app; `integrations:*` and `audit:read`. There is no separate administrator: whoever owns a workspace holds them all, and anyone else holds what their roles grant. `workspaces:create` is as strong as Owner, since the creator of a workspace becomes its Owner.

Holders of `accounts:read` see everyone in one place: **Settings → Team** (`/settings/team`), under Instance settings. It lists every registered account with its status (active, disabled, or removed) and a one-line summary of where it has access. Open a person to see one tab per workspace they're in. A tab holds their role, their workspace-wide permission overrides, their project access, and **Remove from workspace**; the **+** after the last tab adds them to a workspace they're not in, with a role. Nothing applies as you edit: a tab with changes waiting shows a dot, **Confirm** applies them all (additions first, then role and access changes, then removals), and **Cancel** or closing the dialog asks before discarding them. If one change is refused, the dialog stays open, says which one, and keeps whatever hasn't applied for the next Confirm. Each change needs `members:write` in that workspace, so seeing everyone alone doesn't let you change a workspace you don't manage; its tab stays read-only and says why. Disabling, reactivating, and restoring the account itself also live there for holders of `accounts:write`, and removing it for holders of `accounts:delete`; those apply at once, each after its own confirmation. Someone who manages members in a workspace without `accounts:read` finds Team under **Configuration** instead (`/<workspace>/configuration/team`), showing only the workspaces they manage and the people in them. Roles are defined per workspace under **Roles**; Team only assigns them.

Under the role on each workspace, a person's **Every project** row decides which projects they see. *From role*, every member's setting unless changed, gives them the role's project levels on every project, new ones included. *Chosen projects* makes them see only the projects given in the list under it, laid out like a role's domains: one row per project you can open, with the same None, Read, Write, and Delete buttons setting one level for the whole project, and **Areas** opening that project's areas (tickets, docs, memories, and so on) underneath, each with its own level. None takes the project away. While the row reads From role the project rows stay greyed out. Anything not given stays hidden, its name included, a direct link to it reads as not found, and projects made later stay hidden too. Switching back to From role keeps the levels chosen, so switching again brings them back. Setting it needs `members:write` in the workspace and only hands out levels you hold on that project yourself; the Owner always sees every project. A change reaches the person's open browser at once: a project taken away drops out of their sidebar, and a page of it they have open says they no longer have access. The same block sits under each workspace in the invitation dialog, so someone invited onto chosen projects never sees the rest. Over MCP, `account_update` sets it with `every_project` and `project_access`.

A project's **Settings → General** lists, for holders of `members:write`, the people on Chosen projects who can open it, each with a summary of their levels, under **People with access**; access is changed from Team. Deleting a project names the people on Chosen projects who lose it in the confirmation. A ticket's developer and tester pickers offer only people who may open its project.

Seeing who you work with needs no permission. Every member of a workspace can read its people, each with their login, display name, and picture, and nothing about their role or access, through `GET /api/workspaces/{workspaceID}/people`; that is what names chat authors, DMs, and a ticket's developer for someone without `members:write`. A person's display name is the one they set under **Your settings → Profile**, else their sign-in account's name, else their login, and a changed name or picture reaches everyone's open screens without a refresh.

To invite someone new:

1. Open **Settings → Team** (`/settings/team`) and choose **Invite**.
2. Choose one or more workspaces, and select a role for each. The Owner role isn't offered here — transferring ownership is a separate action. Set **Every project** to *Chosen projects* to hold them to the projects you pick under it.
3. Copy the generated link and send it through any channel you trust. Nexul shows it once.
4. The recipient opens the link, signs in through one of the instance's configured OAuth providers, reviews the access package, and accepts it.

The link grants both first admission to the private instance and the selected workspace memberships. It expires after one or seven days and stops working after one successful acceptance.
