---
title: Runners
description: Add the services that build and deploy on your machines, scale them, and remove them.
sidebar:
  order: 5
---

A runner is a service on one of your machines. It connects out to the instance over a WebSocket and runs `docker` and `docker compose` against that machine's own Docker. The instance never connects in and holds no SSH keys.

`nexul install` already put one on your server, named `instance`. You can deploy with it straight away.

## Machines

Every runner belongs to a machine, and a stack deploys to a machine rather than to a runner. Any connected, idle runner on that machine picks up the job, so you scale a machine by adding runners to it. Jobs wait in the queue while no runner on the machine is connected.

Each machine has a stack root, the directory its stack checkouts live under (`stacks/<slug>/repo`). Change it next to **root** on the machine's group on the **Runners** page. A stack keeps its location once it has deployed.

## Adding a runner

1. Open **Runners** and click **Add runner**. To add one to an existing machine, click **Add a runner to this machine** on that machine instead.
2. Fill in:
   - **Runner name**: lower case letters, digits and dashes, up to 32 characters.
   - **Machine**: the machine it joins. Leave it empty and the runner joins a machine named after the computer's hostname, so runners on one computer share a machine.
   - **GitHub token**: optional, for cloning private repositories. It goes into the command, never to the instance.
3. Click **Create install command** and copy the command for the machine's system:

   ```sh
   curl -fsSL https://nexul.io/runner.sh | NEXUL_VERSION=v0.2.1 sh -s -- --server https://nexul.example.com --name build-box-1 --code nxe_…
   ```

   ```powershell
   $env:NEXUL_VERSION='v0.2.1'; & ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box-1 --code nxe_…
   ```

4. Run it on the machine: on Linux as root or with `sudo`, on a Mac as yourself, on Windows in PowerShell (it asks for administrator rights).

The install sets up Docker the same way the [server install](/docs/guide/install/) does, enrolls the runner and starts it as the `nexul-runner-<name>` service. The runner then shows as connected on the **Runners** page.

The code in the command works once, for an hour. If the install fails, create a new command. Agents can do the same with the `host_create` MCP tool and `kind: "runner"`.

For more runners on one machine, run a new command per runner, each with its own name. Each one is a separate service, and the machine runs one job per runner at a time.

## Removing a runner

Click the remove button on the runner's row and confirm. Agents use `host_delete` with `kind: "runner"`.

A connected runner uninstalls itself right away, and a job it was running fails. An offline runner uninstalls itself the next time it comes online, so a removed runner never comes back.

You can also remove it on the machine:

```sh
sudo nexul uninstall runner build-box-1
```

This removes the service and tells the instance. If the instance can't be reached, the runner still comes off the machine; remove it from the **Runners** page afterwards. `nexul status` lists the runners installed on a machine.

## Updates

Runners update themselves. When a runner connects to an instance on a different release, the instance tells it which release to run. The runner downloads it from the instance, checks its checksum, swaps the binary and restarts. A runner in the middle of a job finishes the job first.

## Importing what's already running

A machine may already run containers Nexul didn't deploy, such as a manual `docker run` or a Compose project. Click **Import from this machine** on the machine's group on the **Runners** page. Nexul lists the containers, compose groups and gateways it finds, and you pick which to adopt as unmanaged stacks.

An unmanaged stack shows on the [topology canvas](/docs/guide/topology-and-dns/) and can be exposed, but cannot be deployed until you attach a repository to it.
