---
title: Install
description: Put Nexul on a Linux server, a Mac or a Windows PC with one command.
sidebar:
  order: 1
---

Nexul runs as a self-hosted instance on your own server. One command sets it up on a Linux server, and the same command on a Mac or Windows PC gives you an instance to try on your own computer first.

## On a Linux server

Run this as root, or as a user who can `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

You can [read the script](/install.sh) first. It downloads the `nexul` command for your server's CPU (amd64 or arm64), checks it against the release's `checksums.txt`, puts it in `/usr/local/bin`, and runs `nexul install`. The installer asks three questions, each with a default you accept by pressing Enter:

```
Nexul v0.2.1 installer

Press Enter to accept the default.

  Install directory  [/data/nexul]:
  Web port           [80]:
  Logs UI port       [5080]:

  Docker .......... installed 29.8.1
  Docker Compose .. 2.40.3
  Git ............. present
  Files ........... /data/nexul
  Images .......... pulled
  Stack ........... running
  Runner .......... nexul-runner.service
  Command ......... /usr/local/bin/nexul

Nexul v0.2.1 is running.
  Setup wizard   http://203.0.113.4/
  Data           /data/nexul  (back this folder up)
  Logs UI        http://203.0.113.4:5080/  user nexul@nexul.local
  Logs password  …  (also in /data/nexul/.env)
```

What each step does:

1. **Docker.** Installs Docker Engine with Docker's own script when it is missing, and starts the daemon when it is installed but stopped. Docker installed from snap is refused, because it cannot reach `/data`.
2. **Docker Compose.** When `docker compose` is missing, installs the Compose plugin from your package manager (`docker-compose-plugin`, or `docker-compose-v2` next to Ubuntu's own `docker.io`), and otherwise downloads Docker's published build.
3. **Git.** The runner clones your repositories, so git is installed if it is missing.
4. **Files.** Writes `docker-compose.yml` and a generated `.env` (the logs password and token) into the install directory.
5. **Images and Stack.** Pulls the release's images and starts the server, the logs stack and the automations host, then waits until the server answers.
6. **Runner.** Installs the instance runner as the `nexul-runner` systemd service, so this server can deploy straight away.

A port that is already in use is caught before anything is installed, and you are asked for another one.

To install without questions, pass the answers as flags:

```sh
curl -fsSL https://nexul.io/install.sh | sh -s -- --dir /srv/nexul --port 8080 --logs-port 5081 --yes
```

`NEXUL_VERSION=v0.2.1` before `sh` pins a release. Without it the script takes the newest stable release, or the newest beta while no stable release exists.

Running `nexul install` again is safe: it keeps the directory, the ports and the logs credentials it chose the first time, so it also repairs an install.

There's nothing else to configure first. The server generates its auth secret and the runner secret on first start. The instance URL, the GitHub App and connectors are collected in the [setup wizard](/docs/guide/setup-wizard/).

### What's installed

| Part | Where | Purpose |
| --- | --- | --- |
| `server` | container | The web UI, the API, the runner WebSocket and the MCP server, all on the web port |
| `openobserve` | container | Logs, metrics and traces on the logs port. See [Logs](/docs/guide/logs/) |
| `automations` | container | Runs the instance's default automations |
| `nexul-runner` | systemd service | The `instance` runner that builds and deploys on this server |
| `nexul` | `/usr/local/bin` | The command you upgrade, check and remove the install with |

The install directory holds everything that belongs to the instance:

| Path | Contents |
| --- | --- |
| `docker-compose.yml` | The stack, written by `nexul install` and `nexul upgrade` |
| `.env` | The release, the ports and the logs credentials |
| `data/` | The database, its automatic snapshots and the generated secrets |
| `logs/` | OpenObserve's storage |
| `stacks/` | Checkouts of the stacks you deploy to this server. This is the machine's stack root, `/data/nexul` unless you change it on the Runners page, so it lands here with the default directory |

Back up the install directory and you have backed up the instance.

The stack listens on plain HTTP. For a public domain, put a Cloudflare tunnel or your own TLS-terminating proxy in front of it and enter the `https://` address as the instance URL in the setup wizard. Every URL Nexul derives (OAuth callbacks, the runner install command, the MCP endpoint) comes from that saved instance URL rather than the incoming request, so the proxy doesn't need to forward any extra headers.

### Managing the install

```sh
nexul status              # version, directory, runner and containers
nexul upgrade             # the newest release on your channel; see Upgrade
nexul uninstall           # stop and remove everything except the install directory
nexul uninstall --purge   # also delete the install directory
```

`nexul uninstall` asks before it removes anything, and it leaves the stacks you deployed running. Installing again with `nexul install --dir <the same directory>` brings the same instance back.

## On a Mac

Run this in Terminal as yourself, not with `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

It asks for your password once, to put the `nexul` command in `/usr/local/bin`. Then `nexul install` sets up Docker the way your Mac allows:

- **Docker already running** (Docker Desktop, Colima or another engine): used as it is.
- **Installed but stopped:** Colima is started, or Docker Desktop is opened, and the installer waits for it.
- **No Docker at all:** it installs [Colima](https://github.com/abiosoft/colima), a free Docker engine without a desktop app, with the Docker CLI and Compose through Homebrew, and sets Colima to start when you log in. If Homebrew itself is missing, Homebrew's installer runs first and asks for your password.

The install directory defaults to `~/nexul`, because Docker on a Mac only shares your home folder with its VM.

## On Windows

Install [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) first (or run `winget install -e --id Docker.DockerDesktop`), start it, and wait until it shows the engine running. Then, in PowerShell:

```powershell
irm https://nexul.io/install.ps1 | iex
```

You can [read the script](/install.ps1) first. It downloads `nexul.exe`, checks it against the release's `checksums.txt`, puts it in `%LOCALAPPDATA%\Programs\Nexul` on your PATH, and runs `nexul install`. The installer doesn't install Docker Desktop for you: if Docker is missing, stopped or has no Compose plugin, it stops and says what to do. The install directory defaults to `%USERPROFILE%\nexul`.

### How a Mac or Windows install differs

- The instance runner runs as a container in the stack, because these systems have no systemd.
- The database and logs live in Docker volumes (`nexul_nexul-data`, `nexul_nexul-logs`), not in the install directory, which holds only the compose files and `.env`.
- The **Upgrade** button in the web UI can't upgrade these installs; run `nexul upgrade` in a terminal instead.
- `nexul uninstall --purge` also removes the Docker volumes.

## The single binary

Any platform can also run just the server: download `nexul-<os>-<arch>` from the latest [release](https://github.com/otal-labs/nexul/releases) and run it:

```sh
./nexul serve
```

The binary embeds the web UI, so it serves the UI and the API on port 8080 with nothing else installed. It doesn't include a runner or the logs stack. Add runners from the Runners page (see [Runners](/docs/guide/runners/)), and point logs at any OTLP/HTTP backend with `NEXUL_OTLP_ENDPOINT` (see [Logs](/docs/guide/logs/)). You can also build it yourself:

```sh
make build-single
./dist/nexul serve
```

## Next step

Open the printed URL to reach the setup wizard and connect your [GitHub App](/docs/guide/github-app/). See [Setup wizard](/docs/guide/setup-wizard/) for what each step asks for, and [Upgrade](/docs/guide/upgrade/) for keeping the instance current.
