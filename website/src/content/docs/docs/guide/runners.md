---
title: Runners
description: The host binaries that build and deploy on your servers, and how to add more of them.
sidebar:
  order: 5
---

A runner is a lightweight host binary that connects out to your instance over a WebSocket, authenticates, and executes builds and deploys. The server holds no SSH keys and never connects into a target itself — target credentials live on the runner, and every deploy runs as `docker run` / `docker compose up` against the Docker daemon on the runner's own machine.

## The instance runner

`nexul install` installs one runner on the server itself, named `instance`, as the `nexul-runner` systemd service. That's enough to build and deploy out of the box, with nothing to configure first. `journalctl -u nexul-runner` shows what it is doing.

## Machines

Every runner reports the **machine** it runs on — a name, defaulting to the host's hostname, sent as `NEXUL_MACHINE`. A stack targets a machine by name, not a specific runner: any idle, connected runner reporting that machine name can pick up the job. Two or more runners reporting the same machine form a pool — the runner count on a machine is the only scaling knob, so you scale a machine by starting another runner with the same `NEXUL_MACHINE`.

Work queues while no runner on the named machine is connected, and flushes the moment one connects.

Each machine also has a **stack root** — the directory on that host under which every stack's checkout lives (`/data/nexul` by default). It's a per-machine setting, editable from the Runners page, and sent to the runner in every deploy so it knows where to check out and bind-mount from.

## Adding a runner

Open **Runners** in the web UI and click **Add runner** (or, from an existing machine's group, **Add a runner to this machine** to lock it onto that machine's pool). The dialog asks for:

- **Platform** — Linux/macOS or Windows.
- **GitHub token** — used to clone private repositories; leave it blank to fill in `<github-token>` yourself later.
- **Runner name** — optional, otherwise the runner reports its hostname.

It then hands you a one-line command with the server's WebSocket URL and the shared runner secret already filled in:

```sh
S="<secret>" && curl -fsSL -H "Authorization: Bearer $S" \
  "<instance-url>/api/runners/download/$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')" \
  -o nexul-runner && chmod +x nexul-runner && \
  NEXUL_SERVER_WS="<instance-url>/ws/runner" NEXUL_RUNNER_SECRET="$S" \
  NEXUL_GIT_TOKEN="<github-token>" ./nexul-runner
```

The download comes from your own instance, which serves the matching runner
asset for the instance release. The GitHub App installation is used when a
runner clones a private application repository, not for this runner download.
See [GitHub App](/docs/guide/github-app/) for the repository permission setup.

The shared runner secret is generated on first start and lives in instance settings; seed it yourself with `NEXUL_RUNNER_SECRET` if you need a fixed value ahead of time.

## Updates

A runner reports its version when it connects. If the server is a stable or beta build and the runner is
running something different, the server pushes it the release it should be on. The runner downloads the matching
binary from the server's own download route, verifies its checksum, swaps it in for the one it's running, and
restarts itself into the new version — no separate update step, and nothing to re-run on the machine. A runner
mid-job finishes the job first, then updates on its own next connect. The `instance` runner updates the same way.
Only a runner running inside a container skips this; it follows its image instead.

## Importing what's already running on a machine

If a machine already has containers running outside Nexul — a manual `docker run`, an existing Compose project — point the project wizard's **Import from this machine** action at it. Nexul asks the runner to discover what's there (`docker ps`, `docker inspect`, `docker network ls`) and lets you adopt containers, whole compose groups, or existing gateways as unmanaged stacks. An unmanaged stack can be seen and wired up on the [topology canvas](/docs/guide/topology-and-dns/), but stays undeployable from Nexul until you attach a repository to it.

## Next step

With a runner connected, deploy your first [stack](/docs/guide/stacks-and-deploys/).
