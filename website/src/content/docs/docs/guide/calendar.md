---
title: Calendar
description: Plan meetings in a workspace, invite people, and get a reminder before each one and the day before a ticket is due.
sidebar:
  order: 8
---

Each workspace has a calendar. Open **Calendar** in the sidebar to see your meetings and the tickets due that month or week. Switch between **Month** and **Week**, and use the arrows or **Today** to move. Times show in your own timezone.

A meeting shows only to its organiser and the people invited, the same way a [private channel](/docs/guide/chat-and-voice/#private-channels) does. Nobody else in the workspace sees it, not even on the calendar.

## Plan a meeting

Press **New meeting** and fill in:

- **Title**, the day, and when it starts and ends.
- **Repeats**: not at all, daily, weekly or monthly, with an optional last date. A weekly meeting stays at the same time when the clocks change. A monthly one on the 31st skips the months that have no 31st.
- **Where**: one of the workspace's [voice channels](/docs/guide/chat-and-voice/#voice-channels), or a link to a call somewhere else. A private voice channel can only be picked when everyone invited is in it.
- **Invitees**: any members of the workspace. Each one gets a notification that they were invited.
- **Reminder**: how long before each meeting the reminder goes, 10 minutes unless you pick another time.
- **Notes**, in markdown.

Creating meetings takes **Create meetings** (`calendar:write`). Every role starts without it except the Owner, so an admin turns it on for the roles that need it on the [Access page](/docs/guide/people-and-access/). Without it the button is hidden, or shown locked with the reason for a role set to show locked actions.

## Reply, change, cancel

Select a meeting to open it beside the calendar. A month cell with more than three entries ends in **N more**, which opens that week. It shows where it takes place, the notes, and who is coming, who declined and who hasn't replied yet.

- **Accept** or **Decline** if you're invited. Your reply covers every occurrence of a repeating meeting.
- **Edit** changes a meeting. Its organiser can change it, and so can anyone holding **Change or cancel anyone's meetings** (`calendar:delete`).
- **Cancel** tells everyone invited. A cancelled meeting stays on the calendar, crossed out, until someone restores it with **Restore**.

For a repeating meeting, an edit or a cancel asks whether it's for **This meeting** or **This and following meetings**. Changing only this one leaves the rest as they were. Changing the following ones ends the series there and carries on with your change from that date.

## Reminders

Everyone who hasn't declined gets a reminder before each meeting, in the [inbox](/docs/guide/docs-tickets-and-board/#inbox) and as a phone push. Selecting it opens the meeting. There are no email reminders.

A ticket with a due date shows on the calendar on that day. Its developer gets a reminder at 9:00 the day before, in their own timezone. Nexul learns your timezone from the web or phone app the next time you open it, and uses UTC until then. Set a due date on the ticket page; see [Docs, tickets and the board](/docs/guide/docs-tickets-and-board/#tickets).

## On the phone

The [phone app](/docs/guide/phone-app/) lists your meetings and due tickets for the next 30 days, day by day. Open a meeting to see its details and accept or decline it. Planning and editing meetings stays on the web.

## For agents

An agent plans meetings with `meeting_create`, reads the calendar with `meeting_list` and changes, cancels or replies to a meeting with `meeting_update`, within your own permissions. A due date is a field on `ticket_update`. See [MCP server](/docs/guide/mcp-server/).
