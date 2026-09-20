# A gateway is a service Nexul deploys, and it reaches a stack by joining that stack's network

Reachability from the internet — a Cloudflare tunnel or a Traefik reverse
proxy — is the one thing a self-hosted platform is tempted to bundle into its
own compose stack as a special container. Both kinds are ordinary Nexul
services instead, provisioned through the same seam and deployed by the same
runner as anything else: nothing to install on a server by hand, and the
deploy pipeline is dogfooded by the feature people hit first.

A gateway is not pinned to one docker network. It homes on the network it was
provisioned onto (at most one gateway per network as its home), and joins any
further network with `docker network connect` — the one mechanism that never
rewrites the target's compose file. That is what
lets a compose stack, whose networks come from its own file rather than from
a Nexul setting, be exposed at all; the alternative — requiring the
container's network to equal the gateway's — made every compose stack
unexposable. Tunnel ingress origins and proxy labels therefore key on the
container name, which stays unique once a gateway sits across several stacks.

Decided: 2026-09-10 (the network-joining model; gateways-as-services 2026-08-27)
