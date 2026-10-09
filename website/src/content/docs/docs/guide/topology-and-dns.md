---
title: Topology and DNS
description: See how your services connect, and give them hostnames through a Cloudflare tunnel or a reverse proxy.
sidebar:
  order: 7
---

## The topology canvas

Open **Topology** in the sidebar to see your workspace's services: each container inside its Docker network, the gateways that route traffic in, and the hostnames pointed at them. Each node's status badge shows what the runner last saw: healthy, running, stopped or failed. Click a service to open its stack.

The canvas follows what is deployed, so you can't edit services, networks or routes on it. You can add your own nodes, such as an outside API a service calls, move things around, and draw edges to note a relation. Drawing an edge wires nothing. Each workspace keeps its own layout and drawings.

## Gateways

A gateway is a Cloudflare tunnel or a reverse proxy that Nexul deploys to give one Docker network a way in from the internet. On the canvas it is a hub: hostnames wire in on the left, one row per route shows where traffic lands (`service:port`, plus the container's address once known), and wires go out to the services on the right.

You rarely create one yourself: exposing a container or running **Set up DNS** creates the gateway it needs. To see or delete gateways, open **Settings → DNS**; each lists the hostnames it routes. A gateway belongs to the instance rather than a project, and shows on the topology of every workspace it routes a hostname into.

## Exposures

An exposure routes one hostname to one container.

1. Open the stack and go to **Exposures**.
2. Click **Expose**, pick the **Container**, type the **Hostname** and **Container port**, and pick the **Zone**.
3. Click **Expose**.

Nexul picks or deploys the gateway, adds the DNS record, and the hostname shows on the canvas. **Unexpose** removes it. Exposing needs the Cloudflare connector.

## Setting up DNS

The instance gets its domain in the [setup wizard](/docs/guide/setup-wizard/). To change how it is reached later, open **Settings → DNS** and click **Set up DNS**. Connect Cloudflare under **Settings → Connectors** first.

1. **How should traffic reach this instance?** Pick one:
   - **Bare public address**: an A or AAAA record pointing straight at the server.
   - **Cloudflare tunnel**: no open ports. cloudflared runs on a machine of yours and Cloudflare issues the certificate.
   - **Reverse proxy**: Traefik on ports 80 and 443 of a machine, with Let's Encrypt certificates. Both ports must be open to the internet.
2. **Deploy the tunnel** (tunnel only): name it and pick the machine and Docker network. If your token reaches several Cloudflare accounts, pick the one that owns the domain; a tunnel only serves its own account's zones.
3. **Point your hostname**: pick the zone and subdomain (or, for a bare address, the record type and target). For a tunnel, leave the local service empty to send the hostname to this instance. Three checks then confirm the route, the proxied DNS record, and that the hostname answers over HTTPS. A new record can take a minute.
4. **Go live** confirms the instance answers at its new address.

Nexul deploys the tunnel or proxy itself, like any other stack. There is nothing to install by hand.

### Cloudflare API token permissions

Create the token in the Cloudflare dashboard with:

| Permission | What Nexul uses it for |
| --- | --- |
| Zone → Zone: Read | Listing your zones and the account each belongs to |
| Zone → DNS: Edit | Creating the records that point hostnames at Nexul |
| Account → Cloudflare Tunnel: Edit | Creating tunnels |

Pairing computers by tunnel needs two more:

| Permission | What Nexul uses it for |
| --- | --- |
| Account → Access: Apps and Policies: Edit | Letting only this instance reach each paired computer |
| Account → Access: Service Tokens: Edit | The token this instance presents to get through |

These two also need Zero Trust turned on for the account. In the Cloudflare dashboard, pick a team name and the Free plan. Cloudflare asks for payment details but doesn't charge for Free.

When you connect the token, Nexul checks each permission and names any that are missing. It also lists the domains the token can edit; make sure yours is among them. A missing pairing permission is only a warning.
