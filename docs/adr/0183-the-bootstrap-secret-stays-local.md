# The bootstrap secret stays local, never a secret reference

`NEXUL_AUTH_SECRET` signs sessions, derives the at-rest key, and keys the GitHub webhook and the
enrollment tokens. Unset, it is generated into `auth-secret` beside the database. It may not be a
secret reference, though connector and sign-in secrets may.

Why: backups are database-only files, so they already lack it, and the server's own secrets-manager
token would sit on the same machine, so a reference would protect almost nothing more. It would also
make a secrets-manager outage stop the server from starting, since everything sealed depends on this
key. Connector and sign-in secrets resolve after start, and an outage degrades only them.

Decided: 2026-11-12. Spec: `.scratch/external-secrets/spec.md`.
