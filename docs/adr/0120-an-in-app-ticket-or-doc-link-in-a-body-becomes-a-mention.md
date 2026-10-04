# An in-app ticket or doc link in a body becomes a mention

A ticket or doc reached by URL instead of the @ picker showed as an underlined link beside the picker's live chips: a
link pasted into the editor, written as markdown, or sent through MCP. People expect the same pill either way.

Decision: the editor turns a link to a ticket page (`/<workspace>/tickets/<key or id>`) or a doc page
(`/<workspace>/docs/[<folder>/]<id>`) on this instance into a mention. It does this when the link is pasted or
autolinked, and when a body loads. Only links in the workspace being viewed convert; anything with a query or anchor
does not, and neither does any other page (a board, settings, a memory). A collaborator's change and an undo step are
never converted, so two collaborators never convert the same link twice. A body loaded from its saved text converts: a
read-only view, and the editor that seeds an empty live room, which then saves the mentions. An MCP or HTTP write
resets the room, so its links convert on the next open. Links already in a live room's history stay links in that
editor until the body is next written outside it. The server's markdown conversion is unchanged: it has no instance
URL or workspace to judge a link by, and every body reaches people through the editor.

Ticket URLs carry the key, and the editor cannot look up the ticket's id while converting, so the mention keeps the
key: `[label](/tickets/ERF-7)`. Mention resolution takes the workspace and resolves a key that names exactly one
ticket in it; over MCP, `mention_search` takes it as `workspace_id`. This amends ADR 0089, which said every stored
mention link holds an id that needs no workspace: a key mention needs its body's workspace, and it breaks if the
ticket moves to another project, as the URL it came from would.

Rejected: rewriting links on the server at save time, because the instance URL would have to reach every domain that
normalizes a body, and bodies saved from the editor never pass through that conversion. Rejected: converting links
only as they are rendered, without changing the body, because the pill would then exist only on screen while the
stored body, search, and MCP still saw a link.

Amends ADR 0089. Decided 2026-10-04.
