# 01 — How each T3 Code provider discovers skills

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

On a machine running T3 Code, where does each provider (Claude Code, Codex,
opencode — plus any other driver T3 Code ships) look for agent skills, and
does one install location cover several of them?

Specifically:

- Claude Code: `~/.claude/skills/` and plugins — confirm against current docs.
- Codex: what it reads (`~/.agents/skills/`? `AGENTS.md` only? its own dir?).
- opencode: the owner suspects it scans both `~/.claude/skills/` and
  `~/.agents/skills/` natively — confirm or refute ("for opencode, it will be
  the same thing, or perhaps not?" is the owner's open question).
- Whether T3 Code itself adds a layer (its own skills directory or config)
  on top of the driver's discovery.

The answer decides whether "skills installed" is genuinely a per-provider
boolean or whether one install flips several providers at once, which shapes
the data model (ticket 04) and the install steps (ticket 05).

## Answer

Full findings: [research/skill-discovery-per-provider.md](../research/skill-discovery-per-provider.md)

- T3 Code ships six drivers (Claude, Codex, OpenCode, Cursor, Grok,
  Antigravity); only Claude, Codex, and OpenCode discover skills. T3 Code
  adds no layer of its own — it republishes what each CLI's native discovery
  reports, so installing for the CLI is sufficient.
- Claude Code reads `~/.claude/skills/` + project `.claude/skills/` +
  plugins. Codex reads `~/.agents/skills/` + repo `.agents/skills/` +
  `/etc/codex/skills`. opencode natively scans **both** conventions plus its
  own `~/.config/opencode/skills/` — the owner's hunch was right: opencode
  is covered by the Claude location alone.
- Minimal install set for all three skill-capable providers:
  `~/.claude/skills/` (covers Claude + opencode) plus `~/.agents/skills/`
  (covers Codex). So "installed" is per *install location*, not strictly per
  provider — two locations flip three providers. Feeds ticket 04's grain
  question directly.
- Unverified edge: opencode's precedence when the same skill name exists in
  several of its roots is undocumented.
