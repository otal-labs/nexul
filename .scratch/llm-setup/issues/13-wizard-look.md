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
terms of design") and set the rule that components come from the paid
registries first, then the primitives. The locked references and the exact
component map live outside the repo, in the owner's design-reference log and
project memory.

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

**Components.** Each piece is sourced from the locked component map (kept
outside the repo): the connection panel, the two checks, the terminal
command block, and the per-provider run rows are animated registry
components installed as owned source and driven by live state; the dialog
frame and step tabs are rebuilt from a stepped-modal and a horizontal-stepper
block on the existing `dialog`, `drawer` (mobile sheet), and `tabs`
primitives; the pre-selection uses `checkbox`. Re-skin everything in Mono
Console tokens — no gradients, glow, particles, or isometric views,
monochrome except status colour, reduced motion honoured. Motion tuning goes
through animation-mode once the look is built.
