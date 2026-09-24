---
title: Stacks and Deploys
description: Turn a repository into a running stack, and the ways it can ship its next version.
sidebar:
  order: 6
---

A **stack** is one repository's worth of deployable containers — a compose file, or a single Dockerfile counted as a stack of one. The stack is what gets deployed, rolled back, and torn down; the containers inside it are observed, not deployed on their own.

## Three ways to create a stack

**New project** — the project wizard at `/wizard/project/project` walks you through everything: pick or name a project, pick a repository, name the service, optionally fill in environment variables, and optionally give it a hostname.

**Add a service to an existing project** — the same wizard, entered from a project page with the project already chosen, starting at the repository step.

**Import from a machine** — adopt containers already running on a server instead of deploying from scratch. See [Runners](/docs/guide/runners/) for how discovery and import work.

### The new-stack wizard, step by step

1. **Project** — the stack's project.
2. **Repository** — pick from your installed repositories; Nexul scans it for a compose file or a Dockerfile. If it finds several candidates (a monorepo), you choose one. If it finds nothing, you can point the wizard at a Dockerfile path yourself. If the GitHub App isn't installed on the repository, the wizard links you straight to installing it.
3. **Service** — name the stack and pick the [machine](/docs/guide/runners/) it deploys to. A compose file becomes a `compose` stack; a Dockerfile becomes a `run` stack.
4. **Environment** — only shown when the scan found `.env.example` keys. Values are optional; the first deploy waits until this step is done, since a compose file's `env_file: .env` or `${VAR}` needs the values in place first.
5. **Reach** — optional. Give the service a hostname now (see [Topology and DNS](/docs/guide/topology-and-dns/)), or skip it and do it later from the stack page.
6. **Deploy branches** — optional. The first row is the repository's default branch, redeployed in place on every push at the service's hostname. Add rows like `feature/*` or `staging`: each deploys its own copy, shows the URL an example branch would get (`feature/security-test` → `security-test.example.com`), and picks a network from the machine's networks, listed with what runs on each. Under **Advanced options**, a row's overrides replace the default branch's environment values for that branch. A row on the default branch's network with no overrides uses production's services, including its database, and is never offered to testers. The rows save as the stack's branch deploy rules.
7. **Done** — a link to the stack's page, and a link to see it on the topology canvas.

## The stack page

Each stack has its own page (`/stacks/<id>`), with a header showing its slug, repository, latest deploy, and hostnames, and a section nav:

| Section | What it shows |
| --- | --- |
| Overview | Deploy actions and the containers table |
| Exposures | Hostnames routed to this stack's containers |
| Branch deploys | Rules that turn a push into its own deployment (base stacks only) |
| Deploy history | Every build and deploy, oldest to newest |
| Danger zone | Delete the stack |

### Deploying: one action per stack shape

The Deploy card offers exactly one way to ship a new version, depending on what the stack is:

- **A stack with a repository attached** — a **Build & deploy** form: type a ref (branch, tag, or commit), and Nexul builds it on the runner's machine and deploys the result.
- **A run stack with no repository** (an image running standalone) — a **Redeploy** row: pulls the same image again and restarts the container.
- **A compose stack with no repository, or nothing built yet** — nothing to trigger yet; attach a repository first.

**Rollback** re-deploys the image from the last deploy that reported healthy, and stays disabled until one exists.

### Containers

The containers table lists what the stack declares — one row per compose service, or one row for a run stack's single container — with the image, status, docker networks (and the address the runner reported on each), and ports. It's read-only: the compose file or Dockerfile is the only source of truth for a container.

### Branch deploys

A **branch deploy rule** maps a branch pattern — an exact name like `main`, or a single trailing wildcard like `feature/*` — to a docker network, and optionally a hostname template and a name suffix for the deployed clone. Every push is checked against the stack's rules; there's no separate "environment" concept, an exact rule for `main` and a wildcard rule for `feature/*` are just two rules on the same stack.

- An **exact** rule with no name suffix redeploys the base stack itself in place.
- A **wildcard** rule spawns a **preview deployment**: one clone per matching branch, on its own network and (if a hostname template is set) its own hostname, torn down automatically when the branch is deleted.

In a hostname template, `{branch}` becomes the branch as a hostname label: for a wildcard rule, only the part the wildcard matched, so `feature/dot.test` under `feature/*` with `{branch}.example.com` is served at `dot-test.example.com`. Dots, slashes, and capitals become dashes and lowercase, trimmed to 63 characters.

A rule that deploys its own copy (a wildcard, or an exact name with a name suffix) can carry **overrides**: `KEY=value` lines that replace the base stack's environment values for that branch only, like a `DATABASE_URL` pointing at a QA database. The copy runs with the overrides while the base keeps its own values. Remove an override and the next deploy of that branch goes back to the base value. An in-place rule deploys the base stack itself, so it has nothing to override; edit the base stack's environment instead. Edit overrides from a rule's **Overrides** button, or pass `overrides` on a rule in the `stack_update` MCP tool. Override values are stored and handled like the stack's own environment values.

A stack that is itself a branch deployment (`derived_from` is set) has no rules of its own — it inherits from whatever created it.

## Next steps

Give a stack a hostname from its Exposures section — see [Topology and DNS](/docs/guide/topology-and-dns/). To scale where a stack runs, add more runners to its machine — see [Runners](/docs/guide/runners/).
