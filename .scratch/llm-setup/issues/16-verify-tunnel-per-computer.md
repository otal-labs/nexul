# 16 — Verify the tunnel-per-computer path

**Type:** research
**Status:** claimed
**Blocked by:** 15

## Question

The connection decision rests on three facts. Confirm each against the code
and current Cloudflare and T3 Code sources:

1. Nexul's existing Cloudflare tunnel code (`internal/dns`) can create a
   remotely managed tunnel, fetch its connector token, route a hostname to a
   port on a machine that is **not** a docker runner (a laptop running
   `cloudflared` as a service), report connector status, and delete it all
   again. What changes are needed?
2. Cloudflare Access applications, policies, and service tokens can be
   created and deleted through the API with a scoped API token, on a free
   Cloudflare plan; which token permissions that adds to what Nexul's
   Cloudflare connector asks for today.
3. T3 Code's pairing exchange and its WebSockets work through a Cloudflare
   Tunnel hostname with Access service-token headers added, and whether
   `t3client` can add those headers through its custom HTTP client.

Also: the per-OS commands to install `cloudflared` as a service with a
tunnel token (Linux, macOS, Windows), and how connector status reads over
the API so the dialog can wait for "online".

Findings to `research/tunnel-per-computer.md`; if a fact fails, reopen
ticket 15.
