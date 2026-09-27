# Each runner and automations host enrolls for its own credential

Replaces the shared runner secret and the automations host's tokens file.

Adding a runner or an automations host mints a one-time enrollment code (`nxe_…`), bound to a kind and a name, and
valid for one hour. The install command the instance renders carries it, and `nexul install runner` or
`nexul install automations` trades it for the host's own credential (`nxr_…` or `nxa_…`) before the service first
starts. The instance stores only hashes of both. The credential has no expiry; removing the host from the UI, MCP
(`host_delete`) or the machine (`nexul uninstall runner <name>`) revokes it. A connected host is told to uninstall
itself; an offline one is refused as removed when it reconnects and uninstalls itself then. Revoked credentials are
kept as tombstones, so a returning host is told it was removed rather than that its credential is unknown. The
bundled `instance` runner and automations host enroll the same way, with a code the server writes into its data
directory at boot while neither exists yet.

Why: with one instance secret shared by every runner, removing a runner in the UI did nothing. It re-registered on
its next connect, and revoking one machine meant rotating the secret on all of them.

Rejected: long-lived per-host tokens minted in the UI and pasted into the command. The command sits in shell
history and in the chat it was pasted through, so it must not hold anything that stays valid; a code that works
once and dies within the hour does not. Also rejected: deleting revoked credentials. A host whose credential simply
vanished cannot tell "removed" from "wrong instance" and would retry forever instead of uninstalling itself.

The trade-off: the credential file on the host is the whole identity. Copying it to another machine copies the
host, and losing it means removing the host and adding it again. A code is also useless to anyone who sees it after
it was used, which is the point, but it means a failed install needs a fresh code.

Decided: 2026-09-27
