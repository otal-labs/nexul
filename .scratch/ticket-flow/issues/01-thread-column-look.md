# 01: The thread column on the ticket page

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

The thread moves out from under the body into a left column beside it on
wide screens, and drops back under the body when the screen is narrower
(decided while charting). What does that column look like and how does it
behave?

- Its width, and the screen width where it moves under the body.
- Its height: sticky and the full viewport tall, so the composer stays in
  reach while the body scrolls, or the height of the body.
- Where the composer sits, and how the empty state and "Start chat" read in
  a narrow column.
- The Inbox shows a ticket in a split view; whether the column survives
  there or the thread stays under the body.
- How the plays bottom bar and the trail sit next to it.

Prototype the candidates live on the ticket page at 768, 1024, and 1440px;
the owner picks by looking.

## Answer

Variant D on the `proto/thread-column` branch, picked by the owner over
four other candidates on the live page.

- The ticket page drops its centred width cap and runs full width with the
  normal page gutter: the thread pane starts at the sidebar, the body fills
  the middle with its text held to a readable measure inside the card, and
  the properties rail sits flush right at 18rem.
- The thread pane is sticky and the full viewport tall, composer at its
  foot. Its width is drag-resizable like the Docs and Memories list pane
  (handle on its right edge, keyboard steps, double-click reset, saved once
  on release) and was put through the same frame-budget treatment as the
  list pane drag; the measurements are in the branch's last commit.
- Breakpoints follow the page width beside the sidebar, not the screen:
  the thread pane from 736px of page width (a 1024px screen with the
  sidebar open), the rail beside the body from 1120px; below those the
  thread and then the rail drop under the body.
- The Inbox split view keeps the thread under the body.
