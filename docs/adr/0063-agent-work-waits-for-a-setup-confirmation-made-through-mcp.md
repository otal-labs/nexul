# Agent work waits for a setup confirmation that only an agent can make, through MCP

A paired computer is only as good as what is installed on it: without the
Nexul MCP server connected to each provider and the default skill set
installed, plays and `@Agent` mentions fail or wander, which is why the loop
worked on the owner's machine and not on anyone else's. Setup is therefore
recorded as a confirmation per computer and per provider, false by default,
and a provider cannot run agent work on a computer until both the
computer's and the provider's confirmation are set. The check sits where
every agent run resolves its computer and provider, so plays and mentions
share it, and the one run that bypasses it is the setup turn the setup
wizard starts.

Only an agent can set or withdraw a confirmation, and only through MCP tools;
the web UI shows the state but has no control that changes it. The point of
the confirmation is that an agent assessed the machine: the skill files are
where each provider looks, the harness reports them among the skills it has
discovered, and the provider reached Nexul over MCP to say so. Nothing
withdraws a confirmation on its own, not a skills release, a re-pair, or a
harness update; a different computer, or a provider appearing for the first
time, starts unconfirmed. The confirmation is an honour system: the server
cannot prove which machine made the call, and the agent's assessment is the
trust anchor.

This is deliberately the one hard gate in the lifecycle. Elsewhere Nexul
prefers a visible signal to a block, but a computer without the setup cannot
run the loop at all, so letting it try only produces a worse failure later.
