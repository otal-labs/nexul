# One use-case layer, exactly two adapters: the browser never speaks JSON-RPC

Amended by ADR 0068: MCP tools are shaped per task, so a UI capability must be
reachable through some tool, not through a matching one.

Every domain's behaviour lives in its use-case functions, and exactly two
adapters call them: an HTTP/JSON gateway for the browser and an MCP server for
LLMs. Neither adapter holds business logic and neither is the "real" one —
that symmetry is the MCP-first thesis made concrete, because a capability that
exists in the UI exists in MCP by construction rather than by remembering to
add it. The browser never speaks JSON-RPC and the MCP server never serves
HTML.

Decided: 2026-07-25

## Consequences

A feature is not done when its HTTP route works; it needs the matching MCP
tool. Third-party integrations are not a third adapter — they call the same
HTTP gateway with a scoped token instead of a session (see the integrations
ADRs).
