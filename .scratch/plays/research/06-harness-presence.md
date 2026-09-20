# 06 — Harness presence and pairing-resolution facts

Researched 2026-09-16, on master. Answers what a play button could know
about "is @Agent runnable for this user/project" before a click, and where
that falls short.

## 1. How a paired harness is judged connected

`presence.Keeper` (`internal/presence/keeper.go`) tracks refcounted browser
WebSocket connections per user, not harness liveness directly:

- `Connected`/`Disconnected` (keeper.go:98, keeper.go:119) bump a per-user
  refcount from `liveEventsHandler` (`server/cmd/routes.go:58-64`), which
  wraps the `/ws/events` socket (`server/cmd/routes.go:171`). The first open
  socket for a user triggers `reconcile`; the last close starts a
  `DefaultLinger` (10s, keeper.go:17) teardown timer so a page refresh
  doesn't flap.
- `reconcile` (keeper.go:151) lists the user's unexpired paired computers via
  `cfg.Sessions` (wired to `pairing.Service.ActiveSessions`,
  `server/cmd/services.go:174-178`) and starts one `maintain` goroutine per
  computer (keeper.go:189).
- `maintain` (keeper.go:194) is the actual connection loop, one per
  computer:
  - Looks up the computer's `harness.Client` from the `harness.Registry` by
    `computer.Kind` (keeper.go:202-206); if no client is registered for
    that kind, it logs and exits — the computer never becomes "connected".
  - If `computer.TokenExpiresAt` is not in the future, it returns
    immediately (keeper.go:209-211) — an expired token never dials.
  - Calls `client.Hold(ctx, computer.Session())` (keeper.go:213), which for
    T3 Code opens a real connection; `harness.Client.Hold` (harness.go:129)
    docs say a harness with nothing to hold returns a nil `Conn`, and per
    ADR 0054 the keeper then shows it connected until presence is torn down
    (keeper.go:227-230).
  - `apperrs.ErrUnauthorized` from `Hold` ends the loop for good (no more
    retries) with a warning log — re-pairing is the only way back in, via
    `OnComputersChanged` → `Refresh` (keeper.go:215-219,
    `server/cmd/services.go:169`). Any other dial error retries with
    exponential backoff (1s → 60s cap, keeper.go:20-22, 241-247).
  - State per computer is one of `StateConnecting` / `StateConnected`
    (keeper.go:61-63); a computer with no entry in the map means nothing is
    held — that covers "never dialed" and "gave up after 401" alike, no
    reason string survives either case.

**State held**: `Keeper.users[userID].computers[computerID] -> *loop{state}`
(keeper.go:41-57), in memory only, one entry per computer currently being
maintained for a user with a live browser socket.

**Published**: nothing onto the event bus or a live topic — no
`presence.*` catalog event exists. It is read on demand:

- `Keeper.Status(userID)` (keeper.go:66) returns `map[computerID]state`.
- `GET /api/pairing/presence` (`server/cmd/routes.go:43-56`, mounted at
  `server/cmd/routes.go:112`) returns `{"computers": {...}}` straight from
  `Status`.
- Web side: `useFetchPresence` (`web/src/hooks/PairingHooks.tsx:70-75`)
  polls that endpoint every 15s (comment there: "flips on browser
  connect/disconnect and T3 restarts, not on any user action here" — i.e.
  deliberately polled, not pushed).

## 2. `pairing.ResolveTarget` resolution order and reasons

`resolveTargetSource` (`internal/pairing/usecase.go:387-418`), in order:

1. **Project link** — if `projectID` is non-empty and
   `repo.GetProjectLink` returns a link with `ComputerID` set, use its
   `ComputerID`/`HarnessProjectID`/`Provider`/`Model` (`fromLink=true`,
   usecase.go:388-396). A project-linked computer is fetched with
   `GetComputerByID` regardless of owner (usecase.go:434-438,
   `internal/pairing/repo.go:21`) — it need not belong to the calling user.
2. **User defaults** — `repo.GetDefaults(userID)`; if
   `DefaultComputerID` is set, use it plus `FallbackProjectID`/
   `Provider`/`Model` (usecase.go:398-405).
3. **Sole computer** — if the user has exactly one paired computer, use it
   with no harness project id (usecase.go:407-417); zero computers ->
   `ReasonUnpaired`; more than one with no default -> `ReasonNoDefaultComputer`.

After a source resolves, `ResolveTarget` (usecase.go:356-384) additionally:
- Fetches the computer and checks token expiry (`fetchTargetComputer`,
  usecase.go:420-432): not found -> `ReasonUnpaired`; expired
  (`TokenExpiresAt.Before(now)`) -> `ReasonExpiredToken`.
- If `harnessProjectID` is still empty after step 1-3 -> `ReasonNoDefault`
  (usecase.go:373-375) — a valid, non-expired computer but no harness-side
  project configured for it.

Four `NotConfiguredReason` values (`internal/pairing/model.go:56-65`) map
1:1 onto `replyNotConfigured` (`internal/agent/pipeline.go:376-394`):

