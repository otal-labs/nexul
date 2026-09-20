# Chat is a floating dock over a live page, and the chat page is the same components at full width

Superseded by ADR 0060: chat is a page with three panes, and the dock is gone.

Chat's primary surface is a floating thread dock, openable from
anywhere, that sits over whatever page the user is already on. A dedicated chat
page exists as well, but it is the same components at full width rather than a
second implementation, and a ticket thread renders in place on the ticket page.

The dock is the primary surface because of what `@Agent` does. A turn runs
against the product through MCP and its effects land in the UI — a ticket
appears on the board, a doc gains a section — and the point of a dock is that
the page behind it stays live while that happens, so the user watches the
result arrive instead of asking chat to describe it. A full-page chat would
have made every agent conversation a context switch away from the thing being
changed, which is the opposite of the product's premise. That reasoning does
not apply to browsing channels and DMs, which is why the full-width page still
exists for that.

The cost is that both surfaces have to keep working: panels stay mounted and
only change position and visibility, so a conversation never double-mounts as
it moves between them, and below `sm` the dock collapses to one full-screen
conversation because a floating panel over a phone-width page is neither.

The structural lock is placement and behaviour, not the reference's palette — this is
executed in Mono Console tokens. The borrow and its screenshots are recorded in
the [Mono Console spec](https://nexul.io/docs/contributing/coding-standards/#design-language--the-mono-console).
