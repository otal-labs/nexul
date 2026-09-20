---
title: Install
description: Get a Nexul instance running with Docker Compose or a single binary.
sidebar:
  order: 1
---

Nexul runs as a self-hosted instance on your own machine or server. There are two ways to install it.

## Option A: Docker Compose (recommended)

This is the fastest path and starts everything you need: the server, the web UI, the logs stack, the automations host, and one runner on the same host. The stack pulls prebuilt images from GHCR rather than building from source — see [Upgrade](/docs/guide/upgrade/) for how those images get updated later.

Install [Git](https://git-scm.com/downloads) first. Run these commands in the directory where you want a new `nexul` folder.

Linux / macOS:

```sh
curl -fsSLo install-nexul.sh https://nexul.io/install.sh && bash install-nexul.sh
```

Windows, in PowerShell:

```powershell
& { Invoke-WebRequest https://nexul.io/install.ps1 -OutFile install-nexul.ps1 -ErrorAction Stop; & .\install-nexul.ps1 }
```

You can read the scripts at [nexul.io/install.sh](/install.sh) and [nexul.io/install.ps1](/install.ps1). They clone the repository and run its installer, with your terminal available for the password prompt. An existing `nexul` folder is left untouched.

Already cloned the repository? Run `./install.sh` on Linux / macOS or `.\install.ps1` in PowerShell from inside it. Use these repository scripts again for upgrades or to continue setup after starting Docker Desktop.

The script:

1. Installs Docker if it isn't already present (`get.docker.com` on Linux/macOS, `winget` for Docker Desktop on Windows).
2. Asks for a password for the logs UI. Leave it blank to generate one — it's saved to `.env` either way, so re-running the script keeps the same password.
3. Pulls the stack's images and starts it with `docker compose pull` then `docker compose up -d`.
4. Prints the URL of the setup wizard and the logs UI.

Re-running the script is safe: it keeps whatever logs password is already in `.env`, and it's also how you upgrade — see [Upgrade](/docs/guide/upgrade/).

There's nothing else to configure before you start. The server generates its own auth secret and the runner's shared secret on first start and keeps them on its data volume. Everything else — the instance URL, the GitHub App, connectors — is collected the first time you open the web UI, in the [setup wizard](/docs/guide/setup-wizard/).

If you'd rather skip the script, `docker compose up -d` starts the same stack using the dev defaults baked into `docker-compose.yml`.

### What's in the stack

| Service | Purpose |
| --- | --- |
| `server` | The API, the WebSocket endpoint for runners, and the MCP server |
| `web` | The React frontend, served by nginx on port 80 |
| `runner` | One bundled runner, named `instance`, that builds and deploys on the same host |
| `openobserve` | Logs, metrics, and traces — see [Logs](/docs/guide/logs/) |
| `automations` | Runs the instance's default automations |

The stack listens on plain HTTP on port 80. For a public domain, put a Cloudflare tunnel or your own TLS-terminating proxy in front of it, and enter the `https://` address as the instance URL in the setup wizard. Every URL Nexul derives — OAuth callbacks, the runner install command, the MCP endpoint — comes from that saved instance URL rather than the incoming request, so the proxy doesn't need to forward any extra headers.

## Option B: single binary

Download the `nexul-server-<os>-<arch>` tarball for your platform from the latest [release](https://github.com/otal-labs/nexul/releases), unpack it, and run it:

```sh
./nexul-server
```

The single-binary build embeds the web frontend directly (via `go:embed`), so the server serves the frontend on `/` and the API on `/api/*` — no nginx and no Node.js needed at runtime. You can also build it yourself:

```sh
make build-single
./dist/nexul-server
```

The single binary doesn't include a bundled runner or logs stack. Add runners from the web UI's Runners page (see [Runners](/docs/guide/runners/)), and point logs at any OTLP/HTTP backend with `NEXUL_OTLP_ENDPOINT` (details in [Logs](/docs/guide/logs/)).

## Next step

However you installed it, open the printed URL to reach the setup wizard and connect your [GitHub App](/docs/guide/github-app/). See [Setup wizard](/docs/guide/setup-wizard/) for what each step asks for, and [Upgrade](/docs/guide/upgrade/) for how to keep the instance current later.
