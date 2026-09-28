# 11 — What each phone page does

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

For each picked page, what can a person do on the phone? Reading only, or also acting: comment and change status on a ticket, send messages and images in chat, edit docs (through a WebView of the instance's own editor, or not at all), redeploy a stack. Which Home and Inbox items appear, and how the board list is grouped and filtered.

## Answer

Decided 2026-09-28.

**Tabs:** Inbox · Chat · Board · Deploys · More. More holds Docs, Runners and
Your settings. The web Home is a landing hero, so the phone opens on Inbox.

- **Inbox:** the notification list, newest first. Tapping an item marks it
  read and opens its subject (a ticket, doc or conversation). Mark all read.
- **Chat:** the conversation list, then a thread. Send text messages; images
  in a thread display but can't be sent yet. No reactions and no voice.
- **Board:** a project picker, then tickets grouped by status as sections, with
  a "Mine" filter. No drag.
- **Ticket:** the header, a rendered markdown body, status (changeable in a
  sheet), assignees ("Assign to me"), and a link to its thread in Chat.
- **Docs:** the list per project and a markdown reader. Editing stays
  web-only for now.
- **Deploys:** stacks with their status, then a stack's deploy history, the
  live log of a running deploy, and Redeploy behind a confirm. Services and
  containers are shown read-only.
- **Runners:** the list with status and version.
- **Your settings:** Profile (read-only), Appearance (system, light or dark),
  Devices (list and sign out), the workspace switcher, and Sign out.
- **One instance per install.** Signing out is how you switch instances.
- **No offline cache** in the first cut. When offline, a banner says so and
  queries retry.
