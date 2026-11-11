# Secrets are resolved where they are used, with credentials that never reach Nexul

A secret Nexul uses can be a secret reference to an external secrets manager (1Password first,
`op://vault/item/field`) instead of a stored value. The side that uses the secret resolves it with its
own credential: a runner resolves its stacks' environment, an automations host resolves its
automations' pool, and the server resolves its own connector and sign-in provider secrets.

- **A runner's or host's token lives only on its machine,** scoped to the vaults it needs and set with
  `nexul runner 1password <name>` or `nexul automations 1password <host> --workspace <slug>`. Nexul
  never holds it and never sees a resolved value. The server's own token sits in its `.env`, scoped to
  a vault holding only Nexul's credentials.
- **Why:** a breach of Nexul (its database, a backup, the server) yields no way into a vault. Letting
  the server resolve everything and push values to runners was rejected. It is simpler, but one
  server credential would reach every vault, and every value would pass through Nexul.
- **The trust that stays:** anyone with `stacks:write` can aim a reference at anything the runner's
  token reads and deploy a container that prints it, as with stack environment today. Scoping each
  machine's token to its own vaults is the protection.
- **Consequences:** a deploy with references goes only to a runner that reported a ready token. A
  failed resolution fails the deploy before anything changes. References are resolved on every
  deploy in one batch, with no cache.

Decided: 2026-11-12. User guide: `website/src/content/docs/docs/guide/external-secrets.md`.
