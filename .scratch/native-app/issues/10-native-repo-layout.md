# 10 — native/ layout, shared code, practices and CI

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

How is `native/` set up in this repo? Its package and bun setup, which code it shares from `web/src/models` and `sdk/` and how (Metro `watchFolders`, no barrels, no root package.json), whether it gets its own practices file and how `AGENTS.md` routes to it, which lint, typecheck and test gates run, and its CI job and `dorny/paths-filter` entry.

## Answer

Decided 2026-09-28.

- **`native/` is its own bun package**, like `web/`, `sdk/` and `desktop/`,
  with no root `package.json`. It uses Expo (the current stable SDK at build
  time, verified in the docs), Expo Router, TypeScript strict, and the New
  Architecture.
- **No shared source for the first cut.** The phone calls about a dozen
  endpoints, and `native/src/models/` declares their types itself. Reaching
  into `web/src` through Metro `watchFolders` would pull the web's `@/`
  aliases along. The additive-API rule keeps the two in step. Revisit if drift
  shows.
- **Styling:** Uniwind with react-native-reusables through the shadcn CLI,
  reading Mono Console tokens in the `native/` stylesheet (same names and
  values as `web/src/index.css`). Inter and JetBrains Mono load through
  `expo-font`.
- **State:** TanStack Query for server state, set up with React Native focus
  and online managers. zustand for client state, persisted through the
  expo-sqlite key-value store. The session token goes in `expo-secure-store`.
- **A new `practices/native.md`** routed from the `AGENTS.md` table. It covers
  the house rules that carry over (F1–F7 adapted, early return, no barrels,
  the logging rule) and the native specifics above.
- **Gates:** `bun run lint` (eslint-config-expo), `typecheck`, and `test`
  (jest-expo with React Native Testing Library). A `native` CI job behind a
  `dorny/paths-filter` entry. Building APKs lives in ticket 14, not in CI.
