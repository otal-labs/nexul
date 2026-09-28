---
title: Setup Wizard
description: What happens the first time you open a freshly installed Nexul instance.
sidebar:
  order: 3
---

The first time you open your instance, nobody exists yet, so there's nothing to sign in to. Setup gives Nexul its domain first, then connects the GitHub App from that domain, then runs the owner wizard.

The order matters: GitHub sends you back to the instance's final `https://` address after sign-in, so that address has to exist before the GitHub App does.

## 1. Setup code

Open the setup page the installer printed, `http://<server>:5123/`. It asks for the **setup code** from the same summary (`nxs_…`). Lost it? Run `sudo nexul status` on the server.

The code proves you're the one who installed the server, so nobody who stumbles on the port can take over the instance or use your Cloudflare token. A correct code unlocks setup in this browser for an hour. The code changes every time the server restarts and stops working once the first person has signed in.

## 2. Domain

Choose how traffic reaches Nexul. There's no skip: nothing after this works without a domain.

- **Cloudflare tunnel.** No open ports. Paste a Cloudflare API token; each permission is checked before it's saved. Pick the Cloudflare account that owns your domain, deploy the tunnel, then choose the subdomain and zone. A checklist confirms the tunnel route, the proxied DNS record, and that the hostname answers over HTTPS. Cloudflare issues the certificate.
- **Reverse proxy.** Type the domain. The page shows this server's public address and the A (and AAAA) record to create at your DNS provider, and waits until the domain resolves here. Nexul then runs Traefik on ports 80 and 443, which gets a Let's Encrypt certificate and forwards the domain to Nexul. Both ports must be reachable from the internet.
- **I already have HTTPS.** Your own proxy (nginx, Caddy, a load balancer) already serves an `https://` address that forwards to `http://<server>:5123`. Type it and Nexul checks that it answers.

Every path ends the same way: Nexul checks that the address answers over HTTPS and stores it as the instance URL. See [Topology and DNS](/docs/guide/topology-and-dns/) for more on each path.

On a Mac or Windows install, which is for trying Nexul on one computer, this step is skipped and the instance stays on `http://localhost:5123`.

## 3. Continue on your domain

The page says where Nexul is live and links there. The link carries the setup code, so the domain opens straight at the next step.

## 4. GitHub App (on the domain)

Before anyone can sign in, the instance needs a [GitHub App](/docs/guide/github-app/) to authenticate through. The instance URL is fixed to the domain, and the page shows the exact OAuth callback URL to register on the App (`<instance-url>/auth/callback`). It asks for:

- **GitHub OAuth client ID** and **client secret**, from the App's page.
- **GitHub App slug**, the name in the App's own URL, `github.com/apps/<slug>`.

Each value is verified live, one row at a time: the instance URL reaches this server, the App slug resolves on GitHub, and the client secret is accepted. Once every row is green, **Set up instance** hands off to GitHub to sign you in as the first user.

Until someone has signed in, `/setup` on the domain lets you correct the GitHub App settings. Afterwards, use the settings in the app instead.

## 5. Owner wizard

The very first person to sign in becomes the workspace owner and lands in a three-step wizard:

1. **Introduce yourself.** Set the name and avatar other members will see, or keep the GitHub defaults.
2. **Set up your workspace.** Name the workspace. That's all: a workspace starts with no project.
3. **Connect your tools.** The same connectors list as Configuration → Connectors. Cloudflare already shows as connected if you took the tunnel path.

Finishing opens the [project wizard](/docs/guide/projects-and-repositories/#creating-a-project) to create your first project. The tunnel or reverse proxy set up in step 2 belongs to the instance, not to a project, so it's already running before any project exists.

## 6. Everyone else: first-login wizard

Teammates who sign in after the owner don't see the workspace setup, because the workspace already exists. They confirm the name and avatar they want to use, then land straight in the app.

## Next step

The project wizard takes your first project from a name to a deployed service. See [Projects and repositories](/docs/guide/projects-and-repositories/) and [Stacks and deploys](/docs/guide/stacks-and-deploys/).
