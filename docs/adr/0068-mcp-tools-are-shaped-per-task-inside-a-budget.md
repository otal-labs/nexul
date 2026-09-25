# MCP tools are shaped per task, inside a budget

MCP tools no longer map one-to-one onto use-cases. A tool covers one task an
agent performs, and the server stays well under 100 tools. A new capability
extends an existing tool before it earns a new one.

At one tool per use-case the surface passed 180 tools. Every definition costs
context in every agent session, agents pick the wrong tool more often once a
list passes a few dozen, and at least one client caps an agent at 100 tools
across all its servers.
Most of the count was not capability. It was one-field setters (set a
ticket's status, set its type, move it to a category), list variants that
differed by one filter, and create, rename, reorder, and delete tools for
entities an agent edits as one task.

So one-field setters fold into patch-style updates, list variants into one
filtered list, child collections into their parent's get and update, and
reversible toggles into update fields, where the way back is as visible as
the way in. Three splits stay: reading and changing are separate tools,
because annotations are per tool and a mixed tool makes every read ask for
confirmation; delete is its own tool; starting an execution is its own tool.

The trade-offs:

- An update can call several use-cases, which commit separately. A
  composite update can apply part of a change before a later step fails, so
  it stops at the first failure and says which parts took effect.
- A patch-style update reads the current record and overlays the given
  fields, so an edit made between the read and the write can be lost.
- Parity with the web app (ADR 0019) now holds per capability, not per
  endpoint: every capability is reachable through some tool.
- Tool names and parameters changed wholesale once, so saved prompts that
  name the old tools stop working.
