---
title: Phone app
description: Install the Nexul app on your phone and sign in by scanning a code from the web app.
sidebar:
  order: 13
---

The web app is built for tablets and desktops. On a phone, use the Nexul app: your inbox, chat, the board, deploys and their logs, and docs. It ships for Android, and for iPhone through SideStore. It isn't in an app store; you install it from GitHub.

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

The code works once, for two minutes. If it runs out, press **New code**. Without a camera, press **Enter it by hand** in the app and type the instance address and the 12-character code shown under the QR code.

The phone stays signed in for 90 days after you last used it. It shows under **Devices** with its model, in the web app and in the app's **More → Your settings → Devices**. Sign it out from either.

## What's on the phone

- **Inbox**: your notifications. **Mark all read** clears them; swipe one to the left to mark just that one read without opening it.
- **Chat**: your channels, direct messages, and threads. Read and reply, open an agent's notes, and read a bot's posts with their embeds.
- **Board**: a project's tickets by status. Show only tickets where you're developer or tester, **Assign to me**, or **Open thread**. To move a ticket to another status, hold its card until it lifts and drop it on a status along the bottom, or open the ticket and pick one under **Status**.
- **Deploys**: your stacks and their deploys, with the deploy log and each service's live logs. A stack that runs an image can be redeployed from here; one built from a compose file redeploys from the web.
- **More**: docs to read, runners (if you can see them), and your settings: appearance, devices, and which workspace the app shows.

## Updates

Most updates arrive by themselves: the app checks when it starts and uses the new version from the next start. A bigger change ships as a new APK or IPA on the releases page; install it over the old one.

If your instance runs a Nexul version older than the app needs, the app says so and names the version. Ask whoever runs the instance to [upgrade](/docs/guide/upgrade/), then press **Retry**.
