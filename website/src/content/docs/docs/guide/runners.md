---
title: Runners
description: Add the services that build and deploy on your machines, scale them, and remove them.
sidebar:
  order: 5
---

A runner is a service on one of your machines. It connects out to the instance over a WebSocket and runs `docker` and `docker compose` against that machine's own Docker. The instance never connects in and holds no SSH keys.

`nexul install` already put one on your server, named `instance`. You can deploy with it straight away.

A computer someone adds for their own agent work runs a runner of its own too. It never builds or deploys, and it never shows on the **Runners** page or as a machine, for anyone: only the person who added it sees it, with their computers.

## Reading the page

The line under the title counts the fleet: runners online out of all of them, how many are busy, how many are offline, how many runner codes are still unused, and how many jobs wait. While a job waits, **Waiting for a runner** lists the queue in order, each job with its kind and why it waits on the machine it needs: **no runner online**, **all runners busy**, or **next in line**. A narrow window shows the first three, a wide one the first five; **Show more** lists the rest.

Below it, every machine you are on is one band in a single list, with its runners under it, one line each: the name, the version, and what the runner is doing (the job it runs, **idle**, or **offline** with when it was last seen). A name too long for its column keeps its end, and hovering it shows the whole name. A runner on an older release than the instance shows **↑** after its version. A code made for a machine and not used yet sits under that machine, and runners and codes that belong to no machine are listed last, under **No machine**.

A machine's band stays at the top of the page while you scroll through its runners. Its arrow folds the machine to its band and shows how many rows it hides; a machine whose runners are all offline starts folded.

## Who sees a machine

The **Runners** link shows in the sidebar, the command palette and the phone's More tab only for someone who sees a machine or can add one; the page itself opens for every member by its address and shows the machines you are on, with their runners and the jobs queued for them. A machine you aren't on isn't there at all: not in the list, not in a picker, not over the API or MCP, and its runners' live updates never reach you. The instance Owner (the Owner of the default workspace) sees every machine. Everyone else is put on a machine by a rule on it, for a role or for a person, and an Owner of another workspace is no exception.

What you can do on a machine you are on comes from three permissions: **Deploy to**, **Manage** and **Delete**. Your roles' `machines:deploy`, `machines:write` and `machines:delete` set them, and the machine's rules can allow or deny each on top. Adding a runner to a machine that exists, changing its stack root, discovering what runs on it and removing one of its runners all take **Manage**.

After an upgrade from a release that listed every machine to everyone who could read machines, only the instance Owner is on each machine, and the old `runners` and `machines:read` permissions are gone. Add people back with a rule on each machine.

## People on this machine

Each machine's band on the **Runners** page shows who its rules reach, for the instance Owner and anyone holding **Manage** on it: **Only you** on a machine nobody has a rule on, otherwise the roles and people they name. Click it, or open the band's **⋯** or right-click the band and pick **Permissions…**, to open the permissions panel.

The panel sets four permissions: **See**, **Deploy to**, **Manage** and **Delete**, each Allow, Fallback or Deny, for a role of any workspace (named with its workspace, such as "Member in Northwind") or for a person. Allowing See is what puts someone on the machine: the list, the pickers and the live updates start showing it to them. **Check access** shows a person's answer for each permission and the rule or role it came from. You can't allow what you don't hold on the machine, nor lift a Deny on it, but you can always add a Deny. A change shows at once to the person it names and to everyone who manages the machine.

With the API or MCP, a machine's rules are `resource_type: "machine"` on `PUT /api/permissions` and `permission_overwrite_update`, with the actions `machines:see`, `machines:deploy`, `machines:write` and `machines:delete`. `GET /api/machines` and `machine_list` list the actions you hold on each machine.

## Machines

Every runner belongs to a machine, and a stack deploys to a machine rather than to a runner. Any connected, idle runner on that machine picks up the job, so you scale a machine by adding runners to it. Jobs wait in the queue while no runner on the machine is connected.

Each machine has a stack root, the directory its stack checkouts live under (`stacks/<slug>/repo`). Click it on the machine's band on the **Runners** page to change it. A stack keeps its location once it has deployed.

## Deleting a machine

A machine you hold **Delete** on has **Delete** in the **⋯** menu on its band on the **Runners** page, and in its right-click menu. The confirm names what goes: the machine, the rules for who sees, deploys to, manages and deletes it, and any runner that is offline. Those runners lose their credential, so one that comes back online uninstalls itself. An offline machine is deleted the same way as any other.

A machine can't be deleted while anything uses it. The delete is refused and the message names what is in the way: each stack deployed to the machine (a branch deployment too), each gateway on it, and each runner on it that is connected. For example `machine "edge-box" is still used by stack api, stack web, connected runner build-box-1; move or remove them first`. Move a stack to another machine or delete it from its own page, delete the gateway, and remove a connected runner (see below), then delete the machine. Nothing is removed for you.

