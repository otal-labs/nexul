# 08 — The thread pane on the ticket page

**What to build:** The ticket page runs full width and its thread moves into
a left pane, as locked in [The thread column on the ticket page](01-thread-column-look.md).
Port variant D from the `proto/thread-column` branch (commits up to
f50401b5) without its variant switcher or seed scripts: `TicketPage` lifts
the centred width cap, the grid becomes thread pane, body, rail; the
thread pane is sticky and viewport tall with the composer at its foot; the
body text keeps a readable measure inside its card. The pane is
drag-resizable through the shared pane handle the prototype extracted from
the list pane (`PaneResizeHandle`, `ListPaneResizeHandle` as a thin
wrapper, `ThreadPaneResizeHandle`, `threadPaneStore` persisted as
`thread-pane`, `--thread-pane-width` registered non-inherited). Breakpoints
follow the page width beside the sidebar: pane from 736px, rail beside the
body from 1120px. The Inbox split view keeps the thread under the body.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] The thread pane starts at the sidebar and the rail sits flush right at 1440 and 1706px, sidebar open and collapsed; no centred side margins remain on the ticket page
- [ ] Dragging, arrow keys, Home/End, and double-click reset work; the width persists across reloads and is saved once per drag
- [ ] A drag holds about 58 fps under 6x CPU throttling on a long ticket (the prototype's numbers), measured again on the built code
- [ ] Below the breakpoints the thread, then the rail, drop under the body; checked at 768, 1024, and 1440px
- [ ] The Inbox split view still shows the thread under the body
- [ ] The list pane on Docs and Memories behaves exactly as before; its tests pass
