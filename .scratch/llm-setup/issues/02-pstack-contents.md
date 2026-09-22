# 02 — What cursor/plugins pstack is

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

What does https://github.com/cursor/plugins/tree/main/pstack actually
contain, and can it sit beside mattpocock/skills in a Nexul user's setup?

- What skills/plugins it ships and what each is for.
- Its format: is it the same SKILL.md convention Claude Code reads, or a
  Cursor-specific plugin format needing translation?
- How it installs (CLI, copy, plugin manager) and where it lands on disk.
- License and any attribution constraints.
- Overlap or conflicts with the mattpocock set (duplicate trigger phrases,
  competing lifecycle skills).

Feeds ticket 03 (which sets does setup install) and ticket 07 (the README
mention needs an accurate one-line description).

## Answer

Full findings: [research/pstack-contents.md](../research/pstack-contents.md)

- pstack is Lauren Tan's engineering-discipline pack in Cursor's official
  `cursor/plugins` marketplace: 38 skills (23 one-line "principle" skills),
  2 subagents, and a `poteto-mode` router with 23 playbooks. MIT licensed,
  same as mattpocock/skills.
- Skill files use the standard `SKILL.md` frontmatter Claude Code and
  opencode read, so the content is portable — but the pack is packaged only
  as a Cursor plugin (`.cursor-plugin/plugin.json`, `/add-plugin pstack`).
  There is no Claude plugin manifest and no `npx skills` entry; non-Cursor
  harnesses get it only by manually copying skill files.
- Real name collisions with mattpocock/skills: both ship a `tdd` and a
  `teach` skill (the two `teach`es do entirely different things). A flat
  merge of the two sets breaks one side; side-by-side use needs curation
  (skip or rename pstack's `tdd` and `teach`).
- Verdict for ticket 03: usable as a hand-curated second set, not as a
  required uncurated install. Credit line for ticket 07: "pstack (Lauren
  Tan) — an engineering-discipline pack of principle skills and playbooks
  in Cursor's official plugin marketplace."