Deleting a machine takes **Delete** on it, not **Manage**, and a machine you can't see is not found. Everyone who saw the machine sees its band leave the page at once. To bring one back, add a runner on a machine of that name: you get every action on the new machine, and the old rules don't return. Agents delete a machine with `host_delete` and `kind: "machine"`, the machine's `id` from `machine_list`, and get the same refusal.

## Automations hosts

An [automations host](/docs/guide/automations/#where-automations-run) follows its machine, as a runner does. The Automations page lists the hosts on the machines you can see, and none on a machine you can't; `machine_list` files each under its machine. Adding a host on a machine that exists and removing one take **Manage** on it, placing an automation on one takes **Deploy to**, and a host on a machine Nexul doesn't know yet is the instance Owner's, since a host doesn't add a machine. A host code is checked again when it is used: if the person who made it no longer manages the machine, the code is refused and a new one is needed.

## Adding a runner

1. Open **Runners** and click **Add runner**. To add one to an existing machine, click **Add runner** on that machine's band instead (a **+** in a narrow window, or **Add runner** under a machine with no runners yet). A runner on a machine that exists takes **Manage** on it; one that makes a new machine takes the **Add machines** permission (`machines:create`), which stays with the instance Owner until they hand it out from **Settings → Instance → Access**.
2. Type a **Runner name**: lower case letters, digits and dashes, up to 32 characters. The runner joins a machine named after the computer's hostname, so runners on one computer share a machine.
3. Click **Create install command** and copy the command for the machine's system:

   ```sh
   curl -fsSL https://nexul.io/runner.sh | NEXUL_VERSION=v0.2.1 sh -s -- --server https://nexul.example.com --name build-box-1 --code nxe_…
   ```

   ```powershell
   $env:NEXUL_VERSION='v0.2.1'; & ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box-1 --code nxe_…
   ```

4. Run it on the machine: on Linux as root or with `sudo`, on a Mac as yourself, on Windows in PowerShell (it asks for administrator rights). The dialog shows **Server connected** under the command once a runner with that name connects, with its name and machine.

The install sets up Docker the same way the [server install](/docs/guide/install/) does, enrolls the runner and starts it as the `nexul-runner-<name>` service. The runner then shows as connected on the **Runners** page.

The code in the command works once, for an hour. If the install fails, create a new command. Agents can do the same with the `host_create` MCP tool and `kind: "runner"`.

### Revoking an unused code

A code nobody has used yet is listed on the **Runners** page under the machine it was made for, or under **No machine**, with the time it works until. Open its **⋯** and pick **Revoke code**, and it enrolls nothing; the command that carried it fails with an unknown code, so make a new command. You see the codes you made, the codes for a machine you have **Manage** on, and, as the instance Owner, every runner code. The code itself is never shown again, only its name and machine. Agents list them in `machine_list` (`open_enrollments`) and revoke one with `host_delete` and `code_id`; `host_create`'s result carries that id.

The code that adds a computer is the same kind of code. It is listed under **Unused computer codes** in your settings, to you alone, even if you are the instance Owner. See [Paired computers](/docs/guide/paired-computers/).

A runner that makes a new machine puts you on it with **See**, **Deploy to**, **Manage** and **Delete**, in the same step; you hold no more than that, and the instance Owner or anyone with Manage on the machine can take you off. The code checks you again when the runner uses it: if someone else's machine bears that name by then, or you can no longer add machines, the install is refused and you create a new command.

For more runners on one machine, run a new command per runner, each with its own name. Each one is a separate service, and the machine runs one job per runner at a time.

## Removing a runner

Open the **⋯** on the runner's row, pick **Remove runner** and confirm. It takes **Manage** on the runner's machine. Agents use `host_delete` with `kind: "runner"`. A machine with a connected runner can't be deleted until that runner is removed.

A connected runner uninstalls itself right away, and a job it was running fails. An offline runner uninstalls itself the next time it comes online, so a removed runner never comes back.

You can also remove it on the machine:

```sh
sudo nexul uninstall runner build-box-1
```

This removes the service and tells the instance. If the instance can't be reached, the runner still comes off the machine; remove it from the **Runners** page afterwards. `nexul status` lists the runners installed on a machine.

## Updates

Runners update themselves. When a runner connects to an instance on a different release, the instance tells it which release to run. The runner downloads it from the instance, checks its checksum, swaps the binary and restarts. A runner in the middle of a job finishes the job first. While a runner is behind, the **Runners** page shows **↑** after its version; hover it for the release it moves to.

## Importing what's already running

A machine may already run containers Nexul didn't deploy, such as a manual `docker run` or a Compose project. Open the **⋯** on the machine's band on the **Runners** page and pick **Import from this machine**. Choose the machine and the project the stacks will join, press **Discover**, and Nexul lists the containers, compose groups and gateways it finds, all ticked; untick what to leave alone and press **Import**. They are adopted as unmanaged stacks.

An unmanaged stack shows on the [topology canvas](/docs/guide/topology-and-dns/) and can be exposed, but cannot be deployed until you attach a repository to it. Deleting an unmanaged stack leaves its containers running, because Nexul didn't start them; attaching a repository makes it managed, and deleting it then removes its containers.
