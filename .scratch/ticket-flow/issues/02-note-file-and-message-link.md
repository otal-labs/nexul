# 02: Where a note's file lives and how its message points at it

Type: grilling
Status: open
Blocked by: None — can start immediately

## Question

A note's `.md` file is an attachment owned by the ticket (decided while
charting), and the note's message sits in the ticket's thread. How does the
message point at the file, and what happens when either side goes away?

- The link: a markdown line in the message body, the way chat images are
  referenced today, or a field on the message.
- Deleting the file from the attachments rail: the message keeps a "file
  removed" pill, or the message goes too.
- Deleting the message: the file stays on the ticket, or goes with it.
- The events a note publishes (catalog rows, outbox writes) and the live
  push that makes the pill appear without a refresh.
- Who can open the file: anyone who can read the ticket, the same as other
  ticket attachments.

Technical: arrives as a decided answer for a yes or no.
