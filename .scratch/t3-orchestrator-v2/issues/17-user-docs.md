# 17 — Document protocol 2 for people pairing computers

**What to build:** `website/` paired-computers guide: stable and nightly T3 both work; a computer moves
to the new orchestrator on its own and cannot go back; provider minimums (Codex 0.159, Claude Code
2.1.280, Cursor CLI built 2026-05-09 or later, Grok 1.0.13, OpenCode 2.0.18, Pi 0.99 for setup);
protocol-2 steps show what ran but not tool output; handed-off agents appear as pills. Computer-setup
guide: Pi and Grok steps, Antigravity unsupported for setup. `CONTEXT.md`: Trail (tool results may be
absent) and a hand-off term. Move the roadmap item to shipped in `ROADMAP.md` and
`website/src/pages/roadmap.astro`.

**Blocked by:** 14, 15, 16

**Status:** ready-for-agent

Read first: the spec; `website/` is mobile first.

- [ ] Website builds; pages checked at 320, 375, 414 and 768 px
- [ ] No internal doc codes on public pages; `rg --hidden` scan clean
