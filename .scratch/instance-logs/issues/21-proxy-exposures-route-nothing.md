# 21: Proxy exposures write no Traefik router

Type: task
Status: resolved
Blocked by: None — can start immediately

## What to build

Found while building ticket 18: nothing in the repo writes `traefik.http.*` labels, and the proxy
gateway's Traefik runs with `exposedByDefault=false`. A proxy exposure creates its A record and
joins the networks, but Traefik has no router for it, so the hostname probably reaches nothing.
`ExposuresForService` and `ExposuresForContainer` are only read by one HTTP route.

- Confirm it live on the test box: expose a stack's service through a proxy gateway and request
  the hostname.
- If it routes nothing, label the container at deploy time (a labels field on the deploy frame
  and a compose override for compose stacks), with the router named `dns.RouterName(exposureID)`
  (`nexul-<exposure id>`), which request counts already match on.
- Then the proxy line in the stacks guide's Requests section drops its "for now" sentence.

## Surfaces

- Runner deploy frame and compose override, with tests.
- Docs: the topology and DNS guide, the stacks guide.

## Built

Confirmed live: a proxy hostname answered 404 with no router. The fix writes one router and
service per proxy exposure (`nexul-<exposure id>`) into the running gateway's file config through
the runner, on exposure create, delete and rename, when the gateway turns healthy, and when its
runner connects (ADR 0157). No redeploy of the exposed stack. On the test box the hostname went
from 404 to 200 and the router's request counter appeared.
