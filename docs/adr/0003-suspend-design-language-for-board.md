# Suspend the Midnight Console spec for Board, pending a later unification pass

The Board redesign (`.scratch/board-redesign/`) hit a wall: the token-bound
Midnight Console spec (now the [Mono Console spec](https://nexul.io/docs/contributing/coding-standards/#design-language--the-mono-console)) was constraining the
visual exploration rather than helping it, and the owner didn't like what it
was producing. Rather than iterate inside a spec that isn't working, Board
components (`web/src/components/board/` and Board-specific pages/settings)
are freed to ignore `design-language.md`'s tokens entirely and explore
independently — visual inconsistency with the rest of the app during this
period is accepted, not a bug. Every other domain still follows the spec
unchanged. Once the owner is happy with Board's look, a follow-up pass
reconciles it into one color scheme across the app — tracked as a standing
reminder in `ROADMAP.md`, the same way the pre-release checklist tracks its
items, so it isn't lost between sessions.

**Resolved (2026-08-18):** the owner was happy with Board's black-and-white
look, and the Design unification effort (`.scratch/design-unification/`)
adopted it as the app's monochrome spec of record in
the [Mono Console spec](https://nexul.io/docs/contributing/coding-standards/#design-language--the-mono-console), flipped the shared `:root`/`.dark` tokens to
match it exactly, then retired Board's `.board-bw` override entirely — Board
now renders off the shared tokens like every other domain. This suspension is
lifted; the standing `ROADMAP.md` reminder has been removed per its own
instruction.
