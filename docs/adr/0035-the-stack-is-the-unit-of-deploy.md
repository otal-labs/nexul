# The stack is the unit of deploy; the containers inside it are observed, not deployed

A deploy record used to point at one "service definition", which forced a
compose file with five containers to be either five definitions Nexul
had no way to keep in step, or one definition that lied about what was
running. Splitting it the way OpenShip does — desired state on the **stack**
(one repository's worth of containers: a compose file, or a Dockerfile as a
stack of one), observed facts on one **service** row per container, refreshed
from the runner's `docker inspect` report after every deploy — makes the
compose file the only source of truth for wiring while still giving the
canvas, exposures, and status badges a real per-container record to anchor to.

Deploy, rollback, history, branch rules, and the one-active-deploy guard are
therefore all per stack; a service has no deploy button of its own, and its
row is read-only. A service row is never deleted by an observation that
misses it, only marked `stopped`, so a failed deploy cannot erase what the
canvas is drawing.

A stack's name is always the owner's, and the slug derived from it is what
`docker compose -p` and `docker run --name` use, so a second stack with the
same slug on the same machine is rejected at creation rather than given a
random or numbered suffix. Generated names were considered and rejected:
every other part of the system addresses a container by that name — a tunnel
ingress origin, a Traefik label, a compose bind mount, one container reaching
another over a docker network — and a name Nexul invented is a name
nobody can write down in advance. The same holds for a branch deployment's
`<base>-<branch slug>`: a slug collision with an unrelated stack is a clear
error, never a silent rename.

Decided: 2026-09-08
