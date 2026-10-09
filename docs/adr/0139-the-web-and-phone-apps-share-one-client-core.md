# The web and phone apps share one client core

The phone app was written by porting the web's modules: the permission table, the chat message rules, the bot embed
rules and the live socket each existed twice, kept in step by hand under "Mirrors web/src/…" comments. They drifted.
The phone never got fenced code blocks, had no catch-up after its socket reconnected, closed a socket mid-handshake
differently, kept a sent message's row by its text instead of by the optimistic row it replaced, and retried queries
differently from the web.

Decision: code both apps run lives once, in `client-core/` at the repository root, and both import it as
`@nexul/client-core/<module>`, by full path like any other module.

- **What belongs there.** Pure modules with no React, no navigation, no network and no platform API: the wire shapes
  both apps read and the rules over them (permissions, chat, embeds, people, the reconnecting socket, the query retry
  rule). Components, query hooks, the phone's `defineQuery` declarations (ADR 0136) and the web's followers
  (ADR 0134) stay in each app, and so does anything only one app uses. A module moves here when the second app needs
  it, not before.
- **The adapter rule.** A shared module takes plain values where the platforms differ: the retry rule takes an HTTP
  status, which each app reads off its own error type, and the socket takes a URL and a socket factory. A seam with
  an adapter per app is added only when both apps supply a real one; when the socket is open (always while signed in
  on the web, only in the foreground on the phone) stays in each app's hook, because the web's side of such a seam
  would do nothing. A platform API that behaves differently is replaced by a portable rule rather than injected:
  `httpUrl` is one pattern, since React Native's `URL` accepts anything.
- **Wiring.** The folder has no `package.json`, dependencies or build. The web resolves it through a Vite alias and
  a tsconfig path, lets the dev server read it (`server.fs.allow`), runs its tests once under its own vitest and lints
  it under its own rules (`client-core/eslint.config.js` re-exports the web's config). The phone resolves it through
  the same tsconfig path, which Expo's Metro honours, with the folder in `watchFolders`, and in jest through
  `moduleNameMapper`, with `moduleDirectories` so Babel's helpers resolve from the app. Each app's typecheck compiles
  the modules it imports under its own settings, so a browser-only or React Native-only API fails the other app. The
  CI paths filter runs both apps' jobs when the folder changes.

Settled while moving the modules:

- After the socket reconnects, both apps refetch every open read, since frames sent while it was down are lost. A
  phone returning to the foreground is caught up by the focus manager instead, as ADR 0136 describes.
- Neither app retries a 4xx. The stale time stays per app: 30 seconds on the web, whose socket stays open while
  signed in, hidden tab included, and 0 on the phone, whose socket closes in the background.

The trade-offs: the folder has no test runner of its own, so its tests ride the web's job, and a change to it runs
both apps' CI. A shared module can only use what both runtimes provide.

Rejected: modules inside `sdk/`, whose `exports` map is the public surface automations are written against (chat
grouping would become SDK API) and which targets Bun; a workspace package with its own `package.json` and lockfile,
another install and runner for a folder with no dependency of its own; and keeping the copies with a sync check,
which keeps two copies to review.

Decided 2026-10-09.
