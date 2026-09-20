# SQLite is the spine — no Postgres, no dual-backend abstraction

Nexul is one self-hosted binary plus a file, and the MCP server reads
from the same process it writes in, so a networked database would add an
operational dependency and a round trip for nothing; SQLite also ships FTS5,
which is the whole of v1 search. The second half of the decision is the
load-bearing one: there is no storage abstraction that could accept Postgres
later. SQL dialects diverge exactly where this product lives — JSON,
full-text, vectors — so a dual backend would either half-support both or
flatten the schema to the intersection of the two.

Decided: 2026-07-24

## Consequences

Every query may use SQLite's dialect directly. Scale-out is more instances,
not a bigger database — see ADR 0013 for why the spine also keeps the domains
in one process.
