---
title: Runners
description: The services that build and deploy on your servers, and how to add and remove them.
sidebar:
  order: 5
---

A runner is a small binary that runs as a service on a machine, connects out to your instance over a WebSocket with its own credential, and executes builds and deploys. The server holds no SSH keys and never connects into a target itself: target credentials live on the runner, and every deploy runs as `docker run` / `docker compose up` against the Docker daemon on the runner's own machine.

## The instance runner

`nexul install` installs one runner on the server itself, named `instance`, as the `nexul-runner-instance` service. That's enough to build and deploy out of the box, with nothing to configure first. On Linux, `journalctl -u nexul-runner-instance` shows what it is doing.

## Machines

Every runner belongs to a **machine**. A stack targets a machine by name, not a specific runner: any idle, connected runner on that machine can pick up the job. Two or more runners on the same machine form a pool, and the runner count on a machine is the only scaling knob, so you scale a machine by adding another runner to it.

Work queues while no runner on the machine is connected, and flushes the moment one connects.

Each machine also has a **stack root**: the directory on that host under which every stack's checkout lives, at `stacks/<slug>/repo` beneath it. The runner that creates the machine sets it when it enrolls (its own directory unless it was installed with `--stack-root`), and you can change it on the Runners page. It's sent to the runner in every deploy so it knows where to check out and bind-mount from.

## Adding a runner

Open **Runners** in the web UI and click **Add runner**, or **Add a runner to this machine** on an existing machine's group to add one to that machine's pool. The dialog asks for:

- **Runner name**: lower case letters, digits and dashes, up to 32 characters, unique on the instance.
- **Machine**: the machine it joins. Left empty, the runner gets a machine of its own with the runner's name.
- **GitHub token**: optional, used to clone private repositories. It goes into the command, never to the instance.

**Create install command** gives you one line for Linux or macOS and one for Windows, each carrying a one-time enrollment code:

```sh
curl -fsSL https://nexul.io/runner.sh | NEXUL_VERSION=v0.2.1 sh -s -- --server <instance-url> --name build-box-1 --code nxe_…
```

```powershell
$env:NEXUL_VERSION='v0.2.1'; & ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server <instance-url> --name build-box-1 --code nxe_…
```

Run it on the machine: on Linux as root or a user who can `sudo`, on a Mac as yourself, and on Windows in PowerShell, which asks for administrator rights. `NEXUL_VERSION` pins the runner to the instance's own release, and Docker is checked or set up exactly as the [server install](/docs/guide/install/) does it. The install trades the code for the runner's own credential and starts the runner as the `nexul-runner-<name>` service, with its directory under `/opt/nexul/runner-<name>` on Linux. The code works once and expires after an hour; if the install fails, create a new command.

The instance URL comes from settings, so set it in the setup wizard before adding a runner. The same enrollment is the `host_create` MCP tool with `kind: "runner"`.

### Several runners on one machine

Run the install command once per runner, each with its own name. Every runner is its own service with its own directory, binary and credential, so two runners on one machine never share state, and the machine takes one job per runner at a time. Unless the Add runner dialog names a machine, a runner joins the machine named after its computer's hostname, so runners installed on one computer share a machine.

## Removing a runner

**Remove** on a runner's row, or `host_delete` with `kind: "runner"`, revokes its credential and deletes it from the instance. A connected runner is told to uninstall itself and removes its own service from the machine. A runner that was offline is refused as removed when it next connects and uninstalls itself then, so a removed runner never comes back. A job it was running fails.

On the machine itself:

```sh
nexul uninstall runner build-box-1
```

This removes the service and its directory, and tells the instance to drop the runner. If the instance can't be reached, the runner still comes off the machine; remove it from the Runners page afterwards. `nexul status` lists every runner installed on the machine.

## Updates

A runner reports its version when it connects. If the server is a stable or beta build and the runner is running something different, the server pushes it the release it should be on. The runner downloads the matching binary from the server's own download route, verifies its checksum, swaps it in for the one it's running, and restarts itself into the new version: no separate update step, and nothing to re-run on the machine. A runner mid-job finishes the job first, then updates. The `instance` runner updates the same way.

## Importing what's already running on a machine

If a machine already has containers running outside Nexul — a manual `docker run`, an existing Compose project — point the project wizard's **Import from this machine** action at it. Nexul asks the runner to discover what's there (`docker ps`, `docker inspect`, `docker network ls`) and lets you adopt containers, whole compose groups, or existing gateways as unmanaged stacks. An unmanaged stack can be seen and wired up on the [topology canvas](/docs/guide/topology-and-dns/), but stays undeployable from Nexul until you attach a repository to it.

## Next step

With a runner connected, deploy your first [stack](/docs/guide/stacks-and-deploys/).
