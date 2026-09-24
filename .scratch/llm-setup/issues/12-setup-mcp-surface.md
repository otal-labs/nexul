# 12 — The setup MCP surface

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Name and shape the MCP tools the setup turn calls, one per use case
(`<verb>_<object>`), given the decisions in tickets 04 and 05:

- Read: the computer's overall state and per-driver states.
- Confirm / un-confirm a driver on a computer, taking the computer's id and
  recording the verified skills list the harness reported.
- Confirm / un-confirm the overall setup.
- How the exempt setup turn is marked so the block recognises it, and that
  nothing but the setup turn can claim the exemption.
- Which permission (`<domain>:<action>`) gates each tool — likely pairing's
  existing vocabulary, since the state is the user's own computer.
- The events each write publishes (catalog row plus outbox write).

Mostly engineering detail; bring the owner only the calls that are real
trade-offs.

## Answer

Grilled with the owner 2026-09-24. The key finding: every agent turn assumes
the harness already has Nexul's MCP server connected ("call the whoami tool
first; if it fails, stop"), and connecting it today is a manual,
per-tool job (mint a token, follow the MCP guide per client). On anyone
else's machine that is where plays stop — alongside the missing skills.

- **Naming is `<object>_<verb>`**, matching nearly every existing tool;
  `AGENTS.md` and `practices/architecture.md` said verb first and are fixed
  in the same change. About a dozen verb-first stragglers remain
  (`search_docs`, `search_tickets`, `list_dead_letters`,
  `replay_dead_letter`, the three invitation tools, the five account tools)
  — rename them when this map is sliced.
- **Connecting MCP is the wizard's first step** (amends the setup-flow
  ticket): the wizard mints a dedicated personal access token per computer,
  "Nexul MCP on <computer>", listed and revocable on the computer's row; the
  setup turn writes Nexul's MCP entry into each provider's own config
  (Claude's MCP list, Codex's `config.toml`, opencode's config, and so on);
  then **one short setup turn per provider** confirms itself through MCP, so
  each confirmation proves that provider reaches Nexul and sees its skills.
- **The token is hidden in the saved setup transcript**, the rest kept for
  debugging; un-confirming or unpairing the computer revokes the token.
- **The nexul-memory skill is installed by the wizard** alongside the
  default set, replacing the copy-paste box in settings (amends the
  skill-sets ticket's "no Nexul skill").
- **New tools**: `computer_setup_get` (a computer's overall and per-driver
  state); `computer_setup_confirm_provider` / `computer_setup_unconfirm_provider`
  (computer id, driver kind, the skills list the harness reported);
  `computer_setup_confirm` / `computer_setup_unconfirm` (overall);
  `account_whoami` (the tool the agent prompt already tells agents to call,
  which does not exist yet); `git_get_change_context` (commit or PR number →
  PR, tickets, docs, bugs found after done, decisions-log entries). Exact
  names may shift at implementation, the object-first rule may not.
- **Owner-only**: every setup tool checks the computer belongs to the caller;
  no new permission. **Events**: confirm and un-confirm publish (catalog row
  plus outbox write) with live push.
- **The setup turn's exemption comes from its code path**: only the wizard's
  own server use-case starts one, and it resolves its target without the
  setup guard. No MCP argument or marker can request the exemption.
- **The decisions check** is a server-side consumer of a card entering a
  done column; it writes through the existing memory tools and needs no new
  tool.
