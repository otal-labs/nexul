# 06 — Add a computer in the app; new computers pair only through the runner

**Status:** ready-for-agent

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

- [ ] A person with no Cloudflare connector adds and pairs a computer end to end on nexul-box.
- [ ] Screenshots at 768, 1024 and 1440px of each dialog state and the row, in the PR.
- [ ] Component tests for each check's states, following F1 to F7; no `useEffect` for the live checks.
- [ ] The guide pages match the UI word for word on button and step names.
- [ ] `rg 'T3 Code Setup'` finds nothing in `web/src`, `internal`, `CONTEXT.md`, `practices` and the guide
      (accepted ADRs keep their old wording).
