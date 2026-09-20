# Chat is a page with three panes, not a floating dock

Supersedes ADR 0051. Chat's one surface is the `/chat/:conversationId` page:
the app sidebar, a conversation list grouped into chats and threads, and the
open thread. Opening a conversation from anywhere (the sidebar's channel rows,
a doc's Thread button) navigates to that URL. The floating panel dock and its
launcher popover are gone.

ADR 0051 made the dock primary so the page behind it stayed live while an
`@Agent` turn changed it. In use, the dock was the surface people avoided: a
popover to find a conversation, then a panel too small to read a thread in,
stacked over whatever was on screen. The live-page argument did not hold in
practice either, because a turn's effects arrive as notifications and board
or doc updates that the next visit shows, and a ticket's thread already
renders on its ticket page. A page the user can bookmark, deep-link, and read
at full width is worth more than a panel that follows them around.

The list is grouped into two sections, chats (channels, voice channels,
direct messages) and threads (doc threads), because those are the two things
a person looks for: a place to talk, or the discussion attached to a
document. Selection lives in the URL, so a refresh, a shared link, and the
browser's back button all land on the same thread.

The cost is that an agent conversation is now a navigation away from the
board or doc it acts on. If that turns out to matter, the answer is a side
panel on those pages, not a return of the dock.
