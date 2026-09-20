---
title: Desktop App
description: A thin Electron client that connects to your instance with a connection token.
sidebar:
  order: 14
---

The Nexul desktop app is a thin Electron shell around the same web app your server already serves — it isn't a separate frontend with its own features, just a native window and process for reaching your instance. It's the reference standalone client for connection tokens; a CLI and the runner are candidates for the same bootstrap flow later.

## Connecting

The app doesn't ask for a username or a server address on first run. Instead:

1. In the web app, open **Settings → [Connection token](/docs/guide/api-and-tokens/)** and click **Generate connection token**. This produces a signed token carrying your instance's URL and basic settings — no identity or credentials, so it isn't secret. It expires 30 days after it's generated.
2. Paste the token into the desktop app's **Add instance** field and click **Import**.
3. The instance now shows in the app's list, with a live status (connecting, connected, unreachable, or expired) checked by probing `/api/auth/me` on that instance's origin.
4. Click **Connect**. The app loads your instance's web app in place; you sign in the normal way through GitHub OAuth. Nothing about sign-in is special-cased for the desktop app — it's the same login flow the browser uses.

You can import tokens for more than one instance and switch between them from the same instance list, and remove one without affecting the others.

## Building it

The app lives under `desktop/` as an Electron project (`nexul-desktop`) with its own `package.json`:

```sh
cd desktop
bun install
bun run build   # compiles the Electron main/preload bundle and the launcher UI
bun run start   # build, then launch electron .
```

For a distributable package:

```sh
bun run dist     # electron-builder: AppImage + deb on Linux, nsis on Windows, dmg on macOS
```
