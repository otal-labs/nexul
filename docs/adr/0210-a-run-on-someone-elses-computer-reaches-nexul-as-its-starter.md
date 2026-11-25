# A run on someone else's computer reaches Nexul as its starter

A computer's owner may share it with Run agents (ADR 0148), so another person's plays run in its T3 Code. The agent
there reaches Nexul's MCP server through the provider's own config, which setup filled with the owner's token, so it
would act as the owner. T3 Code takes no MCP server or credential on a thread or a turn, only on a provider instance.

Decision: a run whose starter is not the computer's owner runs on a provider instance of its own.

- At the start, Nexul re-checks the starter's Run agents, mints a token as the starter that expires in 12 hours, and
  adds a T3 provider instance through `server.updateSettings`: a copy of the picked provider's instance (or its built-in
  one) with the token in the instance environment as `NEXUL_GUEST_TOKEN`, marked sensitive, and launch arguments that
  point its `nexul` MCP server at that variable. Claude gets `--strict-mcp-config --mcp-config` with the server reading
  `${NEXUL_GUEST_TOKEN}`, so none of the owner's own MCP servers load; Codex gets `mcp_servers.nexul` with
  `bearer_token_env_var` and its saved header cleared. The token never rides an argument or a config file.
- The turn starts on that instance. When the turn ends the instance is removed and the token revoked; a server that
  stops first leaves the instance until its token expires.
- Nexul's T3 session needs `providers:manage`, which pairing now asks for; a computer paired before it is told to pair
  again. Only protocol-2 T3 Code, and only Claude and Codex, run someone else's agent; others refuse with the reason.
- The run is placed only on a computer picked for it; shared computers stay out of a person's order (amended by ADR
  0211: one may join it). Revoking Run
  agents interrupts the starter's running turns there, and the next placement is refused.
- Everywhere the run appears it says "Run by <starter> using <owner>'s <computer>". The owner's computer lists the
  runs others ran on it, who, which play and how it ended, never what they did.

Kept, and said in the sharing dialog: the agent runs as the owner's OS user, so it can still read the owner's own Nexul
token in the providers' config files. Closing that would mean every run, the owner's included, carrying a per-run
token instead of setup's, which can come later on the same instance mechanism.

Rejected: shipping Run agents on the owner's token; waiting for T3 Code to take a per-thread MCP credential.

Decided: 2026-11-25.
