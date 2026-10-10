# 06 — Add a computer in the app; new computers pair only through the runner

**Status:** ready-for-agent

**Blocked by:** 03, 04

Read first: `practices/react-guide.md` (F1 to F7 and the self-review checklist), `practices/design-language.md`
(shared core, web app), `practices/testing.md`, the spec (The flow, Naming).

## What to build

- Your settings → T3 Code Setup → Computers: **Add a computer** replaces **Pair a computer**. The dialog
  shows the command (macOS / Linux and Windows tabs, Windows marked coming until ticket 09), then three
  live checks driven by `runner.personal_changed` and `computer.paired`: Computer connected, T3 Code found,
  Paired; then the existing Set up step.
- The computer row: each lane's health on its own (runner connected, T3 Code answering through it), Rename,
  Re-pair now, Remove; "Open T3 Code" when the desktop app is closed.
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
