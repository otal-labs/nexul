# 24 — The native/ app scaffold, practices, and CI

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** tickets 10, 12

## What to build

Create `native/` as its own bun package: the current stable Expo SDK (check
the Expo changelog and Context7 before pinning), Expo Router, TypeScript
strict, New Architecture, `expo-updates` configured with the `appVersion`
runtime policy and `updates.url` read from app config (filled by ticket 33).
Android package id `io.nexul.app`, app name "Nexul", scheme `nexul`.

Styling: Uniwind plus react-native-reusables installed through the shadcn CLI;
Mono Console tokens (names and values from `web/src/index.css`, dark and
light) in the native stylesheet; Inter and JetBrains Mono through `expo-font`.
Dark by default, following the system appearance.

Shell: Expo Router tabs Inbox, Chat, Board, Deploys, More (lucide icons),
each a placeholder screen with the right title; a stack per tab for detail
screens. TanStack Query client with React Native focus and online managers;
a zustand persist storage adapter over the expo-sqlite key-value store; an
offline banner.

Write `practices/native.md` (the stack above, which web rules carry over and
how, testing with jest-expo and React Native Testing Library, how to run on
an emulator) and add a row for `native/` to the `AGENTS.md` navigation table
and the enforcement table. Add a `native` job to `.github/workflows/ci.yml`
behind a `dorny/paths-filter` entry: install, lint, typecheck, test.

## Acceptance criteria

- [ ] `bun install`, `bun run lint`, `bun run typecheck`, `bun run test` pass in `native/`
- [ ] `bunx expo prebuild --platform android` then `./gradlew assembleDebug` produces an APK that opens to the tab shell on an emulator
- [ ] Tokens render the Mono Console in dark and light
- [ ] The CI job runs only when `native/` changes

## Surfaces

- Docs: `practices/native.md`, `AGENTS.md`
- CI: new job and path filter

## Read first

`AGENTS.md`, `practices/react-guide.md`, `practices/typescript.md`, `practices/design-language.md`, `practices/testing.md`, tickets 10 and 12, and `research/react-native-best-practices.md`.

## Verification

The acceptance commands above, plus a screenshot of the shell on an emulator.

## Files likely touched

- `native/` (new)
- `practices/native.md` (new), `AGENTS.md`
- `.github/workflows/ci.yml`

**Size:** L
