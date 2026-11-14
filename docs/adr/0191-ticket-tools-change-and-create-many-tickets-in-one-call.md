# Ticket tools change and create many tickets in one call

An agent asked to "assign every ticket in this sprint to me" or "split this doc into tickets" made one MCP call per
ticket: twenty-five `ticket_update` calls for a twenty-five-ticket sprint. The work is spread over the existing tools
rather than added as new ones, because the tool count is a budget (ADR 0068) and a batch is the same task on more rows.

- **`ticket_update` takes `ids`** instead of `id`, up to 100, one `ticket_list` page. Every ticket is resolved before
  any changes, so a mistyped key changes nothing. The same fields then apply to each ticket in turn, and the call
  stops at the first ticket that fails and names the tickets already changed, as a single update names the fields it
  applied. A field that only fits one ticket (`title`, `body`, `edits`, `position`, `revert_to_version`) is refused.
- **`ticket_create` takes `tickets`**, a list of titles and bodies sharing every other field (project, type, category,
  people, doc, origin), which is how a doc splits into tickets. Blank titles are refused before anything is created;
  the rest stops at the first failure, as `ids` does.
- **Finding the batch is one call.** `ticket_list` filters by `category_id`, `status_id` and `developer`, in SQL
  (ADR 0140). An empty string means none (uncategorized, unassigned), as it clears the field on `ticket_update`, so no
  category id or login is reserved as a sentinel.
- **`me` names the caller** in a ticket's developer and tester fields and in the developer filter, so an agent need not
  call `account_get` first. An automation token acts as no member, so it is refused.

A batch is not one transaction: each ticket's use-case commits on its own, so its events, history and audit row stay
exactly what a single call writes. Stopping at the first failure and naming what applied is the rule composite tools
already follow (`practices/mcp.md`, section 8). The advertised size budget rises by 500 characters for the batch
fields; repeated parameter text on the ticket tools was shortened to pay for the rest.

Decided: 2026-11-14.
