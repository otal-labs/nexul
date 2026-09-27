---
title: Install
description: Put Nexul on a Linux server, a Mac or a Windows PC with one command.
sidebar:
  order: 1
---

Nexul runs as a self-hosted instance on your own server. One command installs it on a Linux server, and the same command on a Mac or Windows PC gives you an instance to try on your own computer first.

Every part of Nexul runs as a native service under the system's own service manager: systemd on Linux, launchd on macOS, and Windows services on Windows. Nothing of Nexul runs in a container, and the only port it opens is the one you choose. Docker is still installed, because the runner deploys your stacks with it. Containers reach Nexul at `host.docker.internal`, so on Linux the server also accepts its port from Docker's bridge interfaces while it runs (see [Firewall](#firewall)).

## On a Linux server

Run this as root, or as a user who can `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

You can [read the script](/install.sh) first. It downloads the `nexul` command for your server's CPU (amd64 or arm64), checks it against the release's `checksums.txt`, and runs `nexul install`. The installer asks two questions, each with a default you accept by pressing Enter:

```
Nexul v0.2.1 installer

Press Enter to accept the default.

  Install directory  [/data/nexul]:
  Web port           [80]:

  Docker .......... installed 29.8.1
  Docker Compose .. installed 2.40.3
  Git ............. present
  Command ......... /usr/local/bin/nexul
  User ............ nexul, created
  Files ........... /data/nexul
  Logs ............ OpenObserve v1.0.4 on 127.0.0.1:41873
  Server .......... running on port 80
  Runner .......... nexul-runner-instance
  Automations ..... nexul-automations-instance

Nexul v0.2.1 is running.
  Setup wizard   http://203.0.113.4/
  Data           /data/nexul  (back this folder up)
  Logs UI        http://203.0.113.4/openobserve/  user nexul@nexul.local
  Logs password  …  (also in /data/nexul/.env)
  Upgrade        nexul upgrade
  Status         nexul status
```

What each step does:

1. **Docker, Docker Compose and Git.** The runner deploys stacks with Docker and clones your repositories with git, so these are installed when missing. Docker Engine comes from Docker's own script, and the daemon is started when it is installed but stopped. Docker installed from snap is refused, because it cannot reach `/data`. Compose comes from your package manager (`docker-compose-plugin`, or `docker-compose-v2` next to Ubuntu's own `docker.io`), and otherwise from Docker's published build.
2. **Command.** Puts `nexul` in `/usr/local/bin`, so you can run `nexul status` and `nexul upgrade` later.
3. **User.** Creates the `nexul` system user the server, OpenObserve and the automations host run as.
4. **Files.** Creates the install directory and writes its `.env` with the generated logs password and token.
5. **Logs.** Downloads OpenObserve at the version and checksum pinned in `nexul`, and starts it as `nexul-openobserve`, listening on a free localhost port.
6. **Server.** Downloads `nexul-server` and starts it as `nexul-server` on your web port, then waits until it answers.
7. **Runner and Automations.** Installs the instance's own runner and automations host, both named `instance`, as `nexul-runner-instance` and `nexul-automations-instance`. They enroll with the server like any other runner or automations host, so this server can deploy and run automations straight away.

A port that is already in use is caught before anything is installed, and you are asked for another one.

### Firewall

Deployed containers, cloudflared first, reach Nexul at `http://host.docker.internal:<web port>`. That traffic arrives at the server as inbound, which many cloud images reject unless it's SSH. So while `nexul-server` runs, it adds `iptables` rules accepting the web port from Docker's bridge interfaces (`docker0` and `br-*`), and it removes them when it stops. No other interface is opened, the rules aren't saved to your firewall configuration, and the install summary prints a `Firewall` line saying so. On a host without `iptables` the step is skipped.

To install without questions, pass the answers as flags:

```sh
curl -fsSL https://nexul.io/install.sh | sh -s -- --dir /srv/nexul --port 8080 --yes
```

`NEXUL_VERSION=v0.2.1` before `sh` pins a release. Without it the script takes the newest stable release, or the newest beta while no stable release exists.

Running `nexul install` again is safe: it keeps the directory, the port and the logs credentials it chose the first time, so it also repairs an install. A directory that still holds a Docker Compose install from an earlier release is refused; run that release's `nexul uninstall` first.

There's nothing else to configure first. The server generates its auth secret on first start. The instance URL, the GitHub App and connectors are collected in the [setup wizard](/docs/guide/setup-wizard/).

### What's installed

| Service | Runs as | Purpose |
| --- | --- | --- |
| `nexul-server` | `nexul` | The web UI, the API, the runner and automations WebSockets, the MCP server, and the logs UI at `/openobserve/`, all on the web port |
| `nexul-openobserve` | `nexul` | Logs, metrics and traces, on localhost only. See [Logs](/docs/guide/logs/) |
| `nexul-runner-instance` | root | The `instance` runner that builds and deploys on this server. See [Runners](/docs/guide/runners/) |
| `nexul-automations-instance` | `nexul` | The `instance` automations host, which runs the default automations. See [Automations](/docs/guide/automations/) |

Each service has its own directory under `/opt/nexul` (`server/`, `openobserve/`, `runner-instance/`, `automations-instance/`) holding its binary, its environment file and, for a runner or automations host, its credential. `journalctl -u <service>` shows what a service is doing.

The install directory holds everything that belongs to the instance:

| Path | Contents |
| --- | --- |
| `.env` | The release, the ports and the logs credentials |
| `data/` | The database, its automatic snapshots and the generated secrets |
| `logs/` | OpenObserve's storage |
| `stacks/` | Checkouts of the stacks you deploy to this server. The install directory is the instance runner's stack root, which you can change on the Runners page |

Back up the install directory and you have backed up the instance.

The server listens on plain HTTP. For a public domain, put a Cloudflare tunnel or your own TLS-terminating proxy in front of it and enter the `https://` address as the instance URL in the setup wizard. Every URL Nexul derives (OAuth callbacks, the runner install command, the MCP endpoint) comes from that saved instance URL rather than the incoming request, so the proxy doesn't need to forward any extra headers.

### Managing the install

```sh
nexul status              # every Nexul service on this machine: kind, name, state and version
nexul upgrade             # the newest release on your channel; see Upgrade
nexul uninstall           # stop and remove every Nexul service, keeping the install directory
nexul uninstall --purge   # also delete the install directory and, on Linux, the nexul user
```

`nexul uninstall` asks before it removes anything, and it leaves the stacks you deployed running. Installing again with `nexul install --dir <the same directory>` brings the same instance back.

## On a Mac

Run this in Terminal as yourself, not with `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

It asks for your password once, to put the `nexul` command in `/usr/local/bin`. Then `nexul install` sets up Docker for the runner the way your Mac allows:

- **Docker already running** (Docker Desktop, Colima or another engine): used as it is.
- **Installed but stopped:** Colima is started, or Docker Desktop is opened, and the installer waits for it.
- **No Docker at all:** it installs [Colima](https://github.com/abiosoft/colima), a free Docker engine without a desktop app, with the Docker CLI and Compose through Homebrew, and sets Colima to start when you log in. If Homebrew itself is missing, Homebrew's installer runs first and asks for your password.

The services are LaunchAgents of your user (`io.nexul.nexul-server` and so on), so they run while you are logged in. Their directories are under `~/Library/Application Support/nexul`, and their output goes to `~/Library/Logs/nexul/<service>.log`. The install directory defaults to `~/nexul`.

## On Windows

Install [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) first (or run `winget install -e --id Docker.DockerDesktop`), start it, and wait until it shows the engine running. The runner needs it to deploy stacks. Then, in PowerShell:

```powershell
irm https://nexul.io/install.ps1 | iex
```

You can [read the script](/install.ps1) first. Nexul installs as Windows services, so the script asks for administrator rights and re-runs itself elevated. It downloads `nexul.exe`, checks it against the release's `checksums.txt`, puts it in `%ProgramFiles%\Nexul` on the system PATH, and runs `nexul install`. The installer doesn't install Docker Desktop for you: if Docker is missing, stopped or has no Compose plugin, it stops and says what to do.

Each service is registered with `nexul service-host <service>` as its program, which runs the component and restarts it if it stops. Service directories are under `%ProgramData%\Nexul`, and each one's output goes to `service.log` in its directory. The install directory defaults to `%USERPROFILE%\nexul`.

## Running the server by hand

Any platform can also run just the server: download `nexul-server-<os>-<arch>` from the latest [release](https://github.com/otal-labs/nexul/releases) and run it:

```sh
./nexul-server
```

The binary embeds the web UI, so it serves the UI and the API on port 8080 (`NEXUL_HTTP_ADDR` changes it) with nothing else installed. It doesn't include a runner or OpenObserve. Add runners from the Runners page (see [Runners](/docs/guide/runners/)), and point logs at any OTLP/HTTP backend with `NEXUL_OTLP_ENDPOINT` (see [Logs](/docs/guide/logs/)). You can also build it yourself:

```sh
make build-single
./dist/nexul-server
```

## Next step

Open the printed URL to reach the setup wizard and connect your [GitHub App](/docs/guide/github-app/). See [Setup wizard](/docs/guide/setup-wizard/) for what each step asks for, and [Upgrade](/docs/guide/upgrade/) for keeping the instance current.
