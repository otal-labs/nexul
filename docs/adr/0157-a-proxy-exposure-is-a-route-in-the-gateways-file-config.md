# A proxy exposure is a route in the gateway's file config, written live by the runner

A proxy gateway's Traefik runs with `exposedByDefault=false`, so it routes only containers whose labels name a router,
and nothing in Nexul labelled one. An exposure created its DNS record and joined the gateway to the container's network,
and the hostname still answered 404.

Decision: each proxy exposure becomes one router and one service in the gateway's file-provider config, named
`nexul-<exposure id>` (`dns.RouterName`), rule `Host(<hostname>)` on `websecure` with the default cert resolver, pointing
at `http://<container name>:<port>` on the shared network. The server builds the set of routes for a gateway from its
exposures and sends it in a `gateway_routes` frame; the runner reads the container's base config from its
`NEXUL_TRAEFIK_DYNAMIC` environment variable (the instance's own route, or nothing), merges the routes in and renames the
file into place, and Traefik's file watcher reloads it. The set is sent when an exposure is created or deleted, when a
container comes back under another name, when the gateway's own stack reports healthy (a recreated container starts with
only its base config), and when a runner connects, for every proxy gateway on its machine.

Why not Docker labels, the other candidate. Labels exist only at container creation, so every exposure change would
redeploy the exposed stack, and a compose stack would need an override file written into it. Creating an exposure for a
running stack would restart it, and the stack's own redeploys would have to rebuild labels from the exposures. The file
needs no redeploy of the stack and none of the gateway, which also carries the instance's own route and every other
hostname.

Why the base comes from the container's environment. The routes are rewritten whole each time, so the file must be the
base plus the routes and nothing the server has to remember. The environment variable is the one copy of the base that
every existing gateway already has, so gateways deployed before this need no new spec or redeploy to route. The server
never stores the instance domain to rebuild it.

The cost: the push is best-effort and not retried, so a runner that is connected but fails to write stays stale until the
next push; one that was offline is caught up when it reconnects. The config travels as one frame (read limit 1 MiB) and one command
argument (about 128 KiB), a ceiling of a few hundred exposures per gateway.

Counts: Traefik labels these routers `nexul-<id>@file`, and request counts already strip the provider suffix.

Decided: 2026-10-16
