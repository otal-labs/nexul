---
title: Topology and DNS
description: The canvas that models your infrastructure, and how traffic reaches it from the internet.
sidebar:
  order: 7
---

## The topology canvas

Topology is the map of your infrastructure, drawn as a canvas — every service, the docker network it runs on, and how traffic reaches it. The canvas is a map, not a controller: service nodes, networks, gateway routes and hostnames are derived from what is actually deployed and cannot be edited on the canvas, and drawing an edge documents a relation without wiring anything. Only the hand-drawn nodes and positions are saved.

Open it from **Topology** in the sidebar. Each workspace has its own canvas: it shows the services of that workspace's projects and the gateways that route traffic to them, and it keeps its own layout and drawings. The canvas is built from a few node kinds:

- **Service nodes** — one per container, grouped inside the docker network it runs on.
- **Gateway nodes** — the hub every route into your infrastructure passes through (see below).
- **Hostname nodes** — the public hostnames pointed at a gateway.
- **External nodes** — things outside Nexul's management that a service still talks to.

Each node's accent and status badge reflect what the runner last observed — healthy, running, stopped, or failed.

## Gateways

A **gateway** is a Nexul-deployed service — a Cloudflare tunnel or a reverse proxy — that gives one docker network reachability from the internet. On the canvas, a gateway node is a hub: a hostname pill wires into it on the left, one row per exposure names where that traffic lands (`service:port`, with the container's observed address alongside it once known), and a wire goes out to the matching service node on the right.

Manage gateways directly from Settings → DNS → Gateways, or let the DNS setup stepper create your first one for you. A gateway's stack belongs to the instance rather than to a project: it shows in Settings → DNS, and on the Topology of every workspace it routes a hostname into, listing there only that workspace's hostnames. Its stack page links back to Topology.

## Exposures

An **exposure** routes one hostname through a gateway to one of a stack's containers. From a stack's Exposures section, pick a container and a port; Nexul resolves or provisions the gateway for you, so you never pick one directly — you only choose what should be reachable, and at what hostname.

## Setting up DNS

The instance gets its own domain during first run, before GitHub (see [Setup wizard](/docs/guide/setup-wizard/)). To change how it's reached later, or to set up DNS on an instance installed before that, select **Set up DNS** under Settings → DNS, which opens the DNS setup stepper at `/wizard/onboarding/dns`. It needs the Cloudflare connector connected first (see [GitHub App](/docs/guide/github-app/) for connecting the equivalent GitHub connector — Cloudflare connects with an API token the same way, from Settings → Connectors).

### Cloudflare API token permissions

Create the token in the Cloudflare dashboard with these permissions. The connector dialog checks each one and names any that are missing. The DNS check tries every zone the token can see and lists the domains it can edit, plus any it can only read; it passes when at least one is editable, so make sure the domain you plan to use is on the editable list.

| Permission | What Nexul uses it for |
| --- | --- |
| Zone → Zone: Read | Lists your zones and the Cloudflare account each belongs to. |
| Zone → DNS: Edit | Creates and updates the records that point your hostnames at Nexul. |
| Account → Cloudflare Tunnel: Edit | Creates tunnels and issues the token `cloudflared` runs with. |

Two more permissions are needed for pairing computers by tunnel. Without them the connector still connects; the dialog shows a warning on each missing one.

| Permission | What Nexul uses it for |
| --- | --- |
| Account → Access: Apps and Policies: Edit | Puts an Access rule on each paired computer's hostname so only this instance can reach it. |
| Account → Access: Service Tokens: Edit | Issues the one service token this instance presents to pass those Access rules. |

These two also need Zero Trust enabled on the Cloudflare account. Enable it once in the Cloudflare dashboard: pick a team name and the Free plan. Cloudflare asks for payment details even on the Free plan, but does not charge for it. Until then, the dialog warns that Zero Trust is not enabled. Checking these permissions never creates a token or an Access rule.

### 1. Choose an entry path

| Path | What it does |
| --- | --- |
| Bare public address | Points your hostname straight at the server with an A or AAAA record. |
| Cloudflare tunnel | No open ports — `cloudflared` runs as a service on a runner and routes the hostname through it. |
| Reverse proxy | Deploys Traefik on ports 80 and 443 of a machine. It gets Let's Encrypt certificates and forwards the instance's domain to Nexul. |

Nexul deploys your chosen entry path itself, like any other service — there's nothing to install on the server by hand.

### 2. Deploy the tunnel (tunnel path only)

Name the tunnel, and pick the machine it runs on and the Docker network it should reach. The tunnel's `cloudflared` stack belongs to the instance, not to a project, like every gateway. Nexul creates the tunnel at Cloudflare and runs `cloudflared` on that machine until it connects.

If your token reaches more than one Cloudflare account, choose the account that owns the domain you'll use. A tunnel only serves hostnames in its own account's zones, so the next step only offers that account's zones, and Nexul refuses to route a hostname from another account.

### 3. Point your hostname

- **Bare path** — pick a Cloudflare zone and record type, and give the target address.
- **Tunnel path** — pick the zone and subdomain; Nexul routes it into the tunnel you just deployed and creates its DNS record. Leave the local service empty and the hostname goes to this instance. A checklist then confirms the ingress rule, the proxied DNS record, and that the hostname answers over HTTPS from the server, which can take a minute for a new record. Continue unlocks once all three pass.
- **Reverse proxy path** — Nexul deploys Traefik on your chosen machine, with a Let's Encrypt certificate for the instance's domain, then points the instance's own DNS record at that server's address. Traefik needs ports 80 and 443 reachable from the internet to get the certificate.

### 4. Go live

Once the hostname resolves, this step confirms it and the instance is reachable at that address.

## Next step

Give an individual stack a hostname from its own page — see [Stacks and deploys](/docs/guide/stacks-and-deploys/).
