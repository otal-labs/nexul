# A deploy swaps a routed service by renaming containers

A deploy ran `docker compose up -d` or `docker rm -f` then `docker run`, so a service behind a gateway dropped requests
for as long as its new container took to boot. A gateway reaches a service by its container name through Docker's
embedded DNS: a proxy gateway's Traefik routes and a tunnel gateway's Cloudflare ingress both say `http://<name>:<port>`.

Decision: the server names, on each deploy request, the stack's services some exposure routes to (`swap_services`). For
each one that changed (a compose container whose config hash and image already match is left to `up`, which leaves it
alone), can run twice (no host port, random ones included, no fixed `container_name`, one container running) and has a
healthcheck, the runner starts a second container beside it: `<slug>-next` for a run stack, a second replica through
`compose up --no-deps --no-recreate --scale <service>=2` for compose. It waits for that container's healthcheck, renames
the old container to `<name>-old` and the new one to the old name, then stops the old one. A new container that turns
unhealthy, stops, or is not healthy in five minutes is removed and fails the deploy, and the old one never stopped.
Compose's later `up` leaves the renamed container alone, since it matches the current config. Each container's
observation report says whether it has a healthcheck (`services.healthcheck`), so the stack page says which services a
deploy swaps.

Rewriting the gateway's route to the new container was rejected: a tunnel's ingress lives in Cloudflare and changes in
seconds, not milliseconds, through the server, and the routes and exposures would then name a container that is gone
after the next deploy. Giving the new container the old name as a network alias from its start was rejected: Docker's
DNS answers with every container holding a name, so the gateway would send requests to it before it is healthy, and an
alias cannot be added to a running container without disconnecting it. The cost of renaming is a window of about one
`docker rename` (some 20ms) in which the name resolves to nothing, so a new connection opened then fails once;
connections already open stay on the old container. A compose container's first swap has no window, because compose
gives each container its own name as an alias, which the old one keeps through the rename.

Decided 2026-11-19.
