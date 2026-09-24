# Paired computers are reached through a tunnel on the instance's own Cloudflare

Nexul runs on a server; the harness it drives runs on each user's own
machine, usually behind a home or office router with no address the server
can reach. Pairing by URL only works when both sit on the same machine or
network. Each paired computer therefore runs `cloudflared` as a background
service with a tunnel Nexul creates in the instance's own Cloudflare account,
and gets a hostname of the computer's name plus eight random characters on
the instance's domain. A Cloudflare Access rule on that hostname admits only
requests carrying a service token that only the Nexul server holds, one token
for the whole instance, since Cloudflare caps service tokens per account.
Nexul then pairs with the harness over the hostname exactly as it pairs by
URL today, so the harness client barely changes. One tunnel routes one
hostname per local harness, so the model does not depend on T3 Code.

Pairing's first step installs the tunnel and waits until it is online and
the harness answers through it. URL pairing stays as an advanced option for
machines the server can already reach. The costs are accepted: an instance
must have Cloudflare connected, with Zero Trust enabled once by the owner,
before a laptop can be paired, and each computer has a public hostname,
closed to everything but the Nexul server.

## Considered Options

- **The harness's own hosted relay.** It admits only the harness's own apps
  as clients, with no documented path for another server.
- **A relay built into Nexul**: a small program on the computer holding one
  outbound WebSocket to the server, with streams multiplexed over it. Sound
  and free of Cloudflare, but new transport code and a new dependency to own.
  Kept for instances that ever need laptop pairing without Cloudflare.
- **A WireGuard data plane with its own relay server.** It solves both ends
  sitting behind routers, which Nexul does not have, at the cost of a young
  library with no wire-format promise, a heavy dependency tree, and a UDP
  port on every self-hoster's server. Kept for the day either end has no
  public address.
- **Asking users to join a private network with the server.** It works with
  today's code but makes every user a network administrator.
