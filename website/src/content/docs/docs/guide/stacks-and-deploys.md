---
title: Stacks and deploys
description: Ship a repository as a stack, set its environment, roll it back, read its logs, and deploy its branches as their own copies.
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
| **Environment** | Add, change and remove the stack's environment variables |
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

**Rollback** deploys the image of the last deploy that came up healthy. It shows only when that deploy has an image to go back to. A build leaves none, so a stack that builds from a repository has no Rollback; to go back, **Build & deploy** the earlier ref.

A deploy also starts on every push to a branch that matches one of the stack's branch deploy rules (see below).

## Environment

**Environment** lists the stack's variables, a name and a value each. Values show as dots until you click **Reveal values**. Edit a value, **Add variable**, or remove a row with the **×** beside it, then **Save**. Emptying a value keeps the variable with an empty value; removing the row removes the variable.

Saving doesn't redeploy: the running containers keep their old values, and the next deploy starts with the new ones. **Deploy now**, beside **Save**, ships it at once. It builds the stack's branch when a repository is attached, and redeploys the running image of a single container without one. It waits until your changes are saved and takes `deploys:write`.

- Changing the environment takes `stacks:write` on the stack. Someone who can read the stack but not change it sees the names and never the values: the server withholds them from every read, so no part of a page or an API answer carries them. The same goes for the overrides on a branch deploy rule.
- A copy deployed from a branch has no environment of its own. Its page lists the names, and its values come from the stack and the rule's overrides.
- Two people editing at once: the page shows the other person's save when it arrives, and saving replaces the whole environment with what your page holds.
- A change to a stack, its environment included, reaches live updates, webhooks and automations as the names only.
- Agents list a stack's variables by name with `stack_get` and change them with `stack_update`, which sets and unsets single variables so values they never saw are kept. A value never leaves the server over MCP, whatever the caller may do.

## Retrying a failed deploy

A failed deploy's page has **Retry**. It starts a new deploy of the same commit the failed one built, with the stack's settings as they are now. Change what made it fail, such as a missing environment variable on the stack, click **Retry**, and the build runs again on the same code with the new value.

- A deploy records the commit its runner checked out, shown beside its id. A deploy that failed before the checkout finished has no commit yet, so **Retry** builds the ref it was asked for. A single-container redeploy retries the same image.
- The failed deploy stays in **Deploy history**, and the new one is a separate entry. **Retry** takes the `deploys:write` permission on the stack, the same as a deploy, and a stack runs one deploy at a time, so it is refused while another is pending or running.
- Agents retry with `stack_deploy` and `retry` set to the failed deploy's id, and read each deploy's `ref` and `commit` with `deploy_get`.

## Who can see and deploy a stack

A stack follows its project: whoever is in the project gets what their roles allow on its stacks. To narrow or widen that for one stack, such as letting only one role deploy production, right-click the stack in its project's services, or open **⋯** on the stack page, and pick **Permissions…**. It shows only to someone who holds `stacks:write` on the stack.

The panel sets the stacks and deploys permissions (**Read**, **Write**, **Delete** and **Read container logs** on the stack, **Read** and **Write** on its deploys) for a role or a person, each as **Allow**, **Fallback** or **Deny**; see [Permissions on a project, a folder, a doc or a play](/docs/guide/people-and-access/#permissions-on-a-project-a-folder-a-doc-or-a-play) for how the three work. A stack's rules sit between its project's and the stack: they beat the project's rules, and between a person's roles Allow wins. They reach the stack's deploys and its branch deployments, which have no rules of their own.

Deny **Read** on a stack for a role and the stack drops out of that role's stack lists, deploy lists, Topology's stack list and live updates, deploys included, and its link answers not found. A refused deploy names the rule that refused it, such as `deploys:write required (Denied on stack production for role Developer)`. Deleting a stack deletes its rules. The instance's own stacks, a gateway's, carry no rules of their own: seeing one takes **See** on its machine and reading DNS, and editing or deploying one takes **Deploy to** on its machine and writing DNS.

Agents set the same rules with `permission_overwrite_update` and `resource_type: stack`, and read them with `permission_overwrite_list`.

## Which machines a stack can use

A machine is a server's Docker, so putting a stack on one takes **Deploy to** on that machine, beside the stack permissions above. It is checked when you create a stack, move one to another machine, import containers from a machine, or set up a gateway, and again on every deploy you start. A stack page shows its deploy actions (**Redeploy**, **Rollback**, **Build & deploy**, **Deploy now**, **Retry** and **Cancel deploy**) only when you hold Deploy to on its machine; with **Show locked actions** on for one of your roles they stay in view, dimmed, with the reason `No Deploy to on machine <name>`. Machine pickers list only the machines you may deploy to, and a machine you can't see is answered like a name no machine bears, even when you type its name into a tool.

Taking someone off a machine stops their next deploy there. Deploys Nexul starts itself, after a push to a branch with a deploy rule or a merge, keep running. A stack on a machine you can't see still shows that machine's name on its page, as plain text with no link.

## Logs

**Logs** shows what each container prints, one tab per service: the last lines, then new ones as they arrive. Switch between **All** and **Errors**, or **Pause** to stop following. Values from the stack's environment are masked before they reach your browser. Reading container logs takes the `stacks:logs` permission.

When the stack's runner is offline, the view says so and keeps retrying; **Details** under the sentence shows the server's own message.

Nexul's own logs are elsewhere; see [Logs](/docs/guide/logs/).

## Branch deploys

A branch deploy rule maps a branch pattern to a Docker network. The pattern is an exact name like `main`, or a name ending in one wildcard like `feature/*`. Nexul checks every push against the stack's rules.

- An exact rule with no name suffix redeploys the stack itself. The rule for your default branch works this way.
- A wildcard rule deploys a separate copy for each matching branch, a preview deployment, and tears it down when the branch is deleted.
- An exact rule with a name suffix, like `staging`, deploys one separate copy.

To add a rule, open **Branch deploys** and fill in **Branch pattern**, **Docker network**, and optionally:

- **Hostname template**: `{branch}` becomes the branch as a hostname label. For a wildcard, only the part the wildcard matched counts, so `feature/dot.test` under `feature/*` with `{branch}.example.com` is served at `dot-test.example.com`. Dots, slashes and capitals become dashes and lower case, cut to 63 characters. A template needs a **Port**.
- **Overrides**: `KEY=value` lines that replace the stack's environment values for this branch only, such as a `DATABASE_URL` pointing at a test database. Remove an override and the next deploy of that branch goes back to the stack's value. Rules that redeploy the stack itself have nothing to override; edit the stack's environment instead.

**Live branch deployments** lists the copies running now. A copy has no branch deploy rules of its own, and its permissions are its stack's.

A copy on the same network as the default branch, with no overrides, shares production's services, its database included. Nexul never offers one as a place to test a ticket.

Agents set rules, overrides included, with the `stack_update` MCP tool.
