# 25 — Phone sign-in by QR, the version gate, and the API client

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 22, 24
**Decided in:** tickets 06, 09, 11

## What to build

The first-run screen: "Scan the code in Nexul → Settings → Security →
Devices" with a camera scanner (expo-camera) that accepts
`nexul://connect?host=…&code=…`, plus "Enter it by hand" (instance address and
code). The deep link `nexul://connect` opens the same flow from the system
camera. Flow: `GET <host>/api/about` (not Nexul → a clear error); compare its
version with the app's `MIN_SERVER_VERSION` constant (semantic precedence,
numeric pre-release parts, `dev` always passes); too old → the refusal screen
from ticket 09 with Retry, before the code is spent; else `POST
/api/auth/connect-codes/exchange` with `{code, device: {model, os,
app_version}}` (expo-device, expo-application) and store host plus token in
expo-secure-store.

The API client: base URL is the stored host, bearer token, a 401 signs the
app out (clears secure store and query cache) back to the first-run screen.
The version gate runs again on launch and on foreground; a signed-in user
behind a refusal keeps the session. A live-events WebSocket client
(`/ws?token=`) that connects in the foreground and closes in the background,
fanning events into query invalidation.

## Acceptance criteria

- [ ] Scanning a real code from the web Devices card signs the phone in and the web card flips to connected
- [ ] A server older than `MIN_SERVER_VERSION` shows the refusal before the code is used, and Retry re-checks
- [ ] A wrong or expired code shows one clear error; the sixth wrong try shows the rate-limit message
- [ ] A 401 from any call returns to first-run
- [ ] Version comparison has table-driven tests covering betas, releases and dev

## Surfaces

- UI (native)
- Uses HTTP from tickets 18 and 22

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, `practices/borrowed-practices.md`, tickets 06, 09 and 18's answer.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git. Plus a live run against a local server and the web Devices card.

## Files likely touched

- `native/src/app/` (first-run, scanner, refusal)
- `native/src/api/`, `native/src/stores/`, `native/src/models/`

**Size:** L
