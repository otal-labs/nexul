# First run sets up the domain before GitHub, behind a one-time setup code

A GitHub App's callback must be the instance's final https address, and GitHub sign-in only works when it starts
from that same origin (the OAuth state cookie is per-origin and Secure once the address is https). So first run
no longer asks for the GitHub App on `http://<ip>:<port>`. It asks for a setup code, then gives the instance its
domain (Cloudflare tunnel, reverse proxy with Let's Encrypt, or the owner's own HTTPS), stores that as the instance
URL, and hands off to the domain, where the GitHub App is connected and the owner signs in. The domain step has no
skip. Desktop installs stay on `http://localhost` and skip it.

The installer defaults the web port to 5123 so the reverse proxy can own 80 and 443.

**Setup code and setup pass.** Taking a Cloudflare token and creating DNS records before anyone has signed in
cannot be open to whoever reaches the port first, which is how the old bootstrap page worked. While no user
exists, the server writes a fresh code to `<data>/enroll/setup` on every boot (only its hash is stored), and the
installer and `nexul status` print it. `POST /api/setup/unlock` trades the code for a signed, hour-long setup pass
that `RequireAuth` accepts only while no user exists, and only on an allowlist of the routes first run needs. The
GitHub bootstrap requires the pass too. Every pass stops working once the first user exists.

**Fresh names resolve through a public resolver.** The route checks and the instance-URL check look at a hostname
created seconds earlier. The host's resolver caches a "no such name" answer from the first lookup (Cloudflare zones
hold it for 30 minutes), which would keep the check red long after the record exists. `internal/platform/freshdns`
resolves through 1.1.1.1 first and falls back to the host's resolver for names only a private DNS knows.

**Traefik routes the instance through its file provider.** The Nexul server is a host service, not a container,
so Docker labels cannot describe it and the runner cannot set labels anyway. The proxy gateway's container writes
a dynamic config from an env var to Traefik's file provider on start, with the instance's domain routed to
`host.docker.internal:<port>` (ADR 0076). Every router on the https entry point defaults to the Let's Encrypt
resolver (HTTP-01 on port 80, account and certificates in the `nexul-traefik-acme` volume).

Rejected: keeping the GitHub App first and adding a domain step inside the owner wizard, which leaves the owner
registering a `localhost` or IP callback and re-registering it later. Also rejected: first visitor wins for the
domain step, since it now handles a Cloudflare token. Also rejected: Cloudflare OAuth for the token, because its
callback cannot exist before the domain does.
