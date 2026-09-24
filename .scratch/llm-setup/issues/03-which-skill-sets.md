# 03 — Which skill sets "setup" installs

**Type:** grilling
**Status:** resolved
**Blocked by:** 01, 02

## Question

What exactly must be on a new user's machine before their setup counts as
done?

- Is mattpocock/skills the required baseline (the loop's `/wayfinder`,
  `/grill-with-docs`, `/to-tickets` come from it)?
- Is pstack required, recommended, or merely mentioned? (The owner likes it;
  ticket 02 says what it actually is.)
- Does Nexul ship anything of its own — e.g. a small skill that teaches the
  agent the Nexul MCP tools and the docs→tickets→deploy loop — or do the two
  third-party sets cover it?
- Version story: is "installed" a one-time fact, or does a skill-set update
  ever invalidate it?

Resolve with the owner via /grilling and /domain-modeling; the answer defines
what the per-provider boolean asserts (ticket 04) and what the install steps
are (ticket 05).

## Answer

Grilled with the owner 2026-09-22.

- **mattpocock/skills is the default baseline** the setup installs. Users can
  edit or amend their installed copies to shape their plays; the default is
  what the wizard brings in.
- **pstack is a README credit, not part of setup** (ticket 07). Its 23
  one-line principle skills and `architect`/`interrogate` are noted as
  complement candidates for a later optional pre-selection — they need
  curation (its `tdd` and `teach` collide by name with Matt's), so they never
  gate onboarding.
- **No Nexul-shipped skill for now.** The teaching surface for the setup is
  Nexul's own step-by-step wizard (ticket 05) — the same family as the
  instance-setup and Add-runner wizards — not a skill; Matt's own setup step
  can run inside it. The MCP tool descriptions and workflow prompts carry the
  loop.
- **"Setup done" = the wizard ran to completion on that computer**: the
  default set present in both required install locations (`~/.claude/skills/`
  for Claude Code + opencode, `~/.agents/skills/` for Codex), confirmed
  through the paired machine's own harness via MCP.
- **Nothing ever auto-flips a boolean back to false.** A new mattpocock
  release never invalidates (the user can ask an agent to update, the boolean
  stays); re-pairing and daily nightly updates never invalidate; the state is
  per computer, so a different computer simply starts at false and needs its
  own setup. A newly appearing provider starts at false on its own boolean.
- **The booleans are enforcing, not informational**: a provider on a computer
  cannot be used for agent runs until its boolean is true. Recorded on
  ticket 06, which designs the surfaces around that block.

## Comments

Amended 2026-09-24: a Nexul skill already exists — **nexul-memory**, the
shared-memory protocol, pasted in by hand from settings today. The wizard
installs it alongside the default set and the copy-paste box goes. No other
Nexul skill is planned.