| Reason | Meaning | Message |
|---|---|---|
| `ReasonUnpaired` | no computer resolves at all | "@Agent needs a paired computer — connect one in Settings → Pairing." |
| `ReasonExpiredToken` | computer resolved, bearer session expired | "@Agent's paired computer's session has expired — re-pair it in Settings → Pairing." |
| `ReasonNoDefault` | valid computer, no harness project configured | "@Agent needs a project on the paired computer to run against — link one in this project's settings, or set a fallback in Settings → Pairing." |
| `ReasonNoDefaultComputer` | several paired computers, none default | "@Agent found several paired computers — pick a default one in Settings → Pairing." |

`ResolveTarget` is called from exactly one place in the whole backend:
`internal/agent/pipeline.go:211`, inside `runTurn`, which is already
mid-turn (it posts a system note into the conversation on failure,
pipeline.go:213). No read-only/no-side-effect caller exists
(`rg ResolveTarget internal --type=go` outside tests: usecase.go and
pipeline.go only).

## 3. What the web already has

Hooks in `web/src/hooks/PairingHooks.tsx`, all backed by
`internal/pairing/handler.go:22-34`:

- `useListComputers` (PairingHooks.tsx:23-27) → `GET /api/pairing/computers`
  → `Computer[]` incl. `kind`, `token_expires_at`, `harness_version`
  (`web/src/models/Pairing.tsx:4-13`).
- `useFetchPresence` (PairingHooks.tsx:70-75) → `GET /api/pairing/presence`
  → `{computerId: "connected"|"connecting"}`, polled 15s.
- `useFetchPairingDefaults` (PairingHooks.tsx:97-101) →
  `GET /api/pairing/defaults` → `PairingDefaults` (default computer,
  fallback project, provider, model).
- `useFetchProjectLink(projectId)` (PairingHooks.tsx:116-121) →
  `GET /api/pairing/projects/{id}` → raw `ProjectLink` row (computer id,
  harness project id, provider, model) — **not resolved**: it is whatever
  is stored for that project, with no fallback chain applied and no
  liveness/expiry check.

These are consumed today only by settings pages/components
(`web/src/components/settings/ComputersSection.tsx`,
`ComputerRow.tsx`, `ProjectLinkSection.tsx`, `ProjectLinkForm.tsx`,
`PairingDefaultsSection.tsx`) — nothing outside Settings queries pairing
state yet. A ticket page could reuse `useListComputers` +
`useFetchPresence` + `useFetchProjectLink` + `useFetchPairingDefaults`
without a new query, but would have to **re-implement
`resolveTargetSource`'s fallback order client-side** to turn those four
payloads into one verdict, and still couldn't detect `ReasonExpiredToken`
against a project-linked computer it doesn't own (`GetComputerByID` bypasses
`GetComputer(ctx, userID, ...)`'s ownership scoping — the web's
`useListComputers` only lists the calling user's own computers, so a
project-linked computer owned by someone else is invisible to it).

## 4. Gaps

1. **No read-only resolve endpoint.** `ResolveTarget`/`resolveTargetSource`
   compute exactly the answer the button needs (which computer, which
   reason) but are reachable only via `internal/agent/pipeline.go:211`,
   which is a live turn attempt with a system-note side effect on failure.
   Smallest addition: `GET /api/pairing/resolve?project_id=` returning
   either `{ok: true, computer_id, harness_project_id, provider, model}` or
   `{ok: false, reason: "unpaired"|"expired_token"|"no_default"|"no_default_computer"}`
   — a thin HTTP wrapper around `resolveTargetSource` (skip the bearer-token
   decrypt step, the button doesn't need the token). This single gap covers
   all four `NotConfiguredReason` states for the button.
2. **Resolved-computer ↔ presence linkage.** Even with gap 1 closed, "the
   resolved harness is offline right now" needs cross-referencing the
   resolve response's `computer_id` against `useFetchPresence`'s map — two
   calls, joined client-side. Not a backend gap once (1) ships, but worth
   naming: the resolve endpoint's shape should return `computer_id` even on
   the `ok: true` path specifically so the web can do that join.
3. **Harness-rejected-but-not-yet-expired has no reason.** `maintain`
   (keeper.go:216-219) stops retrying on `ErrUnauthorized` (e.g. the T3 side
   revoked the pairing) even though `TokenExpiresAt` is still in the
   future by Nexul's clock — the computer just silently drops out of the
   presence map with a server-side log line only. `ReasonExpiredToken`
   won't catch this (our clock says valid) and presence shows the same
   "missing entry" as "never dialed". No state distinguishes "expired" from
   "revoked" from "just hasn't connected yet" today. Smallest addition:
   have `maintain` set a third state (e.g. `StateRejected`) instead of
   deleting the loop entry on `ErrUnauthorized`, surfaced through the same
   `Status`/`/api/pairing/presence` map.
4. **"Paired but no harness project for this ticket's project" is already
   covered** by `ReasonNoDefault` once gap 1 ships — no separate gap beyond
   the missing endpoint.

No gap exists for "unpaired" or "several computers, no default" — both are
fully computed server-side today, just not exposed outside a live turn
attempt (gap 1).
