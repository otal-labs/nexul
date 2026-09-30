# Chat's conversations live in the app sidebar, and the chat page is the open conversation

Supersedes ADR 0060. The app sidebar lists every conversation a person browses
to, grouped as Channels, Voice channels, Direct messages, and Threads (doc
threads, shown once one exists). The `/chat/:conversationId` page renders only
the open conversation at full width; a bare `/chat` opens the first channel.
The conversation list pane is gone.

ADR 0060 kept the sidebar's channel rows and added a list pane beside the
thread, so every conversation appeared twice on the chat page, and the list
took a column the thread needed. One list had to go. The sidebar's stays
because it is reachable from every page, where the pane only existed on
`/chat`; the pane's search box went with it, since a workspace's conversation
list is short enough to scan.

Clicking a voice channel joins the call and opens its text chat in the same
click. The separate "open text chat" button existed only because the pane's
voice row joined without navigating.

The cost: with the sidebar collapsed to its rail there is no conversation list
on screen, so switching conversations means expanding it. The Chat rail icon
still opens chat and carries the unread total.
