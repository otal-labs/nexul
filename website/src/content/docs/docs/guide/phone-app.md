---
title: Phone app
description: Install the Nexul app on your phone and sign in by scanning a code from the web app.
sidebar:
  order: 13
---

The web app is built for tablets and desktops. On a phone, use the Nexul app: your inbox, chat, your calendar, the board, deploys and their logs, and docs. It ships for Android, and for iPhone through SideStore. It isn't in an app store; you install it from GitHub.

## Install it

The app's releases are tagged `phone-v<version>-beta` on the [releases page](https://github.com/otal-labs/nexul/releases). Each one holds both files.

**Android:**

1. On the phone, download `nexul-android-<version>.apk` from the newest phone release. Opening the web app on an Android phone shows a **Get the Android app** link to the same page.
2. Open the file. Android asks you to allow installs from your browser the first time.
3. Install. A newer APK installs over the old one and keeps you signed in.

**iPhone:** download `nexul-ios-<version>.ipa` and install it with SideStore, which signs it with your own Apple ID. With a free Apple ID, SideStore needs to refresh the app every 7 days, and the iPhone app gets no push notifications.

## Sign in

The phone signs in with a code from a computer where you're already signed in. There's no password to type.

1. In the web app, open **Settings → Security → Devices** and press **Generate code** under **Connect a phone**.
2. In the app, press **Scan the code** and point the camera at the QR code. Your phone's own camera app works too.
3. The card in the web app shows your phone as connected.

The [Welcome wizard](/docs/guide/welcome-wizard/)'s **Get the phone app** step shows the same code, for anyone whose role gives that step.

The code works once, for two minutes. If it runs out, press **New code**. Without a camera, press **Enter it by hand** in the app and type the instance address and the 12-character code shown under the QR code.

The phone stays signed in for 90 days after you last used it. It shows under **Devices** with its model, in the web app and in the app's **More → Your settings → Devices**. Sign it out from either.

## What's on the phone

- **Inbox**: your notifications. **Mark all read** clears them; swipe one to the left to mark just that one read without opening it, or swipe a read one to put it back as unread. A screen reader gets the same as **Mark read** and **Mark unread** actions on the row.
- **Chat**: your channels, direct messages, and threads. Read and reply, with code blocks shown as scrolling code, open an agent's notes, and read a bot's posts with their embeds. When an agent hands work to another agent, tap the pill under its reply to read that agent's conversation: each stretch of work folds under how long it took and what it did, with a check, a warning when a step failed, or the spinner while it runs. Tap one to see its steps. A message to `@Agent` while it's busy goes into the turn it's working on; hold **Send** to queue it behind that turn instead. Hold one of your own messages to **Edit** or **Delete** it. Edit puts the text back in the composer, where **Save** sends the change and **Cancel** drops it. Delete asks first and is permanent: the message goes for everyone in the conversation. With a screen reader, the same two are actions on your message.
- **Board**: a project's tickets by status. Show only tickets where you're developer or tester, **Assign to me**, or **Open thread**. A role that can only read tickets sees no way to change a status or assign one; a role set to show locked actions sees them dimmed, and a tap says why. To move a ticket to another status, hold its card until it lifts and drop it on a status along the bottom, or open the ticket and pick one under **Status**.
- **Deploys**: your stacks and their deploys, with the deploy log and each service's live logs. A deploy that is waiting or running has **Cancel deploy** under its title, for whoever may deploy the stack; the confirm names the stack, and the runner stops the deploy where it is. **Redeploy** and **Cancel deploy** follow the same rule as the Board: hidden without the permission, or dimmed for a role set to show locked actions, where a tap says why. A stack that runs an image can be redeployed from here; one built from a compose file redeploys from the web.
- **More**: **Search**, the same workspace search as the web's command palette for the tickets, docs, pull requests, stacks and channels you can open, grouped by kind with tickets and docs that only mention your words under **In text** (a pull request opens its ticket; automations and memories are found on the web); the calendar, docs to read and, where a doc has [questions to answer](/docs/guide/docs-tickets-and-board/#clarify-a-doc), to answer, runners (the machines you are on), and your settings: appearance, devices, and which workspace the app shows. A ticket or doc mentioned in a doc or ticket opens when you tap it; one you can't open shows its title with a lock and doesn't open. The calendar lists the next 30 days of your meetings and the tickets due, by day and in the phone's own time zone. Open a meeting to see who is invited and to accept or decline; creating and changing meetings stays on the web. A meeting invite, cancellation or reminder in the inbox opens the meeting.

The Board, Deploys and docs show the projects you're in. Until a project lists you or one of your roles, the app has no Board or Deploys tab, and chat and the inbox work as usual.

While the app is open, everything on it updates as it changes. If the connection drops, the app catches up on what it missed when it reconnects, and again each time you bring it back to the front.

## Updates

Most updates arrive by themselves: the app checks when it starts and uses the new version from the next start. A bigger change ships as a new APK or IPA on the releases page; install it over the old one.

If your instance runs a Nexul version older than the app needs, the app says so and names the version. Ask whoever runs the instance to [upgrade](/docs/guide/upgrade/), then press **Retry**.
