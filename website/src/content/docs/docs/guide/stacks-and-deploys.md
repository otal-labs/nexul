---
title: Stacks and Deploys
description: Ship a repository as a stack, roll it back, read its logs, and deploy its branches as their own copies.
sidebar:
  order: 6
---

A stack is one repository's containers: a compose file, or a Dockerfile as a stack of one. You deploy, roll back and delete the stack as a whole.

## Creating a stack

The project wizard creates stacks. It picks the repository, finds its compose file or Dockerfile, asks for the machine, environment values, a hostname and branches, then deploys. See [Projects and repositories](/docs/guide/projects-and-repositories/#creating-a-project) for each step.

To adopt containers already running on a machine instead, see [Importing what's already running](/docs/guide/runners/#importing-whats-already-running).

## The stack page

Open a stack from its project's services, or from its node on **Topology**. The page has these sections:

| Section | What you do there |
| --- | --- |
| **Overview** | Deploy, roll back, and see each container's image, status, networks and ports |
| **Logs** | Read each container's output live |
| **Exposures** | Route hostnames to the stack's containers (see [Topology and DNS](/docs/guide/topology-and-dns/#exposures)) |
| **Branch deploys** | Deploy other branches as their own copies |
| **Deploy history** | Every build and deploy, with its log |
| **Danger zone** | Delete the stack |

The containers table is read-only. To change a container, change the compose file or Dockerfile and deploy again.

## Deploying

The **Deploy** card shows one way to ship, depending on the stack:

- **A stack with a repository**: type a branch, tag or commit in **Build & deploy ref** and click **Build & deploy**. The runner builds it on the stack's machine and deploys the result.
- **A single container with no repository**: **Redeploy** pulls the same image again and restarts the container.
- **A compose stack with no repository**: attach a repository first.

**Rollback** deploys the image of the last deploy that came up healthy. It stays disabled until there is one.

A deploy also starts on every push to a branch that matches one of the stack's branch deploy rules (see below).

## Logs

**Logs** shows what each container prints, one tab per service: the last lines, then new ones as they arrive. Switch between **All** and **Errors**, or **Pause** to stop following. Values from the stack's environment are masked before they reach your browser. Reading container logs takes the `stacks:logs` permission.

Nexul's own logs are elsewhere; see [Logs](/docs/guide/logs/).

## Branch deploys

A branch deploy rule maps a branch pattern to a Docker network. The pattern is an exact name like `main`, or a name ending in one wildcard like `feature/*`. Nexul checks every push against the stack's rules.

- An exact rule with no name suffix redeploys the stack itself. The rule for your default branch works this way.
- A wildcard rule deploys a separate copy for each matching branch, a preview deployment, and tears it down when the branch is deleted.
- An exact rule with a name suffix, like `staging`, deploys one separate copy.

To add a rule, open **Branch deploys** and fill in **Branch pattern**, **Docker network**, and optionally:

- **Hostname template**: `{branch}` becomes the branch as a hostname label. For a wildcard, only the part the wildcard matched counts, so `feature/dot.test` under `feature/*` with `{branch}.example.com` is served at `dot-test.example.com`. Dots, slashes and capitals become dashes and lower case, cut to 63 characters. A template needs a **Port**.
- **Overrides**: `KEY=value` lines that replace the stack's environment values for this branch only, such as a `DATABASE_URL` pointing at a test database. Remove an override and the next deploy of that branch goes back to the stack's value. Rules that redeploy the stack itself have nothing to override; edit the stack's environment instead.

**Live branch deployments** lists the copies running now. A copy has no rules of its own.

A copy on the same network as the default branch, with no overrides, shares production's services, its database included. Nexul never offers one as a place to test a ticket.

Agents set rules, overrides included, with the `stack_update` MCP tool.
