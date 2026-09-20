# An integration is an external service with a scoped, revocable token, never in-process code

A third-party integration runs as its own container or binary on the owner's
infrastructure. Nexul pushes events to it as HMAC-signed webhook
deliveries and it calls back through the same HTTP gateway the browser uses,
authenticated by a token scoped to a least-privilege subset of the permission
grid. Running third-party code in-process would put it next to the secrets and
the single SQLite writer, and Go has no sandbox worth trusting; a separate
process is a boundary the operating system enforces for free. The price is
that "install" means "run this companion service", which is acceptable for an
audience already running Docker.

Tokens are revocable and revocation is immediate — it kills both API access
and webhook fan-out — because that is the only lever an owner has over code
they did not write.

Trust is a two-value tier (`verified` or `community`) shown at install rather
than a numeric rating: ratings need volume to mean anything, and a score of
"4.8 from three installs" is worse than no score at all.

Decided: 2026-07-25.
