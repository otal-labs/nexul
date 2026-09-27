# 07 — Research: Traefik forwarding to a host service with Let's Encrypt

**Type:** research
**Status:** ready-for-agent
**Blocked by:** None — can start immediately

## Question

How does the gateway Traefik (deployed by the runner as a container, docker provider, `exposedByDefault=false`) also (a) forward one domain to the Nexul server running on the host at `host.docker.internal:<port>` and (b) obtain and renew a Let's Encrypt certificate for it and for later exposures, using only what a run-strategy deploy can express (image, command, ports, mounts, env)? Compare: Traefik v3 labels on its own container (can a service point at a URL?), the file provider with a mounted dynamic config the runner writes, and ACME HTTP-01 storage (volume for `acme.json`). Answer from Traefik's official docs (Context7 or the docs site), with the exact flags.
