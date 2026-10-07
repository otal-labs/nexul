---
title: Install
description: Put Nexul on a Linux server with one command, or try it on your Mac or Windows PC first.
sidebar:
  order: 1
---

Every part of Nexul runs as a native service under the system's service manager: systemd on Linux, launchd on a Mac, Windows services on Windows. Nothing of Nexul runs in a container. Docker still gets installed, because the runner deploys your stacks with it.

## On a Linux server

Run this as root, or as a user who can `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

You can [read the script](/install.sh) first. It downloads the `nexul` command for your CPU (amd64 or arm64), checks it against the release's `checksums.txt`, and runs `nexul install`. Press Enter to accept each default:

```
Nexul v0.2.1 installer

Press Enter to accept the default.

  Install directory  [/data/nexul]:
  Web port           [5123]:

  Docker .......... installed 29.8.1
  Docker Compose .. installed 2.40.3
  Git ............. present
  Command ......... /usr/local/bin/nexul
  User ............ nexul, created
  Files ........... done
  Logs ............ OpenObserve v1.0.4 on 127.0.0.1:41873
  Server .......... running on port 5123
  Runner .......... nexul-runner-instance
  Automations ..... nexul-automations-instance

Nexul v0.2.1 is running.
  Setup page     http://203.0.113.4:5123/
  Setup code     nxs_…
  Data           /data/nexul  (back this folder up)
  Logs UI        http://203.0.113.4:5123/openobserve/  user nexul@nexul.local
  Logs password  …  (also in /data/nexul/.env)
  Upgrade        nexul upgrade
  Status         nexul status
  Firewall       port 5123 now accepts Docker containers on this machine, so a tunnel can reach Nexul
```

Open the setup page and enter the setup code. The [setup wizard](/docs/guide/setup-wizard/) takes it from there, including the domain and HTTPS. The web port defaults to 5123 so ports 80 and 443 stay free for the reverse proxy the wizard can deploy.

To install without questions, pass the answers as flags:

```sh
curl -fsSL https://nexul.io/install.sh | sh -s -- --dir /srv/nexul --port 8080 --yes
```

Put `NEXUL_VERSION=v0.2.1` before `sh` to pin a release. Without it you get the newest stable release, or the newest beta while there is no stable one.

The installer:

- installs Docker, Docker Compose and git when they are missing. It refuses Docker from snap, which cannot reach `/data`.
- creates a `nexul` system user and the install directory, and generates the logs password.
- starts OpenObserve for [logs](/docs/guide/logs/), the server, and the instance's own runner and automations host, both named `instance`.

If the port you pick is taken, it asks for another before installing anything. Running it again is safe: it keeps the directory, port and credentials it chose the first time, so it also repairs a broken install.

### Firewall

Deployed containers, cloudflared first, reach Nexul at `http://host.docker.internal:<web port>`. Many cloud images reject that inbound traffic. So while `nexul-server` runs, it adds `iptables` rules accepting the web port from Docker's bridge interfaces (`docker0` and `br-*`), and removes them when it stops. No other interface is opened and your saved firewall configuration is left alone. On a host without `iptables` this is skipped.

### What's installed

| Service | Runs as | What it does |
| --- | --- | --- |
| `nexul-server` | `nexul` | The web app, the API, the MCP server and the logs UI, all on the web port |
| `nexul-openobserve` | `nexul` | Logs, metrics and traces, on localhost only |
| `nexul-runner-instance` | root | Builds and deploys on this server. See [Runners](/docs/guide/runners/) |
| `nexul-automations-instance` | `nexul` | Runs the default automations. See [Automations](/docs/guide/automations/) |

Each service's binary and settings live under `/opt/nexul`. Run `journalctl -u <service>` to see what one is doing.

Everything that belongs to the instance lives in the install directory:

| Path | Contents |
| --- | --- |
| `.env` | The release, the ports and the logs credentials |
| `data/` | The database, its snapshots and the generated secrets |
| `logs/` | OpenObserve's storage |
| `stacks/` | Checkouts of the stacks this server deploys |

Back up the install directory and you have backed up the instance.

### Managing the install

```sh
sudo nexul status              # every Nexul service here, its state and version, plus the setup code while there is one
sudo nexul upgrade             # see Upgrade
sudo nexul uninstall           # remove every Nexul service, keep the install directory
sudo nexul uninstall --purge   # also delete the install directory and the nexul user
```

Uninstall asks before it removes anything and leaves your deployed stacks running. `nexul install --dir <the same directory>` brings the same instance back.

## On a Mac

Run the same command in Terminal as yourself, not with `sudo`:

```sh
curl -fsSL https://nexul.io/install.sh | sh
```

It asks for your password once, to put `nexul` in `/usr/local/bin`. For Docker it uses an engine that is already running, starts Colima or Docker Desktop if one is installed but stopped, and otherwise installs Colima through Homebrew (installing Homebrew first if needed).

Run `nexul status` and the other `nexul` commands without `sudo` here. The services run while you are logged in. Their output goes to `~/Library/Logs/nexul/<service>.log`, and the install directory is `~/nexul`.

## On Windows

Install [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) (or `winget install -e --id Docker.DockerDesktop`), start it, and wait until the engine is running. Then, in PowerShell:

```powershell
irm https://nexul.io/install.ps1 | iex
```

You can [read the script](/install.ps1) first. It asks for administrator rights, puts `nexul.exe` in `%ProgramFiles%\Nexul`, and runs `nexul install`. If Docker is missing, stopped or has no Compose plugin, it stops and tells you what to do. Each service writes `service.log` in its folder under `%ProgramData%\Nexul`, and the install directory is `%USERPROFILE%\nexul`.

On a Mac or Windows PC the instance stays on `http://localhost:5123` for you to try on your own computer. To serve a team, install on a server.

## Running the server by hand

To run only the server, download `nexul-server-<os>-<arch>` from the [releases](https://github.com/otal-labs/nexul/releases) and start it:

```sh
./nexul-server
```

It serves the web app and the API on port 8080 (`NEXUL_HTTP_ADDR` changes it). There is no runner and no OpenObserve: add runners from the **Runners** page, and send logs elsewhere with `NEXUL_OTLP_ENDPOINT` (see [Logs](/docs/guide/logs/#a-server-run-by-hand)).
