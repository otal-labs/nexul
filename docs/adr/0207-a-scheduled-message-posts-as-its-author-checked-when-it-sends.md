# A scheduled message posts as its author, checked when it sends

Teams post the same kinds of messages ahead of time: the reminder the day before a meeting, written right after the
last one, or the weekly standup prompt. Meeting reminders don't cover them, because a reminder is a notification to
each invitee, never a message in the channel everyone reads.

Decision: a member may write a message now and have Nexul post it in a conversation later, once or on a repeat.

- A scheduled message is its own row (`scheduled_messages`, migration 0168), not a message with a future time: a message
  that exists is one every reader of the conversation sees, while a scheduled one is its author's alone until it posts.
  It goes with its conversation and with its author's account.
- It repeats daily, weekly or monthly, stepped in the timezone it was saved with, so a weekly 09:00 stays 09:00 across a
  clock change, and stepped from its first send, so a monthly one on the 31st skips the months without one. These are a
  meeting's repeat rules (ADR 0168), copied rather than shared: chat cannot import calendar, and a platform
  package for four cases was more than the copy. It has no end date; cancelling it stops it.
- Scheduling checks that the author may post there now. Sending checks it again, as the author, at the send time: a
  member who left, lost the channel or lost `channels:post` meanwhile has the message dropped, never posted with rights
  they no longer hold, and a dropped one is logged. Holding the post's rights at scheduling time only was rejected: it
  would let a removed member keep speaking in a channel.
- One loop sends what is due, woken by any write to the table and otherwise sleeping until the next send, at most an
  hour. A send moves or removes the schedule and inserts the message in one transaction that first checks the schedule
  is still as it was read, so an edit or cancel at the moment it sends posts nothing twice and nothing stale.
- A message missed while the server was down posts once when it comes back, late; a repeating one then skips to its
  first send still ahead, rather than posting every missed one.
- Only its author lists, changes, sends early or cancels it; anyone else reads it as not found.
  `chat.scheduled_message.changed` carries ids, never the text, reaches only the author's sockets, and is members-only,
  so no integration or automation receives it. The message it posts is an ordinary `chat.message.created`.
- Agents reach it through the tools that exist: `message_post` takes `schedule` (with `id` it changes, cancels or sends
  one early), and `message_list` takes `scheduled`. No new tool was added; the advertised `tools/list` budget rises from
  195,000 to 196,100 characters for the new fields.

Decided 2026-11-20.
