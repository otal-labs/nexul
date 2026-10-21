# Nexul does not pass T3 Code's own interface through to a browser

A computer on a network that only lets out port 443 cannot use T3 Connect, whose tunnel needs outbound port 7844 to
Cloudflare. Nexul's runner reaches T3 Code over its own outbound connection on 443, so the question was whether Nexul
should also open T3 Code's own interface (its threads, diffs, terminal and files) for such a computer.

Decision: no. Everything Nexul does with a computer already works over 443 through the runner: chat with T3 threads
from the web and phone apps, plays, setup, MCP, the project picker and cloning. That covers what is needed away from
the computer. T3 Code's own interface stays reachable only through T3's own routes (T3 Connect, a LAN or VPN, Tailscale).

Rejected:
- Reverse-proxying T3 Code under a Nexul route. T3's web app expects the origin root and a host-derived cookie, so it
  needs a subdomain and certificate per computer, plus a per-person T3 session and a new permission. That is weeks of
  work and a new attack surface on the instance, for a need chat already meets.
- A `nexul connect` port forward to localhost on the person's laptop. It is smaller, but laptop-only and still a new
  command and permission.
- Exposing T3 Code at a public Nexul hostname for T3's own apps, which would put a shell-running API behind a pairing
  token alone.
- Building diffs, files or a terminal into Nexul. This waits until someone names a piece they miss.

The guide's "When the network only allows port 443" section on paired computers says what works and what doesn't.
Decided 2026-10-21.
