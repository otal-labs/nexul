# 08: Where a stack's request counts come from

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

An error-rate rule needs, per stack, how many requests its hostnames served and how many
answered 5xx, every minute. Nexul holds no request data for deployed apps today: the proxy
gateway runs Traefik with neither metrics nor access logs on, and the tunnel gateway runs
cloudflared with its metrics on `:20241` for the health check only. Where do the counts come
from, for each gateway kind, without storing request logs?

Technical: answered in this ticket, not grilled with the owner.

## Answer

- **Proxy gateways: Traefik's Prometheus metrics, read by the machine runner.** `traefikSpec`
  (`internal/dns/gateway.go`) adds `TRAEFIK_METRICS_PROMETHEUS=true`,
  `TRAEFIK_METRICS_PROMETHEUS_ADDROUTERSLABELS=true`, a `metrics` entry point on `:8082` and
  `TRAEFIK_METRICS_PROMETHEUS_ENTRYPOINT=metrics`. The port is not published: the runner on the
  same Linux host reads `http://<container ip>:8082/metrics` on the gateway's network every 30
  seconds and sends the deltas of `traefik_router_requests_total{code,router}` in a new
  `request_counts` frame. Existing gateways pick it up on the next gateway redeploy, which the
  upgrade that ships this triggers.
- **Matching a router to a stack.** The server maps each router to the exposure that labelled it,
  and through the exposure to its stack and hostname. The router names the exposure labels set
  must be stable and carry the exposure id; the build ticket confirms where those labels are
  written (`ExposuresForService` has no caller outside `internal/dns` today) and fixes their
  names if they don't.
- **Tunnel gateways: Cloudflare's analytics, read by the server.** cloudflared counts requests and
  errors per tunnel only (`cloudflared_tunnel_total_requests`, `cloudflared_tunnel_request_errors`,
  no hostname label), and one tunnel serves many hostnames. Cloudflare's GraphQL API answers per
  hostname: `httpRequestsAdaptiveGroups` filtered by `clientRequestHTTPHost`, grouped by
  `edgeResponseStatus`, on every plan with 31 days kept. The server queries it once a minute for
  the tunnel hostnames of stacks that have a rule turned on, one query per zone, with the
  connector's existing token. It reads a window that ends two minutes ago, because the dataset
  arrives a little late.
- **The token needs Analytics Read.** The Cloudflare connector's checks gain `analytics_read`
  beside `dns_edit` and `zone_read`, and the token instructions add "Zone · Analytics · Read". A
  token without it doesn't break anything else: a tunnel stack's rule is greyed out with "Your
  Cloudflare token can't read analytics; add Zone Analytics Read to it". No new env var and no
  new token.
- **Nothing per request is kept.** Both sources arrive as counts. The server keeps a rolling
  per-stack, per-minute table (`stack_request_minutes`: stack, minute, requests, errors_5xx),
  pruned after a day. It is enough to evaluate a rule and draw the stack page's small chart, and
  ADR 0091's rule against stored output holds.
- **Gaps say so.** A minute where the runner was disconnected or the API failed is stored as
  missing, not zero. A rule never fires or recovers on missing minutes. The stack page shows "no
  request data since 10:42" after five missing minutes.
