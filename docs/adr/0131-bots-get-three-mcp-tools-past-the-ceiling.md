# Bots get three MCP tools, and the ceiling rises to 111

Bots (ADR 0129) are managed from a channel's settings in the browser, but an agent had no way to list, add, rename, or
retire one, so a person asking an agent to wire a CI job into a channel could only be told to click through the
settings. The server stood at 107 tools under the ceiling of 108 that ADR 0118 set.

Folding came first and found no home. The only neighbour is chat, and carrying bots there would make chat learn what a
webhook is: its URL, its token, its regenerate and restore, and its own permission domain. Ticket 02 of the bots effort
drew the boundary the other way on purpose, with `internal/botwebhook` reaching chat through a one-method poster seam
and chat never importing it. Reads and changes stay apart (ADR 0068), so list is its own tool; create stays apart from
update because an update is a patch on an existing bot and a create needs a conversation.

Decision: three tools in `internal/botwebhook/mcp.go` over the same use-cases as the browser.

- `botwebhook_list` is read-only and returns a bot's `url` only to a caller holding `botwebhook:write`, since the URL
  is the bot's credential; the use-case decides, so a read-only agent sees no URL, as in the browser.
- `botwebhook_create` returns the new bot with its URL.
- `botwebhook_update` is a patch carrying name, avatar (empty clears), `regenerate`, and `deleted` true or false, so
  delete and restore are one field and the way back is as visible as the way in. Delete needs `botwebhook:delete` and
  restore needs `botwebhook:write`, both checked in the use-case.

An avatar travels as a base64 `data:image/` URL, the form a person's avatar already has. No tool posts as a bot: a
sender uses the URL like any other client. The ceiling in `internal/mcp/surface_test.go` becomes 111, one above the
110 tools the server now has.

This is the one exception to `practices/mcp.md`'s rule that a result carries no credential-bearing URL: the browser
already shows the same URL to the same permission holders, and an agent that creates a bot has to hand the URL to a
sender, so a separate reveal tool would add a fourth definition to say what the third already does.

The trade-off: three more definitions in every session, and a client that caps an agent at 100 tools across all its
servers drops three more of Nexul's. Accepted over leaving bots to the browser alone (ADR 0019).

Decided 2026-10-07, amending ADR 0118's tool ceiling.
