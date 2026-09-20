---
title: Topology and DNS
description: The canvas that models your infrastructure, and how traffic reaches it from the internet.
sidebar:
  order: 7
---

## The topology canvas

Topology is the map of your infrastructure, drawn as a canvas — every service, the docker network it runs on, and how traffic reaches it. The canvas is a map, not a controller: service nodes, networks, gateway routes and hostnames are derived from what is actually deployed and cannot be edited on the canvas, and drawing an edge documents a relation without wiring anything. Only the hand-drawn nodes and positions are saved.

Open it from **Topology** in the sidebar. The canvas is built from a few node kinds:

- **Service nodes** — one per container, grouped inside the docker network it runs on.
- **Gateway nodes** — the hub every route into your infrastructure passes through (see below).
- **Hostname nodes** — the public hostnames pointed at a gateway.
- **External nodes** — things outside Nexul's management that a service still talks to.

Each node's accent and status badge reflect what the runner last observed — healthy, running, stopped, or failed.

## Gateways

A **gateway** is a Nexul-deployed service — a Cloudflare tunnel or a reverse proxy — that gives one docker network reachability from the internet. On the canvas, a gateway node is a hub: a hostname pill wires into it on the left, one row per exposure names where that traffic lands (`service:port`, with the container's observed address alongside it once known), and a wire goes out to the matching service node on the right.

Manage gateways directly from Settings → DNS → Gateways, or let the DNS setup stepper create your first one for you.

## Exposures

An **exposure** routes one hostname through a gateway to one of a stack's containers. From a stack's Exposures section, pick a container and a port; Nexul resolves or provisions the gateway for you, so you never pick one directly — you only choose what should be reachable, and at what hostname.

## Setting up DNS

DNS setup asks a stepped series of questions, either from the owner wizard's optional last step or from Settings → DNS if you skipped it. It needs the Cloudflare connector connected first (see [GitHub App](/docs/guide/github-app/) for connecting the equivalent GitHub connector — Cloudflare connects with an API token the same way, from Settings → Connectors).

### 1. Choose an entry path

| Path | What it does |
| --- | --- |
| Bare public address | Points your hostname straight at the server with an A or AAAA record. |
| Cloudflare tunnel | No open ports — `cloudflared` runs as a service on a runner and routes the hostname through it. |
| Reverse proxy | Deploys Traefik as a service on a runner, which routes hostnames to containers. |

Nexul deploys your chosen entry path itself, like any other service — there's nothing to install on the server by hand.

### 2. Deploy the tunnel (tunnel path only)

Name the tunnel, pick the project and runner it deploys to, and the docker network it should reach. Nexul creates the tunnel at Cloudflare and runs `cloudflared` on that runner until it connects.

### 3. Point your hostname

- **Bare path** — pick a Cloudflare zone and record type, and give the target address.
- **Tunnel path** — pick the zone and subdomain; Nexul routes it into the tunnel you just deployed and creates its DNS record.
- **Reverse proxy path** — Nexul deploys Traefik on your chosen runner, then points the instance's own DNS record at that server's address.

### 4. Go live

Once the hostname resolves, this step confirms it and the instance is reachable at that address.

## Next step

Give an individual stack a hostname from its own page — see [Stacks and deploys](/docs/guide/stacks-and-deploys/).
