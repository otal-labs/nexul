---
title: Desktop app
description: Open your instance in its own window with the desktop app, connected by a connection token.
sidebar:
  order: 14
---

The desktop app opens your instance's web app in a window of its own. It has no features of its own and holds no account: you sign in the same way you do in a browser.

## Connecting

1. In the web app, open **Settings → Security → Devices** and click **Copy connection token** on **Connect the desktop app**. The token holds your instance's address, not your account, and expires after 30 days.
2. In the desktop app, paste it under **Add instance** and click **Import**.
3. The instance appears under **Instances** with its status: **Connected**, **Unreachable** or **Token expired**.
4. Click **Connect** and sign in.

Import a token from each instance you use to switch between them. **Remove** drops one without touching the others. When a token expires, copy a new one and import it again.

## Building it

There is no prebuilt download yet. Build it from the `desktop/` folder of the repository:

```sh
cd desktop
bun install
bun run start   # build and launch
bun run dist    # package it: AppImage and deb on Linux, an installer on Windows, a dmg on macOS
```
