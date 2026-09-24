# 13 — The setup wizard's look

**Type:** prototype
**Status:** resolved
**Blocked by:** 06, 15

## Question

What do the pair-a-computer dialog and the setup wizard look and feel like —
the tunnel step (per-OS command, waiting for the connector, verified), T3
Code pairing over the verified hostname, the entry from a computer's row, the
pre-selection step, the running setup turn, and the confirmed/failed end
states — at 320, 375, 414, and 768px? Run through design-mode: references,
the owner picks, lock it. Reuse the existing wizard family (instance setup,
Add runner) and the Mono Console; ticket 06 decides what the pairing row and
refusals show, so this one follows it.

## Answer

Design session 2026-09-24. The owner delegated the pick ("I trust you in
terms of design") and set the component rule: codedvisuals first, then
shadcncraft, then shadcn/ui. The three locked references are saved outside
the repo in the owner's design-reference log, with notes.

**The look.** One dark dialog with segmented step tabs across the top —
**Connect → Pair T3 Code → Set up** — that becomes a full-screen sheet at
320–414px, where the tabs shrink to "Step 2 of 3 · Pair T3 Code".

- **Connect (tunnel)**: numbered install commands on the left in
  copyable terminal blocks with per-OS tabs; on the right a live connection
  panel, *this computer ↔ Nexul*, idle and "Waiting for connection…" until
  both checks pass, then connected; the two checks listed under it
  ("Tunnel online", "T3 Code answering"). An alert card inside the step
  explains a missing prerequisite with its fix (Cloudflare not connected,
  Zero Trust not enabled). Next stays disabled until connected. On mobile
  the panel sits above the commands.
- **Pair T3 Code**: the existing pairing fields with the verified hostname
  pre-filled, inline errors.
- **Set up**: the coarse pre-selection (both skill locations ticked), then
  one live row per provider — spinner while its setup turn runs, check and
  dimmed when confirmed, a failed state with Retry — with the agent's short
  commentary collapsible under the active row, never louder than the list.
- **Entry point**: the computer's row in pairing settings keeps the settings
  row pattern — setup badge, one line per provider, Set up / Re-run setup
  opening this dialog.

**Components, in the owner's source order.**

| Piece | Source | Notes |
|---|---|---|
| Connection panel | codedvisuals `connections-sync` | Drive idle vs connected from live state; pulse only once connected |
| The two checks | codedvisuals `status-health-check` | `items` from tunnel status and the T3 probe |
| Install commands | codedvisuals `code-terminal` | Static lines (`animated={false}`), copy button; matches the terminal motif |
| Setup runs | codedvisuals `ai-tools` | One row per provider; status from setup-turn events, not its demo timeline |
| Dialog frame, step tabs | shadcncraft `modal-1`, `progress-3` | **Pro items; no licence key is configured.** With a key, add the registry header; without one, fall back per the rule |
| Fallback frame | shadcn/ui `dialog`, `drawer`, `tabs`, `checkbox` | Dialog on desktop, drawer/sheet on mobile |

codedvisuals components install as owned source files (React + Motion) and
all take data through props. Re-skin every one in Mono Console tokens:
`gradient`, `glow`, `particles`, and `isometric` off; monochrome except
status colour; reduced motion honoured. Register the codedvisuals registry
in `web/components.json` with the `${CODEDVISUALS_TOKEN}` placeholder when
building. Motion tuning goes through animation-mode once the look is built.
