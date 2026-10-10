# 06 — Add a computer in the app; new computers pair only through the runner

**Status:** resolved

**Blocked by:** 03, 04

Read first: `practices/react-guide.md` (F1 to F7 and the self-review checklist), `practices/design-language.md`
(shared core, web app), `practices/testing.md`, the spec (The flow, Naming).

## What to build

- Rename the settings page "T3 Code Setup" to **Computers** (owner, 2026-10-10): the nav label
  (`YourSettingsNav.tsx`), the panel's label, the setup-card references, and every message that sends a
  person to "Settings → T3 Code Setup" (`internal/pairing/model.go`, `internal/agent/pipeline.go` and their
  tests, `web/src/models/Pairing.tsx`), plus `CONTEXT.md`, `practices/design-language.md` and the guide pages.
  Check with `rg 'T3 Code Setup'`. The page keeps its sections (Computers, Projects, Defaults).
  "Personal runner" stays out of the UI.
- Your settings → Computers: **Add a computer** replaces **Pair a computer**. The dialog shows the command
  (macOS / Linux and Windows tabs, macOS and Windows marked coming until tickets 08 and 09; the Linux command
  is `curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`, and the dialog says it needs sudo and which
  account it installs for), then three live checks driven by `runner.personal_changed` and `computer.paired`:
  Computer connected, T3 Code found, Paired; then the existing Set up step.
- The computer row: each lane's health on its own (runner connected, T3 Code answering through it), Rename,
  Re-pair now, Remove; "Open T3 Code" when the desktop app is closed.
- A computer whose runner was revoked (its owner's account disabled, then reactivated; ticket 04) keeps its row and shows
  **Add this computer again**, which opens the dialog with a fresh command for that row
  (`POST /api/pairing/computers/enrollments {id}`).
- The tunnel step, the pairing-link step and Pair by URL are no longer reachable for a new computer. They
  stay only on old computers' rows, for re-pairing, until ticket 14.
- The owner wizard's last step uses the same dialog; its subtitle drops "from anywhere" and the tunnel.
- `paired-computers.md` rewritten for the new flow; `setup-wizard.md` step 4; ROADMAP.md and the roadmap
  page list the effort.

## Acceptance criteria

- [x] A person with no Cloudflare connector adds and pairs a computer end to end on nexul-box.
- [x] Screenshots at 768, 1024 and 1440px of each dialog state and the row, in the PR.
- [x] Component tests for each check's states, following F1 to F7; no `useEffect` for the live checks.
- [x] The guide pages match the UI word for word on button and step names.
- [x] `rg 'T3 Code Setup'` finds nothing in `web/src`, `internal`, `CONTEXT.md`, `practices` and the guide
      (accepted ADRs keep their old wording).

## Comments

Built: `AddComputerDialog` (command step `AddComputerStep`, three checks `ComputerChecks` over a shared `CheckRow`, then
the existing Set up step), `RunnerComputerRow` for computers added with their app (`ComputerItem` picks it, tunnel and URL
computers keep `ComputerRow`), `RenameComputerForm`, `hooks/ComputerHooks.tsx` (`useEnrollComputer`, `usePairNow`,
`useRenameComputer`), `models/ComputerChecks.tsx` (`computerChecks`, `computerLanes`, `addedWithRunner`). The web
`Computer` model gains `runner` and `pair_error`. No migration.

- Two server changes. A failed pairing through the runner publishes `computer.pair_failed` (ephemeral, members-only,
  ids only, `ownFrame`), so the dialog's Paired check turns Failed with the reason without waiting for a facts change.
  `POST /api/pairing/computers/enrollments {id}` for a computer reached through its runner drops its stored session:
  the runner that left ended Nexul's T3 Code sessions, so without this a re-added computer read paired and never
  re-paired (its session ran 23 more days). A tunnel computer keeps its session, for ticket 12's adoption.
- The page is Computers; the tab, the card ("Your computers"), the header meta ("N computers") and every message that
  sent people to "Settings → T3 Code Setup" or "Settings → Pairing" follow. "Pair a computer in Settings to run plays."
  reads "Add a computer in Settings to run plays."
- New computers reach only Add a computer: `NameComputerForm`, `TunnelPrerequisiteAlert`, the tunnel-create hook and
  schema, and Pair by URL in the dialog are deleted; a tunnel computer still pairing resumes at its Tunnel step from its
  row, and the old row's Re-pair keeps the link form. The routes and MCP tools stay for ticket 14. A runner computer's
  row offers Re-pair now (`POST .../pair`, no body) and never the link form, so the relay `server_url` is never prefilled.
- The UI calls the personal runner "the Nexul app", as the facts row already did.
- The dialog's macOS / Linux tab shows the one command (macOS since #565, its note says T3 Code on a Mac answers only
  while the person is logged in); the Windows tab says Coming soon and renders no command until ticket 09.
- Box check (nexul-box-addpc from `nexul-box/clean`, v0.3.99-beta.9 built from this branch, no Cloudflare connector):
  the owner pressed Add a computer in the browser, the command was copied from the dialog and run as `alice` (a sudo
  user, password typed at sudo's prompt), which installed T3 Code 0.0.45 and the runner; within 25 seconds of the run
  starting the three checks read Connected, Found (port 3773) and Paired, and Set up opened with alice's providers
  listed through the relay. No model turns were run. Screenshots in the PR.
- For 12: `forgetRunnerSession` only clears a computer whose `server_url` is the relay address; adoption of a tunnel
  computer must keep that. For 13: the owner wizard's last step needs no Cloudflare now; its copy says so. For 14: the
  remaining tunnel UI is `ConnectStep`, `PairT3CodeStep`, `PairT3CodeForm`, `PairingLinkField`, `PairCommands`,
  `TunnelChecks`, `ComputerRow`'s link Re-pair and `PairingStepTabs`.
