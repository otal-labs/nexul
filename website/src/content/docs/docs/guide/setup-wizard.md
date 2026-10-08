---
title: Setup wizard
description: Give a fresh instance its domain, connect the GitHub App, and sign in as the owner.
sidebar:
  order: 3
---

Setup runs in a fixed order: the domain first, then the GitHub App on that domain, then your owner account. GitHub sends you back to the instance's final `https://` address after sign-in, so that address has to exist before the App does.

## 1. Enter the setup code

Open the setup page the installer printed, `http://<server>:5123/`, and enter the **Setup code** from the same summary (`nxs_…`). Lost it? Run `sudo nexul status` on the server.

The code proves you installed the server, so a stranger who finds the port cannot take over the instance. A correct code unlocks setup in this browser for an hour. The server writes a new code each time it starts, each one good for a day, and setup closes for good once someone has signed in.

## 2. Give Nexul a domain

Pick how traffic reaches Nexul. There is no skip, because GitHub sign-in needs the domain.

- **Cloudflare tunnel**: no open ports. Paste a Cloudflare API token (see [the permissions it needs](/docs/guide/topology-and-dns/#cloudflare-api-token-permissions)), pick the account that owns your domain, deploy the tunnel, then choose the subdomain and zone. Cloudflare issues the certificate.
- **Reverse proxy**: type the domain. The page shows the A (and AAAA) record to create at your DNS provider and waits until the domain resolves to this server. Nexul then runs Traefik on ports 80 and 443 with a Let's Encrypt certificate. Both ports must be reachable from the internet.
- **I already have HTTPS**: your own proxy already serves an `https://` address that forwards to `http://<server>:5123`. Type it in.

Each path ends with Nexul checking that the address answers over HTTPS, then saving it as the instance URL.

On a Mac or Windows install this step is skipped and the instance stays on `http://localhost:5123`.

## 3. Continue on your domain

The page links to your new address. The link carries the setup code, so you land straight on the next step.

## 4. Connect a GitHub App

Create the [GitHub App](/docs/guide/github-app/) if you haven't yet. The page shows the callback URL to register on it, `<instance-url>/auth/callback`, then asks for:

- **GitHub OAuth client ID** and **GitHub OAuth client secret**, from the App's page.
- **GitHub App slug**, the name in the App's URL, `github.com/apps/<slug>`.

Click **Verify**. Three checks run: **Instance URL reaches this server**, **App slug resolves**, and **Client secret accepted**. When all three pass, **Set up instance** sends you to GitHub to sign in as the first user.

Got a value wrong? Until someone has signed in, open `/setup` on your domain to correct it.

## 5. Owner wizard

The first person to sign in becomes the workspace owner and walks through four steps:

1. **Introduce yourself**: the name and avatar others see. The GitHub ones are the default.
2. **Set up your workspace**: name it. A new workspace has no projects yet.
3. **Connect your tools**: the same list as **Settings → Connectors**. Cloudflare already shows as connected if you used the tunnel.
4. **Set up T3 Code**: pair the computer your agents run on, the same dialog as **Settings → T3 Code Setup**. This step is required: setup finishes once one computer is paired. Pairing through a tunnel needs Cloudflare connected with Zero Trust enabled, so connect it in the previous step if you chose another way to reach Nexul. See [paired computers](/docs/guide/paired-computers/).

Then the [project wizard](/docs/guide/projects-and-repositories/#creating-a-project) opens to create your first project.

Everyone who signs in after you only confirms their name and avatar, then lands in the app.

## If something goes wrong

- **"That code is wrong or has expired"**: the server restarted and wrote a new code, or yours is over a day old. Run `sudo nexul status` for the current one, or `sudo systemctl restart nexul-server` for a fresh one.
- **"Too many wrong codes from this address"**: wait a few minutes and try again.
- **"Setup is already done on this instance"**: someone has signed in. Sign in instead.
- **The domain check never passes**: for the reverse proxy, check that the DNS record points at this server and that ports 80 and 443 are open. For your own proxy, check it forwards to the web port.
